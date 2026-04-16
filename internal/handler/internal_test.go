package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gopay/internal/models"
	"gopay/internal/service"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

// TestListFailedOrders_Success 测试查询失败订单成功
func TestListFailedOrders_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	router := setupTestRouter(db)

	now := time.Now()

	// 期望查询待通知订单
	rows := sqlmock.NewRows([]string{
		"id", "order_no", "app_id", "out_trade_no", "channel", "amount",
		"currency", "subject", "body", "status", "notify_status", "retry_count",
		"channel_order_no", "pay_url", "paid_at", "notified_at", "expires_at", "created_at", "updated_at",
	}).AddRow(
		1, "ORD_001", "test_app", "OUT_001", "wechat_native", 10000,
		"CNY", "测试商品", "测试描述", models.OrderStatusPaid, models.NotifyStatusPending, 2,
		"WX_001", "", &now, nil, now.Add(2*time.Hour), now, now,
	).AddRow(
		2, "ORD_002", "test_app", "OUT_002", "alipay_qr", 20000,
		"CNY", "测试商品2", "测试描述2", models.OrderStatusPaid, models.NotifyStatusPending, 1,
		"ALI_001", "", &now, nil, now.Add(2*time.Hour), now, now,
	)

	mock.ExpectQuery("SELECT (.+) FROM orders WHERE status = \\$1 AND notify_status = \\$2 AND retry_count < 5").
		WithArgs(models.OrderStatusPaid, models.NotifyStatusPending, 100).
		WillReturnRows(rows)

	// 执行请求
	req := httptest.NewRequest(http.MethodGet, "/internal/failed-orders", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 验证响应
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp["code"])

	data := resp["data"].(map[string]any)
	assert.Equal(t, float64(2), data["total"])

	orders := data["orders"].([]any)
	assert.Len(t, orders, 2)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestListFailedOrders_Empty 测试查询失败订单为空
func TestListFailedOrders_Empty(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	router := setupTestRouter(db)

	// 期望查询返回空结果
	rows := sqlmock.NewRows([]string{
		"id", "order_no", "app_id", "out_trade_no", "channel", "amount",
		"currency", "subject", "body", "status", "notify_status", "retry_count",
		"channel_order_no", "pay_url", "paid_at", "notified_at", "expires_at", "created_at", "updated_at",
	})

	mock.ExpectQuery("SELECT (.+) FROM orders WHERE status = \\$1 AND notify_status = \\$2 AND retry_count < 5").
		WithArgs(models.OrderStatusPaid, models.NotifyStatusPending, 100).
		WillReturnRows(rows)

	// 执行请求
	req := httptest.NewRequest(http.MethodGet, "/internal/failed-orders", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 验证响应
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp["code"])

	data := resp["data"].(map[string]any)
	assert.Equal(t, float64(0), data["total"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestListFailedOrders_DatabaseError 测试数据库错误
func TestListFailedOrders_DatabaseError(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	router := setupTestRouter(db)

	// 期望查询返回错误
	mock.ExpectQuery("SELECT (.+) FROM orders WHERE status = \\$1 AND notify_status = \\$2 AND retry_count < 5").
		WithArgs(models.OrderStatusPaid, models.NotifyStatusPending, 100).
		WillReturnError(sql.ErrConnDone)

	// 执行请求
	req := httptest.NewRequest(http.MethodGet, "/internal/failed-orders", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 验证响应
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var resp map[string]any
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "INTERNAL_ERROR", resp["code"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestRetryNotify_OrderNotFound 测试订单不存在
func TestRetryNotify_OrderNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	router := setupTestRouter(db)

	// 期望查询订单返回不存在
	mock.ExpectQuery("SELECT (.+) FROM orders WHERE order_no = \\$1").
		WithArgs("NONEXISTENT").
		WillReturnError(sql.ErrNoRows)

	// 执行请求
	req := httptest.NewRequest(http.MethodPost, "/internal/retry-notify/NONEXISTENT", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 验证响应
	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp map[string]any
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "ORDER_NOT_FOUND", resp["code"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestRetryNotify_OrderNotPaid 测试订单未支付
func TestRetryNotify_OrderNotPaid(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	router := setupTestRouter(db)

	now := time.Now()

	// 期望查询订单（状态为 pending）
	rows := sqlmock.NewRows([]string{
		"id", "order_no", "app_id", "out_trade_no", "channel", "amount",
		"currency", "subject", "body", "status", "notify_status", "retry_count",
		"channel_order_no", "pay_url", "paid_at", "notified_at", "expires_at", "created_at", "updated_at",
	}).AddRow(
		1, "ORD_001", "test_app", "OUT_001", "wechat_native", 10000,
		"CNY", "测试商品", "测试描述", models.OrderStatusPending, models.NotifyStatusPending, 0,
		"", "http://pay.url", nil, nil, now.Add(2*time.Hour), now, now,
	)

	mock.ExpectQuery("SELECT (.+) FROM orders WHERE order_no = \\$1").
		WithArgs("ORD_001").
		WillReturnRows(rows)

	// 执行请求
	req := httptest.NewRequest(http.MethodPost, "/internal/retry-notify/ORD_001", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 验证响应
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]any
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "INVALID_REQUEST", resp["code"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestRetryNotify_EmptyOrderNo 测试空订单号
func TestRetryNotify_EmptyOrderNo(t *testing.T) {
	db, _, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	router := setupTestRouter(db)

	// 执行请求（空订单号）
	req := httptest.NewRequest(http.MethodPost, "/internal/retry-notify/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 验证响应 - 路由不匹配，返回 404
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestFindOrderByOutTradeNo 测试通过 out_trade_no 查找订单
func TestFindOrderByOutTradeNo(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	// 初始化 orderService
	mockChannelManager := NewMockChannelManager()
	testOrderService := service.NewOrderService(db, mockChannelManager)
	InitServices(testOrderService)

	now := time.Now()

	tests := []struct {
		name        string
		outTradeNo  string
		setupMock   func()
		expectError bool
	}{
		{
			name:       "订单存在",
			outTradeNo: "OUT_001",
			setupMock: func() {
				rows := sqlmock.NewRows([]string{
					"id", "order_no", "app_id", "out_trade_no", "channel", "amount",
					"currency", "subject", "body", "status", "notify_status", "retry_count",
					"channel_order_no", "pay_url", "paid_at", "notified_at", "expires_at", "created_at", "updated_at",
				}).AddRow(
					1, "ORD_001", "test_app", "OUT_001", "wechat_native", 10000,
					"CNY", "测试商品", "测试描述", models.OrderStatusPaid, models.NotifyStatusPending, 0,
					"WX_001", "", &now, nil, now.Add(2*time.Hour), now, now,
				)

				mock.ExpectQuery("SELECT (.+) FROM orders WHERE out_trade_no = \\$1").
					WithArgs("OUT_001").
					WillReturnRows(rows)
			},
			expectError: false,
		},
		{
			name:       "订单不存在",
			outTradeNo: "NONEXISTENT",
			setupMock: func() {
				mock.ExpectQuery("SELECT (.+) FROM orders WHERE out_trade_no = \\$1").
					WithArgs("NONEXISTENT").
					WillReturnError(sql.ErrNoRows)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			order, err := findOrderByOutTradeNo(context.Background(), tt.outTradeNo)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, order)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, order)
				assert.Equal(t, tt.outTradeNo, order.OutTradeNo)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
