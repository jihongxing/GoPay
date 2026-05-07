package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"gopay/pkg/channel"
	"gopay/pkg/errors"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	mock2 "github.com/stretchr/testify/mock"
)

// MockRefundProvider 是支持退款的 PaymentChannel mock
type MockRefundProvider struct {
	MockPaymentChannel
}

func (m *MockRefundProvider) Refund(ctx context.Context, req *channel.RefundRequest) (*channel.RefundResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*channel.RefundResponse), args.Error(1)
}

func (m *MockRefundProvider) QueryRefund(ctx context.Context, req *channel.RefundRequest) (*channel.RefundResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*channel.RefundResponse), args.Error(1)
}

// TestRefundService_Refund 测试退款功能
func TestRefundService_Refund(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		req       *RefundRequest
		setupMock func(sqlmock.Sqlmock, *MockChannelManager, *MockRefundProvider)
		wantErr   bool
		errType   errors.ErrorType
	}{
		{
			name: "成功全额退款",
			req: &RefundRequest{
				OrderNo: "ORD_20260416_001",
				Amount:  0, // 0 表示全额退款
				Reason:  "用户申请退款",
			},
			setupMock: func(mock sqlmock.Sqlmock, cm *MockChannelManager, rp *MockRefundProvider) {
				// Mock 查询订单
				orderRows := sqlmock.NewRows([]string{
					"id", "order_no", "app_id", "out_trade_no", "channel", "amount", "currency",
					"subject", "body", "status", "notify_status", "retry_count",
					"channel_order_no", "pay_url", "paid_at", "notified_at",
					"expires_at", "created_at", "updated_at",
				}).AddRow(
					1, "ORD_20260416_001", "test_app", "BIZ_001", "wechat_native", 10000, "CNY",
					"测试商品", "", "paid", "notified", 0,
					"WX123456", "", &now, &now,
					now.Add(2*time.Hour), now, now,
				)
				mock.ExpectQuery("SELECT (.+) FROM orders WHERE order_no").
					WithArgs("ORD_20260416_001").
					WillReturnRows(orderRows)

				// Mock 渠道管理器返回支持退款的 provider
				cm.On("GetProvider", "test_app", "wechat_native").Return(rp, nil)

				// Mock 退款接口调用
				rp.On("Refund", mock2.Anything, mock2.MatchedBy(func(req *channel.RefundRequest) bool {
					return req.Amount == 10000 && req.Reason == "用户申请退款"
				})).Return(&channel.RefundResponse{
					RefundNo:         "REF_001",
					PlatformTradeNo:  "WX123456",
					PlatformRefundNo: "WX_REF_001",
					Status:           channel.RefundStatusSuccess,
					Amount:           10000,
					RefundedAt:       &now,
				}, nil)

				// Mock 保存退款交易记录
				mock.ExpectExec("INSERT INTO transactions").
					WillReturnResult(sqlmock.NewResult(1, 1))

				// Mock 更新订单状态为已退款
				mock.ExpectExec("UPDATE orders SET status").
					WithArgs("refunded", "ORD_20260416_001").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantErr: false,
		},
		{
			name: "成功部分退款",
			req: &RefundRequest{
				OrderNo: "ORD_20260416_002",
				Amount:  5000, // 部分退款
				Reason:  "部分退款",
			},
			setupMock: func(mock sqlmock.Sqlmock, cm *MockChannelManager, rp *MockRefundProvider) {
				orderRows := sqlmock.NewRows([]string{
					"id", "order_no", "app_id", "out_trade_no", "channel", "amount", "currency",
					"subject", "body", "status", "notify_status", "retry_count",
					"channel_order_no", "pay_url", "paid_at", "notified_at",
					"expires_at", "created_at", "updated_at",
				}).AddRow(
					2, "ORD_20260416_002", "test_app", "BIZ_002", "alipay_qr", 10000, "CNY",
					"测试商品", "", "paid", "notified", 0,
					"ALI123456", "", &now, &now,
					now.Add(2*time.Hour), now, now,
				)
				mock.ExpectQuery("SELECT (.+) FROM orders WHERE order_no").
					WithArgs("ORD_20260416_002").
					WillReturnRows(orderRows)

				cm.On("GetProvider", "test_app", "alipay_qr").Return(rp, nil)

				rp.On("Refund", mock2.Anything, mock2.MatchedBy(func(req *channel.RefundRequest) bool {
					return req.Amount == 5000
				})).Return(&channel.RefundResponse{
					RefundNo:         "REF_002",
					PlatformTradeNo:  "ALI123456",
					PlatformRefundNo: "ALI_REF_002",
					Status:           channel.RefundStatusSuccess,
					Amount:           5000,
					RefundedAt:       &now,
				}, nil)

				mock.ExpectExec("INSERT INTO transactions").
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectExec("UPDATE orders SET status").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantErr: false,
		},
		{
			name: "订单号为空",
			req: &RefundRequest{
				OrderNo: "",
				Amount:  1000,
			},
			setupMock: func(mock sqlmock.Sqlmock, cm *MockChannelManager, rp *MockRefundProvider) {
				// 不需要 mock，会在参数验证阶段失败
			},
			wantErr: true,
			errType: errors.TypeInvalidRequest,
		},
		{
			name: "订单不存在",
			req: &RefundRequest{
				OrderNo: "ORD_NOTEXIST",
				Amount:  1000,
			},
			setupMock: func(mock sqlmock.Sqlmock, cm *MockChannelManager, rp *MockRefundProvider) {
				mock.ExpectQuery("SELECT (.+) FROM orders WHERE order_no").
					WithArgs("ORD_NOTEXIST").
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
			errType: errors.TypeOrderNotFound,
		},
		{
			name: "订单未支付",
			req: &RefundRequest{
				OrderNo: "ORD_PENDING",
				Amount:  1000,
			},
			setupMock: func(mock sqlmock.Sqlmock, cm *MockChannelManager, rp *MockRefundProvider) {
				orderRows := sqlmock.NewRows([]string{
					"id", "order_no", "app_id", "out_trade_no", "channel", "amount", "currency",
					"subject", "body", "status", "notify_status", "retry_count",
					"channel_order_no", "pay_url", "paid_at", "notified_at",
					"expires_at", "created_at", "updated_at",
				}).AddRow(
					3, "ORD_PENDING", "test_app", "BIZ_003", "wechat_native", 10000, "CNY",
					"测试商品", "", "pending", "pending", 0,
					"", "", nil, nil,
					now.Add(2*time.Hour), now, now,
				)
				mock.ExpectQuery("SELECT (.+) FROM orders WHERE order_no").
					WithArgs("ORD_PENDING").
					WillReturnRows(orderRows)
			},
			wantErr: true,
			errType: errors.TypeInvalidRequest,
		},
		{
			name: "退款金额超过订单金额",
			req: &RefundRequest{
				OrderNo: "ORD_20260416_003",
				Amount:  20000, // 超过订单金额
			},
			setupMock: func(mock sqlmock.Sqlmock, cm *MockChannelManager, rp *MockRefundProvider) {
				orderRows := sqlmock.NewRows([]string{
					"id", "order_no", "app_id", "out_trade_no", "channel", "amount", "currency",
					"subject", "body", "status", "notify_status", "retry_count",
					"channel_order_no", "pay_url", "paid_at", "notified_at",
					"expires_at", "created_at", "updated_at",
				}).AddRow(
					4, "ORD_20260416_003", "test_app", "BIZ_003", "wechat_native", 10000, "CNY",
					"测试商品", "", "paid", "notified", 0,
					"WX123456", "", &now, &now,
					now.Add(2*time.Hour), now, now,
				)
				mock.ExpectQuery("SELECT (.+) FROM orders WHERE order_no").
					WithArgs("ORD_20260416_003").
					WillReturnRows(orderRows)
			},
			wantErr: true,
			errType: errors.TypeInvalidAmount,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建 mock DB
			db, dbMock, err := sqlmock.New()
			assert.NoError(t, err)
			defer db.Close()

			// 创建 mock 依赖
			mockCM := new(MockChannelManager)
			mockRP := new(MockRefundProvider)

			// 设置 mock 期望
			tt.setupMock(dbMock, mockCM, mockRP)

			// 创建服务
			orderService := NewOrderService(db, mockCM)
			refundService := NewRefundService(db, orderService, mockCM)

			// 执行测试
			ctx := context.Background()
			resp, err := refundService.Refund(ctx, tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, resp)
				if tt.errType != 0 {
					bizErr, ok := err.(*errors.BusinessError)
					assert.True(t, ok, "错误应该是 BusinessError 类型")
					assert.Equal(t, tt.errType, bizErr.Type)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.NotEmpty(t, resp.RefundNo)
			}

			// 验证所有期望都被满足
			assert.NoError(t, dbMock.ExpectationsWereMet())
		})
	}
}

// TestRefundService_QueryRefund 测试查询退款
func TestRefundService_QueryRefund(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		orderNo   string
		refundNo  string
		setupMock func(sqlmock.Sqlmock, *MockChannelManager, *MockRefundProvider)
		wantErr   bool
	}{
		{
			name:     "成功查询退款",
			orderNo:  "ORD_20260416_001",
			refundNo: "REF_001",
			setupMock: func(mock sqlmock.Sqlmock, cm *MockChannelManager, rp *MockRefundProvider) {
				orderRows := sqlmock.NewRows([]string{
					"id", "order_no", "app_id", "out_trade_no", "channel", "amount", "currency",
					"subject", "body", "status", "notify_status", "retry_count",
					"channel_order_no", "pay_url", "paid_at", "notified_at",
					"expires_at", "created_at", "updated_at",
				}).AddRow(
					1, "ORD_20260416_001", "test_app", "BIZ_001", "wechat_native", 10000, "CNY",
					"测试商品", "", "refunded", "notified", 0,
					"WX123456", "", &now, &now,
					now.Add(2*time.Hour), now, now,
				)
				mock.ExpectQuery("SELECT (.+) FROM orders WHERE order_no").
					WithArgs("ORD_20260416_001").
					WillReturnRows(orderRows)

				cm.On("GetProvider", "test_app", "wechat_native").Return(rp, nil)

				rp.On("QueryRefund", mock2.Anything, mock2.Anything).Return(&channel.RefundResponse{
					RefundNo:         "REF_001",
					PlatformTradeNo:  "WX123456",
					PlatformRefundNo: "WX_REF_001",
					Status:           channel.RefundStatusSuccess,
					Amount:           10000,
					RefundedAt:       &now,
				}, nil)
			},
			wantErr: false,
		},
		{
			name:     "订单号为空",
			orderNo:  "",
			refundNo: "REF_001",
			setupMock: func(mock sqlmock.Sqlmock, cm *MockChannelManager, rp *MockRefundProvider) {
				// 不需要 mock
			},
			wantErr: true,
		},
		{
			name:     "退款单号为空",
			orderNo:  "ORD_001",
			refundNo: "",
			setupMock: func(mock sqlmock.Sqlmock, cm *MockChannelManager, rp *MockRefundProvider) {
				// 不需要 mock
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, dbMock, err := sqlmock.New()
			assert.NoError(t, err)
			defer db.Close()

			mockCM := new(MockChannelManager)
			mockRP := new(MockRefundProvider)

			tt.setupMock(dbMock, mockCM, mockRP)

			orderService := NewOrderService(db, mockCM)
			refundService := NewRefundService(db, orderService, mockCM)

			ctx := context.Background()
			resp, err := refundService.QueryRefund(ctx, tt.orderNo, tt.refundNo)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, tt.refundNo, resp.RefundNo)
			}

			assert.NoError(t, dbMock.ExpectationsWereMet())
		})
	}
}

// TestRefundService_GenerateRefundNo 测试生成退款单号
func TestRefundService_GenerateRefundNo(t *testing.T) {
	service := &RefundService{}

	// 生成多个退款单号，确保唯一性
	refundNos := make(map[string]bool)
	for i := 0; i < 100; i++ {
		refundNo := service.generateRefundNo()

		assert.NotEmpty(t, refundNo, "退款单号不应该为空")
		assert.False(t, refundNos[refundNo], "退款单号应该唯一: %s", refundNo)
		refundNos[refundNo] = true

		// 检查前缀
		assert.True(t, len(refundNo) >= 4, "退款单号长度应该 >= 4")
		assert.Equal(t, "REF_", refundNo[:4], "退款单号应该以 REF_ 开头")
	}
}
