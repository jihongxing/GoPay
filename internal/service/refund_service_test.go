package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"gopay/pkg/channel"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	mock2 "github.com/stretchr/testify/mock"
)

// MockRefundableChannel implements PaymentChannel + refundProvider
type MockRefundableChannel struct {
	mock2.Mock
}

func (m *MockRefundableChannel) Name() string { return "mock_refund" }
func (m *MockRefundableChannel) Close() error { return nil }
func (m *MockRefundableChannel) CreateOrder(ctx context.Context, req *channel.CreateOrderRequest) (*channel.CreateOrderResponse, error) {
	return nil, nil
}
func (m *MockRefundableChannel) QueryOrder(ctx context.Context, req *channel.QueryOrderRequest) (*channel.QueryOrderResponse, error) {
	return nil, nil
}
func (m *MockRefundableChannel) HandleWebhook(ctx context.Context, req *channel.WebhookRequest) (*channel.WebhookResponse, error) {
	return nil, nil
}
func (m *MockRefundableChannel) Refund(ctx context.Context, req *channel.RefundRequest) (*channel.RefundResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*channel.RefundResponse), args.Error(1)
}
func (m *MockRefundableChannel) QueryRefund(ctx context.Context, req *channel.RefundRequest) (*channel.RefundResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*channel.RefundResponse), args.Error(1)
}

func newRefundService(db *sql.DB, cm ChannelManagerInterface) *RefundService {
	os := NewOrderService(db, cm)
	return NewRefundService(db, os, cm)
}

var orderCols = []string{
	"id", "order_no", "app_id", "out_trade_no", "channel", "amount", "currency",
	"subject", "body", "status", "notify_status", "retry_count",
	"channel_order_no", "pay_url", "paid_at", "notified_at",
	"expires_at", "created_at", "updated_at",
}

func addPaidOrderRow(rows *sqlmock.Rows) *sqlmock.Rows {
	now := time.Now()
	return rows.AddRow(1, "ORD_001", "app1", "BIZ_001", "wechat_native", 10000, "CNY",
		"商品", "", "paid", "notified", 0, "WX001", "", &now, &now,
		now.Add(2*time.Hour), now, now)
}

func TestRefundService_Refund_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mockCM := new(MockChannelManager)
	mockPC := new(MockRefundableChannel)

	// Mock QueryOrder
	orderRows := sqlmock.NewRows(orderCols)
	addPaidOrderRow(orderRows)
	mock.ExpectQuery("SELECT (.+) FROM orders WHERE order_no").WithArgs("ORD_001").WillReturnRows(orderRows)

	mockCM.On("GetProvider", "app1", "wechat_native").Return(mockPC, nil)
	mockPC.On("Refund", mock2.Anything, mock2.Anything).Return(&channel.RefundResponse{
		RefundNo:         "RFD_001",
		PlatformTradeNo:  "WX001",
		PlatformRefundNo: "WXRFD_001",
		Status:           channel.RefundStatusSuccess,
		Amount:           10000,
	}, nil)

	// Mock saveRefundTransaction
	mock.ExpectExec("INSERT INTO transactions").WillReturnResult(sqlmock.NewResult(1, 1))
	// Mock updateRefundStatus (status == SUCCESS)
	mock.ExpectExec("UPDATE orders").WillReturnResult(sqlmock.NewResult(1, 1))

	svc := newRefundService(db, mockCM)
	resp, err := svc.Refund(context.Background(), &RefundRequest{OrderNo: "ORD_001", Amount: 10000})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "ORD_001", resp.OrderNo)
	assert.Equal(t, int64(10000), resp.Amount)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRefundService_Refund_OrderNotPaid(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	now := time.Now()
	orderRows := sqlmock.NewRows(orderCols).
		AddRow(1, "ORD_001", "app1", "BIZ_001", "wechat_native", 10000, "CNY",
			"商品", "", "pending", "pending", 0, "", "", nil, nil,
			now.Add(2*time.Hour), now, now)
	mock.ExpectQuery("SELECT (.+) FROM orders WHERE order_no").WithArgs("ORD_001").WillReturnRows(orderRows)

	svc := newRefundService(db, new(MockChannelManager))
	resp, err := svc.Refund(context.Background(), &RefundRequest{OrderNo: "ORD_001"})

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "已支付")
}

