package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"gopay/internal/models"
	"gopay/internal/service"
)

func setupHandlerBranchEnv(t *testing.T, db *sql.DB) {
	t.Helper()

	mockChannelManager := NewMockChannelManager()
	orderSvc := service.NewOrderService(db, mockChannelManager)
	notifySvc := service.NewNotifyService(db, orderSvc)

	InitServices(orderSvc)
	InitWebhookServices(mockChannelManager, notifySvc)

	t.Cleanup(func() {
		InitServices(nil)
		InitWebhookServices(nil, nil)
	})
}

func newHandlerOrderRows(orderNo, appID, outTradeNo, status string) *sqlmock.Rows {
	now := time.Now()
	paidAt := now.Add(-1 * time.Hour)

	return sqlmock.NewRows([]string{
		"id", "order_no", "app_id", "out_trade_no", "channel", "amount", "currency",
		"subject", "body", "status", "notify_status", "retry_count",
		"channel_order_no", "pay_url", "paid_at", "notified_at",
		"expires_at", "created_at", "updated_at",
	}).AddRow(
		1, orderNo, appID, outTradeNo, models.ChannelWechatNative, 10000, "CNY",
		"测试商品", "测试描述", status, models.NotifyStatusPending, 0,
		"PLAT_001", "http://pay.url", &paidAt, nil,
		now.Add(2*time.Hour), now, now,
	)
}

func TestQueryOrder_EmptyOrderNo(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, _, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	setupHandlerBranchEnv(t, db)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)

	QueryOrder(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]any
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "INVALID_REQUEST", resp["code"])
}

func TestQueryOrder_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	setupHandlerBranchEnv(t, db)

	mock.ExpectQuery("SELECT (.+) FROM orders WHERE order_no = \\$1").
		WithArgs("ORD_404").
		WillReturnError(sql.ErrNoRows)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/orders/ORD_404", nil)
	c.Params = gin.Params{{Key: "order_no", Value: "ORD_404"}}

	QueryOrder(c)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp map[string]any
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "ORDER_NOT_FOUND", resp["code"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRetryNotify_EmptyOrderNo_Direct(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, _, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	setupHandlerBranchEnv(t, db)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/internal/retry-notify", nil)

	RetryNotify(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]any
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "INVALID_REQUEST", resp["code"])
}

func TestRetryNotify_InvalidStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	setupHandlerBranchEnv(t, db)

	mock.ExpectQuery("SELECT (.+) FROM orders WHERE order_no = \\$1").
		WithArgs("ORD_001").
		WillReturnRows(newHandlerOrderRows("ORD_001", "test_app", "OUT_001", models.OrderStatusPending))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/internal/retry-notify/ORD_001", nil)
	c.Params = gin.Params{{Key: "order_no", Value: "ORD_001"}}

	RetryNotify(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]any
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "INVALID_REQUEST", resp["code"])

	assert.NoError(t, mock.ExpectationsWereMet())
}
