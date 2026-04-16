package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gopay/internal/service"
	"gopay/pkg/channel"
)

// MockChannelManager 用于测试的 Mock 渠道管理器
type MockChannelManager struct {
	providers map[string]channel.PaymentChannel
}

func NewMockChannelManager() *MockChannelManager {
	return &MockChannelManager{
		providers: make(map[string]channel.PaymentChannel),
	}
}

func (m *MockChannelManager) GetProvider(appID, channelName string) (channel.PaymentChannel, error) {
	key := appID + "_" + channelName
	if provider, ok := m.providers[key]; ok {
		return provider, nil
	}
	// 返回默认的 Mock Provider
	return &MockPaymentChannel{}, nil
}

func (m *MockChannelManager) SetProvider(appID, channelName string, provider channel.PaymentChannel) {
	key := appID + "_" + channelName
	m.providers[key] = provider
}

// setupTestRouter 设置测试路由
func setupTestRouter(db *sql.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 使用 Mock 渠道管理器
	mockChannelManager := NewMockChannelManager()
	orderService := service.NewOrderService(db, mockChannelManager)
	notifyService := service.NewNotifyService(db, orderService)

	InitServices(orderService)

	// 初始化 webhook 服务（使用 mock ChannelManager）
	channelMgr := service.NewChannelManager(db)
	InitWebhookServices(channelMgr, notifyService)

	// 注册路由
	router.POST("/api/v1/checkout", Checkout)
	router.GET("/api/v1/orders/:order_no", QueryOrder)
	router.GET("/internal/failed-orders", ListFailedOrders)
	router.POST("/internal/retry-notify/:order_no", RetryNotify)

	return router
}