func TestRefundService_Refund_AmountExceedsOrder(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	orderRows := sqlmock.NewRows(orderCols)
	addPaidOrderRow(orderRows)
	mock.ExpectQuery("SELECT (.+) FROM orders WHERE order_no").WithArgs("ORD_001").WillReturnRows(orderRows)

	svc := newRefundService(db, new(MockChannelManager))
	resp, err := svc.Refund(context.Background(), &RefundRequest{OrderNo: "ORD_001", Amount: 99999})

	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestRefundService_Refund_NilRequest(t *testing.T) {
	db, _, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	svc := newRefundService(db, new(MockChannelManager))
	resp, err := svc.Refund(context.Background(), nil)
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestRefundService_Refund_EmptyOrderNo(t *testing.T) {
	db, _, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	svc := newRefundService(db, new(MockChannelManager))
	resp, err := svc.Refund(context.Background(), &RefundRequest{OrderNo: ""})
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestRefundService_Refund_NotInitialized(t *testing.T) {
	svc := &RefundService{}
	resp, err := svc.Refund(context.Background(), &RefundRequest{OrderNo: "ORD_001"})
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "not initialized")
}

func TestRefundService_QueryRefund_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mockCM := new(MockChannelManager)
	mockPC := new(MockRefundableChannel)

	now := time.Now()
	orderRows := sqlmock.NewRows(orderCols).
		AddRow(1, "ORD_001", "app1", "BIZ_001", "wechat_native", 10000, "CNY",
			"商品", "", "refunded", "notified", 0, "WX001", "", &now, &now,
			now.Add(2*time.Hour), now, now)
	mock.ExpectQuery("SELECT (.+) FROM orders WHERE order_no").WithArgs("ORD_001").WillReturnRows(orderRows)

	mockCM.On("GetProvider", "app1", "wechat_native").Return(mockPC, nil)
	mockPC.On("QueryRefund", mock2.Anything, mock2.Anything).Return(&channel.RefundResponse{
		RefundNo:         "RFD_001",
		PlatformTradeNo:  "WX001",
		PlatformRefundNo: "WXRFD_001",
		Status:           channel.RefundStatusSuccess,
		Amount:           10000,
	}, nil)

	svc := newRefundService(db, mockCM)
	resp, err := svc.QueryRefund(context.Background(), "ORD_001", "RFD_001")

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "RFD_001", resp.RefundNo)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRefundService_QueryRefund_EmptyParams(t *testing.T) {
	db, _, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	svc := newRefundService(db, new(MockChannelManager))

	_, err = svc.QueryRefund(context.Background(), "", "RFD_001")
	assert.Error(t, err)

	_, err = svc.QueryRefund(context.Background(), "ORD_001", "")
	assert.Error(t, err)
}

func TestRefundService_QueryRefund_NotInitialized(t *testing.T) {
	svc := &RefundService{}
	resp, err := svc.QueryRefund(context.Background(), "ORD_001", "RFD_001")
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestRefundService_ChannelNotSupportRefund(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mockCM := new(MockChannelManager)
	// Use MockPaymentChannel which does NOT implement refundProvider
	mockPC := new(MockPaymentChannel)

	orderRows := sqlmock.NewRows(orderCols)
	addPaidOrderRow(orderRows)
	mock.ExpectQuery("SELECT (.+) FROM orders WHERE order_no").WithArgs("ORD_001").WillReturnRows(orderRows)

	mockCM.On("GetProvider", "app1", "wechat_native").Return(mockPC, nil)

	svc := newRefundService(db, mockCM)
	resp, err := svc.Refund(context.Background(), &RefundRequest{OrderNo: "ORD_001", Amount: 10000})

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "does not support refund")
}
