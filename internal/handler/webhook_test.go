package handler

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"gopay/internal/models"
	"gopay/internal/service"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

// setupTestOrderService 创建测试用的 OrderService
func setupTestOrderService(db *sql.DB) *service.OrderService {
	mockChannelManager := NewMockChannelManager()
	return service.NewOrderService(db, mockChannelManager)
}

// TestWebhook_OrderQuery 测试订单查询逻辑
func TestWebhook_OrderQuery(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

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
				orderRows := sqlmock.NewRows([]string{
					"id", "order_no", "app_id", "out_trade_no", "channel", "amount", "currency",
					"subject", "body", "status", "notify_status", "retry_count",
					"channel_order_no", "pay_url", "paid_at", "notified_at",
					"expires_at", "created_at", "updated_at",
				}).AddRow(
					1, "ORD_001", "test_app", "OUT_001", "wechat_native", 10000, "CNY",
					"测试商品", "测试描述", models.OrderStatusPending, models.NotifyStatusPending, 0,
					"", "http://pay.url", nil, nil,
					now.Add(2*time.Hour), now, now,
				)

				mock.ExpectQuery("SELECT (.+) FROM orders WHERE out_trade_no = \\$1").
					WithArgs("OUT_001").
					WillReturnRows(orderRows)
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

			orderService := setupTestOrderService(db)
			order, err := orderService.QueryOrderByOutTradeNoGlobal(context.Background(), tt.outTradeNo)

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

// TestWebhook_OrderStatusUpdate 测试订单状态更新逻辑
func TestWebhook_OrderStatusUpdate(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	now := time.Now()

	tests := []struct {
		name          string
		orderNo       string
		currentStatus string
		newStatus     string
		setupMock     func()
		expectError   bool
	}{
		{
			name:          "待支付订单更新为已支付",
			orderNo:       "ORD_001",
			currentStatus: models.OrderStatusPending,
			newStatus:     models.OrderStatusPaid,
			setupMock: func() {
				mock.ExpectBegin()
				mock.ExpectQuery("SELECT status FROM orders WHERE order_no = \\$1 FOR UPDATE").
					WithArgs("ORD_001").
					WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow(models.OrderStatusPending))
				mock.ExpectExec("UPDATE orders SET status = \\$1, paid_at = \\$2, updated_at = NOW\\(\\) WHERE order_no = \\$3").
					WithArgs(models.OrderStatusPaid, sqlmock.AnyArg(), "ORD_001").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectError: false,
		},
		{
			name:          "已支付订单重复更新",
			orderNo:       "ORD_002",
			currentStatus: models.OrderStatusPaid,
			newStatus:     models.OrderStatusPaid,
			setupMock: func() {
				mock.ExpectBegin()
				mock.ExpectQuery("SELECT status FROM orders WHERE order_no = \\$1 FOR UPDATE").
					WithArgs("ORD_002").
					WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow(models.OrderStatusPaid))
				mock.ExpectRollback()
			},
			expectError: true, // 应该返回 OrderPaidError
		},
		{
			name:          "订单不存在",
			orderNo:       "NONEXISTENT",
			currentStatus: "",
			newStatus:     models.OrderStatusPaid,
			setupMock: func() {
				mock.ExpectBegin()
				mock.ExpectQuery("SELECT status FROM orders WHERE order_no = \\$1 FOR UPDATE").
					WithArgs("NONEXISTENT").
					WillReturnError(sql.ErrNoRows)
				mock.ExpectRollback()
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			orderService := setupTestOrderService(db)
			paidAt := now
			err := orderService.UpdateOrderStatus(context.Background(), tt.orderNo, tt.newStatus, &paidAt, 10000)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// TestWebhook_IdempotentHandling 测试幂等性处理
func TestWebhook_IdempotentHandling(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	now := time.Now()
	paidAt := now.Add(-1 * time.Hour)

	// 查询已支付订单
	orderRows := sqlmock.NewRows([]string{
		"id", "order_no", "app_id", "out_trade_no", "channel", "amount", "currency",
		"subject", "body", "status", "notify_status", "retry_count",
		"channel_order_no", "pay_url", "paid_at", "notified_at",
		"expires_at", "created_at", "updated_at",
	}).AddRow(
		1, "ORD_001", "test_app", "OUT_001", "wechat_native", 10000, "CNY",
		"测试商品", "测试描述", models.OrderStatusPaid, models.NotifyStatusNotified, 0,
		"wx_trade_001", "http://pay.url", &paidAt, &now,
		now.Add(2*time.Hour), now, now,
	)

	mock.ExpectQuery("SELECT (.+) FROM orders WHERE out_trade_no = \\$1").
		WithArgs("OUT_001").
		WillReturnRows(orderRows)

	// 尝试更新已支付订单
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT status FROM orders WHERE order_no = \\$1 FOR UPDATE").
		WithArgs("ORD_001").
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow(models.OrderStatusPaid))
	mock.ExpectRollback()

	orderService := setupTestOrderService(db)

	// 1. 查询订单
	order, err := orderService.QueryOrderByOutTradeNoGlobal(context.Background(), "OUT_001")
	assert.NoError(t, err)
	assert.Equal(t, models.OrderStatusPaid, order.Status)

	// 2. 尝试更新（应该被拒绝）
	err = orderService.UpdateOrderStatus(context.Background(), "ORD_001", models.OrderStatusPaid, &now, 10000)
	assert.Error(t, err) // 应该返回 OrderPaidError

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestWebhook_ConcurrentUpdate 测试并发更新（行锁）
func TestWebhook_ConcurrentUpdate(t *testing.T) {
	t.Skip("需要真实数据库测试行锁行为")
	// 这个测试需要真实数据库来验证 FOR UPDATE 的行锁机制
	// 在单元测试中使用 sqlmock 无法模拟真实的并发场景
}

// 注意：以下测试需要真实的支付渠道配置，跳过
func TestWechatWebhook_Integration(t *testing.T) {
	t.Skip("需要真实的微信支付配置和私钥文件")
	// 集成测试应该在独立的测试环境中运行
	// 需要：
	// 1. 真实的微信支付配置
	// 2. 有效的平台证书
	// 3. 测试数据库
}

func TestAlipayWebhook_Integration(t *testing.T) {
	t.Skip("需要真实的支付宝配置和私钥文件")
	// 集成测试应该在独立的测试环境中运行
	// 需要：
	// 1. 真实的支付宝配置
	// 2. 有效的应用私钥和支付宝公钥
	// 3. 测试数据库
}
