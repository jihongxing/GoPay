package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"gopay/internal/models"
)

func TestNotifyService_BuildNotifyRequest_Additional(t *testing.T) {
	now := time.Date(2026, 4, 16, 10, 30, 0, 0, time.UTC)
	svc := NewNotifyService(nil, nil)

	req := svc.buildNotifyRequest(&models.Order{
		OrderNo:        "ORD_1",
		OutTradeNo:     "OUT_1",
		Amount:         10000,
		Status:         models.OrderStatusPaid,
		PaidAt:         &now,
		Channel:        "wechat_native",
		ChannelOrderNo: "WX_1",
	})

	assert.Equal(t, "ORD_1", req.OrderNo)
	assert.Equal(t, now.Format(time.RFC3339), req.PaidAt)
	assert.Equal(t, "wechat_native", req.Channel)
	assert.Contains(t, svc.buildNotifyRequestBody(&models.Order{OrderNo: "ORD_1"}), `"order_no":"ORD_1"`)
}

func TestNotifyService_DoNotify(t *testing.T) {
	svc := NewNotifyService(nil, nil)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":"SUCCESS"}`))
	}))
	defer server.Close()

	success, statusCode, respBody, duration, err := svc.doNotify(context.Background(), &models.Order{
		OrderNo:    "ORD_1",
		OutTradeNo: "OUT_1",
		Amount:     10000,
		Status:     models.OrderStatusPaid,
		Channel:    "wechat_native",
	}, server.URL)

	assert.NoError(t, err)
	assert.True(t, success)
	assert.Equal(t, http.StatusOK, statusCode)
	assert.Contains(t, respBody, "SUCCESS")
	assert.GreaterOrEqual(t, duration, time.Duration(0))
}

func TestNotifyService_DoNotify_BadURL(t *testing.T) {
	svc := NewNotifyService(nil, nil)

	success, statusCode, _, _, err := svc.doNotify(context.Background(), &models.Order{
		OrderNo:    "ORD_1",
		OutTradeNo: "OUT_1",
		Amount:     10000,
		Status:     models.OrderStatusPaid,
		Channel:    "wechat_native",
	}, "://bad-url")

	assert.Error(t, err)
	assert.False(t, success)
	assert.Equal(t, 0, statusCode)
}

func TestNotifyService_SaveNotifyLog(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	svc := NewNotifyService(db, nil)
	mock.ExpectExec("INSERT INTO notify_logs").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = svc.saveNotifyLog(&models.NotifyLog{
		OrderNo:        "ORD_1",
		CallbackURL:    "https://callback.example.com",
		RequestBody:    "{}",
		ResponseStatus: http.StatusOK,
		ResponseBody:   "{}",
		Success:        true,
		ErrorMsg:       "",
		DurationMs:     15,
	})

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNotifyService_RetryNotify_InvalidStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	orderService := NewOrderService(db, nil)
	svc := NewNotifyService(db, orderService)

	rows := sqlmock.NewRows([]string{"id", "order_no", "app_id", "out_trade_no", "channel", "amount", "currency",
		"subject", "body", "status", "notify_status", "retry_count", "channel_order_no", "pay_url",
		"paid_at", "notified_at", "expires_at", "created_at", "updated_at"}).
		AddRow(1, "ORD_1", "app-1", "OUT_1", "wechat_native", 10000, "CNY", "subject", "body", models.OrderStatusPending,
			models.NotifyStatusPending, 0, "WX_1", "", nil, nil, time.Now(), time.Now(), time.Now())
	mock.ExpectQuery("SELECT (.+) FROM orders WHERE order_no = \\$1").
		WithArgs("ORD_1").
		WillReturnRows(rows)

	err = svc.RetryNotify(context.Background(), "ORD_1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "订单状态不正确")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNotifyService_RetryNotify_OrderNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	orderService := NewOrderService(db, nil)
	svc := NewNotifyService(db, orderService)

	mock.ExpectQuery("SELECT (.+) FROM orders WHERE order_no = \\$1").
		WithArgs("ORD_404").
		WillReturnError(sql.ErrNoRows)

	err = svc.RetryNotify(context.Background(), "ORD_404")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "order not found")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNotifyService_NotifyWithRetry_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	orderService := NewOrderService(db, nil)
	svc := NewNotifyService(db, orderService)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":"SUCCESS"}`))
	}))
	defer server.Close()
	svc.httpClient = server.Client()

	order := &models.Order{
		OrderNo:        "ORD_1",
		AppID:          "app-1",
		OutTradeNo:     "OUT_1",
		Amount:         10000,
		Status:         models.OrderStatusPaid,
		Channel:        "wechat_native",
		ChannelOrderNo: "WX_1",
	}

	mock.ExpectExec("INSERT INTO notify_logs").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("UPDATE orders\\s+SET retry_count = retry_count \\+ 1, updated_at = NOW\\(\\)\\s+WHERE order_no = \\$1").
		WithArgs("ORD_1").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("UPDATE orders\\s+SET\\s+notify_status = \\$1, notified_at = NOW\\(\\), updated_at = NOW\\(\\)\\s+WHERE order_no = \\$2").
		WithArgs(models.NotifyStatusNotified, "ORD_1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	svc.notifyWithRetry(context.Background(), order, server.URL)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNotifyService_ProcessPendingNotifies_Empty(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	orderService := NewOrderService(db, nil)
	svc := NewNotifyService(db, orderService)

	mock.ExpectQuery("SELECT (.+) FROM orders WHERE status = \\$1 AND notify_status = \\$2 AND retry_count < 5").
		WithArgs(models.OrderStatusPaid, models.NotifyStatusPending, 100).
		WillReturnRows(sqlmock.NewRows([]string{"id", "order_no", "app_id", "out_trade_no", "channel", "amount", "currency",
			"subject", "body", "status", "notify_status", "retry_count", "channel_order_no", "pay_url",
			"paid_at", "notified_at", "expires_at", "created_at", "updated_at"}))

	err = svc.ProcessPendingNotifies(context.Background())
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNotifyService_GetErrorMsg_Additional(t *testing.T) {
	svc := NewNotifyService(nil, nil)
	assert.Equal(t, "", svc.getErrorMsg(nil))
	assert.Equal(t, assert.AnError.Error(), svc.getErrorMsg(assert.AnError))
}

func TestNotifyService_BuildNotifyRequestBody_JSON(t *testing.T) {
	svc := NewNotifyService(nil, nil)
	body := svc.buildNotifyRequestBody(&models.Order{
		OrderNo:    "ORD_1",
		OutTradeNo: "OUT_1",
		Amount:     10000,
		Status:     models.OrderStatusPaid,
		Channel:    "wechat_native",
	})

	var req NotifyRequest
	err := json.Unmarshal([]byte(body), &req)
	assert.NoError(t, err)
	assert.Equal(t, "ORD_1", req.OrderNo)
	assert.True(t, strings.Contains(body, "wechat_native"))
}