func TestCheckout_Success(t *testing.T) {
	// 创建 mock 数据库
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	// 设置期望的数据库操作

	// 1. 期望查询 app 配置
	appRows := sqlmock.NewRows([]string{
		"id", "app_id", "app_name", "app_secret", "callback_url", "status", "created_at", "updated_at",
	}).AddRow(
		1, "test_app", "测试应用", "secret123", "http://example.com/callback", "active", time.Now(), time.Now(),
	)
	mock.ExpectQuery("SELECT (.+) FROM apps WHERE app_id = \\$1 AND status = 'active'").
		WithArgs("test_app").
		WillReturnRows(appRows)

	// 2. 期望检查订单是否存在
	mock.ExpectQuery("SELECT (.+) FROM orders WHERE (.+)").
		WithArgs(sqlmock.AnyArg(), "OUT_001").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	// 3. 期望插入订单（不在事务中）
	mock.ExpectExec("INSERT INTO orders").
		WillReturnResult(sqlmock.NewResult(1, 1))

	// 设置路由
	router := setupTestRouter(db)

	// 构建请求
	reqBody := CheckoutRequest{
		AppID:      "test_app",
		OutTradeNo: "OUT_001",
		Amount:     10000,
		Subject:    "测试商品",
		Body:       "测试商品描述",
		Channel:    "wechat_native",
		NotifyURL:  "http://example.com/notify",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/checkout", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// 执行请求
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 验证响应
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp["code"])
	assert.NotNil(t, resp["data"])

	// 验证所有期望的数据库操作都被执行
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckout_InvalidRequest(t *testing.T) {
	db, _, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	router := setupTestRouter(db)

	tests := []struct {
		name       string
		reqBody    any
		wantStatus int
	}{
		{
			name: "缺少 app_id",
			reqBody: map[string]any{
				"out_trade_no": "OUT_001",
				"amount":       10000,
				"subject":      "测试商品",
				"channel":      "wechat_native",
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "金额为负数",
			reqBody: CheckoutRequest{
				AppID:      "test_app",
				OutTradeNo: "OUT_001",
				Amount:     -100,
				Subject:    "测试商品",
				Channel:    "wechat_native",
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "缺少 subject",
			reqBody: CheckoutRequest{
				AppID:      "test_app",
				OutTradeNo: "OUT_001",
				Amount:     10000,
				Channel:    "wechat_native",
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.reqBody)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/checkout", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestQueryOrder_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	router := setupTestRouter(db)

	// 设置期望的查询
	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "order_no", "app_id", "biz_order_no", "channel", "amount",
		"currency", "subject", "body", "status", "notify_status", "retry_count",
		"channel_order_no", "pay_url", "paid_at", "notified_at", "expires_at", "created_at", "updated_at",
	}).AddRow(
		1, "ORD_001", "test_app", "OUT_001", "wechat_native", 10000,
		"CNY", "测试商品", "测试描述", "pending", "pending", 0,
		"", "", nil, nil, now.Add(2*time.Hour), now, now,
	)

	mock.ExpectQuery("SELECT (.+) FROM orders WHERE order_no = \\$1").
		WithArgs("ORD_001").
		WillReturnRows(rows)

	// 执行请求
	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders/ORD_001", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 验证响应
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp["code"])

	data := resp["data"].(map[string]any)
	assert.Equal(t, "ORD_001", data["order_no"])
	assert.Equal(t, "test_app", data["app_id"])
	assert.Equal(t, float64(10000), data["amount"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestQueryOrder_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	router := setupTestRouter(db)

	// 设置期望的查询（返回空结果）
	mock.ExpectQuery("SELECT (.+) FROM orders WHERE order_no = \\$1").
		WithArgs("NONEXISTENT").
		WillReturnError(sql.ErrNoRows)

	// 执行请求
	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders/NONEXISTENT", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 验证响应
	assert.Equal(t, http.StatusNotFound, w.Code)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListFailedOrders(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	router := setupTestRouter(db)

	// 设置期望的查询（注意：查询的是 status=paid, notify_status=pending）
	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"id", "order_no", "app_id", "out_trade_no", "channel", "amount",
		"currency", "subject", "body", "status", "notify_status", "retry_count",
		"channel_order_no", "pay_url", "paid_at", "notified_at", "expires_at", "created_at", "updated_at",
	}).AddRow(
		1, "ORD_001", "test_app", "OUT_001", "wechat_native", 10000,
		"CNY", "测试商品", "测试描述", "paid", "pending", 3,
		"WX_001", "", &now, nil, now.Add(2*time.Hour), now, now,
	)

	mock.ExpectQuery("SELECT (.+) FROM orders WHERE status = \\$1 AND notify_status = \\$2 AND retry_count < 5").
		WithArgs("paid", "pending", 100).
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
	assert.Equal(t, float64(1), data["total"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRetryNotify(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	router := setupTestRouter(db)

	// 设置期望的查询和更新
	now := time.Now()

	// 1. 期望查询订单
	rows := sqlmock.NewRows([]string{
		"id", "order_no", "app_id", "out_trade_no", "channel", "amount",
		"currency", "subject", "body", "status", "notify_status", "retry_count",
		"channel_order_no", "pay_url", "paid_at", "notified_at", "expires_at", "created_at", "updated_at",
	}).AddRow(
		1, "ORD_001", "test_app", "OUT_001", "wechat_native", 10000,
		"CNY", "测试商品", "测试描述", "paid", "failed", 3,
		"WX_001", "", &now, nil, now.Add(2*time.Hour), now, now,
	)

	mock.ExpectQuery("SELECT (.+) FROM orders WHERE order_no = \\$1").
		WithArgs("ORD_001").
		WillReturnRows(rows)

	// 2. 期望查询应用信息
	appRows := sqlmock.NewRows([]string{
		"id", "app_id", "app_name", "app_secret", "callback_url", "status", "created_at", "updated_at",
	}).AddRow(
		1, "test_app", "测试应用", "secret123", "http://example.com/callback", "active", now, now,
	)

	mock.ExpectQuery("SELECT (.+) FROM apps WHERE app_id = \\$1").
		WithArgs("test_app").
		WillReturnRows(appRows)

	// 3. 期望更新通知状态为 pending（准备重试）
	mock.ExpectExec("UPDATE orders SET retry_count = 0, notify_status = \\$1, updated_at = NOW\\(\\) WHERE order_no = \\$2").
		WithArgs("pending", "ORD_001").
		WillReturnResult(sqlmock.NewResult(1, 1))

	// 执行请求
	req := httptest.NewRequest(http.MethodPost, "/internal/retry-notify/ORD_001", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 验证响应
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "SUCCESS", resp["code"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

// MockPaymentChannel 用于测试的 Mock 支付渠道
type MockPaymentChannel struct {
	createOrderFunc   func(ctx context.Context, req *channel.CreateOrderRequest) (*channel.CreateOrderResponse, error)
	queryOrderFunc    func(ctx context.Context, req *channel.QueryOrderRequest) (*channel.QueryOrderResponse, error)
	handleWebhookFunc func(ctx context.Context, req *channel.WebhookRequest) (*channel.WebhookResponse, error)
}

func (m *MockPaymentChannel) Name() string {
	return "mock"
}

func (m *MockPaymentChannel) CreateOrder(ctx context.Context, req *channel.CreateOrderRequest) (*channel.CreateOrderResponse, error) {
	if m.createOrderFunc != nil {
		return m.createOrderFunc(ctx, req)
	}
	return &channel.CreateOrderResponse{
		PlatformTradeNo: "MOCK_TRADE_001",
		PayURL:          "http://mock.pay/qr",
	}, nil
}

func (m *MockPaymentChannel) QueryOrder(ctx context.Context, req *channel.QueryOrderRequest) (*channel.QueryOrderResponse, error) {
	if m.queryOrderFunc != nil {
		return m.queryOrderFunc(ctx, req)
	}
	return &channel.QueryOrderResponse{
		Status:          channel.OrderStatusPending,
		PlatformTradeNo: "MOCK_TRADE_001",
	}, nil
}

func (m *MockPaymentChannel) HandleWebhook(ctx context.Context, req *channel.WebhookRequest) (*channel.WebhookResponse, error) {
	if m.handleWebhookFunc != nil {
		return m.handleWebhookFunc(ctx, req)
	}
	return &channel.WebhookResponse{
		Success:         true,
		Status:          channel.OrderStatusPaid,
		PlatformTradeNo: "MOCK_TRADE_001",
		PaidAmount:      10000,
		PaidAt:          time.Now(),
		ResponseBody:    []byte("success"),
	}, nil
}

func (m *MockPaymentChannel) Close() error {
	return nil
}

func TestCheckout_Integration(t *testing.T) {
	// 这是一个更完整的集成测试示例
	// 实际使用时需要真实的数据库连接

	t.Skip("需要真实数据库连接")

	// 创建真实的数据库连接
	// db, err := sql.Open("postgres", "postgres://...")
	// assert.NoError(t, err)
	// defer db.Close()

	// router := setupTestRouter(db)

	// 执行完整的下单流程测试
	// 1. 创建订单
	// 2. 查询订单
	// 3. 模拟 webhook 回调
	// 4. 再次查询订单验证状态
}
