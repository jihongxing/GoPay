package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"gopay/internal/models"
	"gopay/pkg/channel"
	"gopay/pkg/errors"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	mock2 "github.com/stretchr/testify/mock"
)

// MockPaymentChannel 是 PaymentChannel 的 mock
type MockPaymentChannel struct {
	mock2.Mock
}

func (m *MockPaymentChannel) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockPaymentChannel) CreateOrder(ctx context.Context, req *channel.CreateOrderRequest) (*channel.CreateOrderResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*channel.CreateOrderResponse), args.Error(1)
}

func (m *MockPaymentChannel) QueryOrder(ctx context.Context, req *channel.QueryOrderRequest) (*channel.QueryOrderResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*channel.QueryOrderResponse), args.Error(1)
}

func (m *MockPaymentChannel) HandleWebhook(ctx context.Context, req *channel.WebhookRequest) (*channel.WebhookResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*channel.WebhookResponse), args.Error(1)
}

func (m *MockPaymentChannel) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockChannelManager 是 ChannelManager 的 mock
type MockChannelManager struct {
	mock2.Mock
}

func (m *MockChannelManager) GetProvider(appID, channelName string) (channel.PaymentChannel, error) {
	args := m.Called(appID, channelName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(channel.PaymentChannel), args.Error(1)
}

// TestOrderService_CreateOrder 测试创建订单
func TestOrderService_CreateOrder(t *testing.T) {
	tests := []struct {
		name          string
		req           *CreateOrderRequest
		setupMock     func(sqlmock.Sqlmock, *MockChannelManager, *MockPaymentChannel)
		wantErr       bool
		checkResponse func(*testing.T, *CreateOrderResponse)
	}{
		{
			name: "成功创建订单",
			req: &CreateOrderRequest{
				AppID:      "test_app_id",
				OutTradeNo: "ORDER_001",
				Amount:     100,
				Subject:    "测试商品",
				Channel:    "wechat_native",
			},
			setupMock: func(mock sqlmock.Sqlmock, cm *MockChannelManager, pc *MockPaymentChannel) {
				// Mock 查询应用
				rows := sqlmock.NewRows([]string{"id", "app_id", "app_name", "app_secret", "callback_url", "status", "created_at", "updated_at"}).
					AddRow(1, "test_app_id", "测试应用", "secret", "http://callback.com", "active", time.Now(), time.Now())
				mock.ExpectQuery("SELECT (.+) FROM apps WHERE app_id").WithArgs("test_app_id").WillReturnRows(rows)

				// Mock 检查订单是否存在
				mock.ExpectQuery("SELECT (.+) FROM orders WHERE app_id").WithArgs(int64(1), "ORDER_001").WillReturnError(sql.ErrNoRows)

				// Mock 渠道管理器
				cm.On("GetProvider", "test_app_id", "wechat_native").Return(pc, nil)

				// Mock 支付渠道创建订单
				pc.On("CreateOrder", mock2.Anything, mock2.Anything).Return(&channel.CreateOrderResponse{
					PlatformTradeNo: "WX123456",
					PayURL:          "weixin://pay/123456",
					QRCode:          "https://qr.code/123456",
				}, nil)

				// Mock 保存订单
				mock.ExpectExec("INSERT INTO orders").WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantErr: false,
			checkResponse: func(t *testing.T, resp *CreateOrderResponse) {
				assert.NotEmpty(t, resp.OrderNo)
				assert.Equal(t, "weixin://pay/123456", resp.PayURL)
				assert.Equal(t, "https://qr.code/123456", resp.QRCode)
			},
		},
		{
			name: "金额为0",
			req: &CreateOrderRequest{
				AppID:      "test_app_id",
				OutTradeNo: "ORDER_002",
				Amount:     0,
				Subject:    "测试商品",
				Channel:    "wechat_native",
			},
			setupMock: func(mock sqlmock.Sqlmock, cm *MockChannelManager, pc *MockPaymentChannel) {
				// 不需要 mock，因为会在验证阶段失败
			},
			wantErr: true,
		},
		{
			name: "应用不存在",
			req: &CreateOrderRequest{
				AppID:      "invalid_app_id",
				OutTradeNo: "ORDER_003",
				Amount:     100,
				Subject:    "测试商品",
				Channel:    "wechat_native",
			},
			setupMock: func(mock sqlmock.Sqlmock, cm *MockChannelManager, pc *MockPaymentChannel) {
				// Mock 查询应用失败
				mock.ExpectQuery("SELECT (.+) FROM apps WHERE app_id").WithArgs("invalid_app_id").WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建 mock DB
			db, dbMock, err := sqlmock.New()
			assert.NoError(t, err)
			defer db.Close()

			// 创建 mock ChannelManager 和 PaymentChannel
			mockCM := new(MockChannelManager)
			mockPC := new(MockPaymentChannel)

			// 设置 mock 期望
			tt.setupMock(dbMock, mockCM, mockPC)

			// 创建 OrderService，注入 mock ChannelManager
			service := &OrderService{
				db:             db,
				channelManager: mockCM,
			}

			// 执行测试
			ctx := context.Background()
			resp, err := service.CreateOrder(ctx, tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				if tt.checkResponse != nil {
					tt.checkResponse(t, resp)
				}
			}

			// 验证所有期望都被满足
			assert.NoError(t, dbMock.ExpectationsWereMet())
		})
	}
}

// TestOrderService_QueryOrder 测试查询订单
func TestOrderService_QueryOrder(t *testing.T) {
	tests := []struct {
		name      string
		orderNo   string
		setupMock func(sqlmock.Sqlmock)
		wantErr   bool
		errType   errors.ErrorType
	}{
		{
			name:    "成功查询订单",
			orderNo: "ORD_20260416_123456",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "order_no", "app_id", "out_trade_no", "channel", "amount", "currency",
					"subject", "body", "status", "notify_status", "retry_count",
					"channel_order_no", "pay_url", "paid_at", "notified_at",
					"expires_at", "created_at", "updated_at",
				}).AddRow(
					1, "ORD_20260416_123456", "test_app_id", "ORDER_001", "wechat_native", 100, "CNY",
					"测试商品", "商品描述", "paid", "notified", 0,
					"WX123456", "weixin://pay/123456", time.Now(), time.Now(),
					time.Now().Add(2*time.Hour), time.Now(), time.Now(),
				)
				mock.ExpectQuery("SELECT (.+) FROM orders WHERE order_no").WithArgs("ORD_20260416_123456").WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name:    "订单不存在",
			orderNo: "INVALID_ORDER",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT (.+) FROM orders WHERE order_no").WithArgs("INVALID_ORDER").WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
			errType: errors.TypeOrderNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建 mock DB
			db, mock, err := sqlmock.New()
			assert.NoError(t, err)
			defer db.Close()

			// 设置 mock 期望
			tt.setupMock(mock)

			// 创建 OrderService
			service := NewOrderService(db, nil)

			// 执行测试
			ctx := context.Background()
			order, err := service.QueryOrder(ctx, tt.orderNo)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, order)
				if tt.errType != 0 {
					bizErr, ok := err.(*errors.BusinessError)
					assert.True(t, ok)
					assert.Equal(t, tt.errType, bizErr.Type)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, order)
				assert.Equal(t, tt.orderNo, order.OrderNo)
			}

			// 验证所有期望都被满足
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// TestOrderService_UpdateOrderStatus 测试更新订单状态
func TestOrderService_UpdateOrderStatus(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name       string
		orderNo    string
		status     string
		paidAt     *time.Time
		paidAmount int64
		setupMock  func(sqlmock.Sqlmock)
		wantErr    bool
		errType    errors.ErrorType
	}{
		{
			name:       "成功更新为已支付",
			orderNo:    "ORD_20260416_001",
			status:     models.OrderStatusPaid,
			paidAt:     &now,
			paidAmount: 10000,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				rows := sqlmock.NewRows([]string{"status"}).AddRow("pending")
				mock.ExpectQuery("SELECT status FROM orders WHERE order_no").WithArgs("ORD_20260416_001").WillReturnRows(rows)
				mock.ExpectExec("UPDATE orders SET status").WithArgs(models.OrderStatusPaid, &now, "ORD_20260416_001").WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			wantErr: false,
		},
		{
			name:       "订单不存在",
			orderNo:    "ORD_NOTEXIST_001",
			status:     models.OrderStatusPaid,
			paidAt:     &now,
			paidAmount: 10000,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery("SELECT status FROM orders WHERE order_no").WithArgs("ORD_NOTEXIST_001").WillReturnError(sql.ErrNoRows)
				mock.ExpectRollback()
			},
			wantErr: true,
			errType: errors.TypeOrderNotFound,
		},
		{
			name:       "订单已支付",
			orderNo:    "ORD_PAID_001",
			status:     models.OrderStatusPaid,
			paidAt:     &now,
			paidAmount: 10000,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				rows := sqlmock.NewRows([]string{"status"}).AddRow("paid")
				mock.ExpectQuery("SELECT status FROM orders WHERE order_no").WithArgs("ORD_PAID_001").WillReturnRows(rows)
				mock.ExpectRollback()
			},
			wantErr: true,
			errType: errors.TypeOrderPaid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建 mock DB
			db, mock, err := sqlmock.New()
			assert.NoError(t, err)
			defer db.Close()

			// 设置 mock 期望
			tt.setupMock(mock)

			// 创建 OrderService
			service := NewOrderService(db, nil)

			// 执行测试
			ctx := context.Background()
			err = service.UpdateOrderStatus(ctx, tt.orderNo, tt.status, tt.paidAt, tt.paidAmount)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != 0 {
					bizErr, ok := err.(*errors.BusinessError)
					assert.True(t, ok)
					assert.Equal(t, tt.errType, bizErr.Type)
				}
			} else {
				assert.NoError(t, err)
			}

			// 验证所有期望都被满足
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// TestOrderService_GenerateOrderNo 测试生成订单号
func TestOrderService_GenerateOrderNo(t *testing.T) {
	service := &OrderService{}

	// 生成多个订单号，确保唯一性
	orderNos := make(map[string]bool)
	for i := 0; i < 100; i++ {
		orderNo := service.generateOrderNo()

		// 检查格式
		assert.NotEmpty(t, orderNo, "订单号不应该为空")

		// 检查唯一性
		assert.False(t, orderNos[orderNo], "订单号应该唯一: %s", orderNo)
		orderNos[orderNo] = true

		// 检查前缀
		assert.True(t, len(orderNo) >= 4, "订单号长度应该 >= 4")
		assert.Equal(t, "ORD_", orderNo[:4], "订单号应该以 ORD_ 开头")
	}
}

// TestOrderService_BuildWebhookURL 测试构建 Webhook URL
func TestOrderService_BuildWebhookURL(t *testing.T) {
	service := &OrderService{}

	tests := []struct {
		name    string
		channel string
		want    string
	}{
		{
			name:    "wechat channel",
			channel: "wechat",
			want:    "http://localhost:8080/api/v1/webhook/wechat",
		},
		{
			name:    "alipay channel",
			channel: "alipay",
			want:    "http://localhost:8080/api/v1/webhook/alipay",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := service.buildWebhookURL(tt.channel)
			assert.Equal(t, tt.want, got)
		})
	}
}

