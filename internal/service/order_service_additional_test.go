package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"gopay/internal/models"
)

func TestOrderService_QueryOrderByOutTradeNoGlobal(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	service := NewOrderService(db, nil)
	now := time.Now()
	rows := sqlmock.NewRows([]string{"id", "order_no", "app_id", "out_trade_no", "channel", "amount", "currency",
		"subject", "body", "status", "notify_status", "retry_count", "channel_order_no", "pay_url",
		"paid_at", "notified_at", "expires_at", "created_at", "updated_at"}).
		AddRow(1, "ORD_1", "app-1", "OUT_1", "wechat_native", 10000, "CNY", "subject", "body",
			models.OrderStatusPaid, models.NotifyStatusNotified, 1, "WX_1", "pay://url", &now, &now, now, now, now)
	mock.ExpectQuery("SELECT (.+) FROM orders WHERE out_trade_no = \\$1").
		WithArgs("OUT_1").
		WillReturnRows(rows)

	order, err := service.QueryOrderByOutTradeNoGlobal(context.Background(), "OUT_1")
	assert.NoError(t, err)
	assert.Equal(t, "ORD_1", order.OrderNo)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOrderService_QueryOrderByOutTradeNo_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	service := NewOrderService(db, nil)
	mock.ExpectQuery("SELECT (.+) FROM orders WHERE app_id = \\$1 AND out_trade_no = \\$2").
		WithArgs("app-1", "OUT_404").
		WillReturnError(sql.ErrNoRows)

	order, err := service.QueryOrderByOutTradeNo(context.Background(), "app-1", "OUT_404")
	assert.Error(t, err)
	assert.Nil(t, order)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOrderService_ListPendingNotifyOrders(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	service := NewOrderService(db, nil)
	now := time.Now()
	rows := sqlmock.NewRows([]string{"id", "order_no", "app_id", "out_trade_no", "channel", "amount", "currency",
		"subject", "body", "status", "notify_status", "retry_count", "channel_order_no", "pay_url",
		"paid_at", "notified_at", "expires_at", "created_at", "updated_at"}).
		AddRow(1, "ORD_1", "app-1", "OUT_1", "wechat_native", 10000, "CNY", "subject", "body",
			models.OrderStatusPaid, models.NotifyStatusPending, 0, "WX_1", "pay://url", &now, nil, now, now, now)
	mock.ExpectQuery("SELECT (.+) FROM orders WHERE status = \\$1 AND notify_status = \\$2 AND retry_count < 5").
		WithArgs(models.OrderStatusPaid, models.NotifyStatusPending, 20).
		WillReturnRows(rows)

	orders, err := service.ListPendingNotifyOrders(context.Background(), 20)
	assert.NoError(t, err)
	assert.Len(t, orders, 1)
	assert.Equal(t, "ORD_1", orders[0].OrderNo)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOrderService_ListFailedOrders(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	service := NewOrderService(db, nil)
	now := time.Now()
	rows := sqlmock.NewRows([]string{"id", "order_no", "app_id", "out_trade_no", "channel", "amount", "currency",
		"subject", "body", "status", "notify_status", "retry_count", "channel_order_no", "pay_url",
		"paid_at", "notified_at", "expires_at", "created_at", "updated_at"}).
		AddRow(1, "ORD_1", "app-1", "OUT_1", "wechat_native", 10000, "CNY", "subject", "body",
			models.OrderStatusPaid, models.NotifyStatusFailedNotify, 3, "WX_1", "pay://url", &now, nil, now, now, now)
	mock.ExpectQuery("SELECT (.+) FROM orders WHERE notify_status = \\$1").
		WithArgs(models.NotifyStatusFailedNotify, 20).
		WillReturnRows(rows)

	orders, err := service.ListFailedOrders(context.Background(), 20)
	assert.NoError(t, err)
	assert.Len(t, orders, 1)
	assert.Equal(t, "ORD_1", orders[0].OrderNo)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOrderService_IncrementAndNotifyStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	service := NewOrderService(db, nil)
	mock.ExpectExec("UPDATE orders\\s+SET retry_count = retry_count \\+ 1, updated_at = NOW\\(\\)\\s+WHERE order_no = \\$1").
		WithArgs("ORD_1").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("UPDATE orders\\s+SET\\s+notify_status = \\$1, notified_at = NOW\\(\\), updated_at = NOW\\(\\)\\s+WHERE order_no = \\$2").
		WithArgs(models.NotifyStatusNotified, "ORD_1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	assert.NoError(t, service.IncrementRetryCount(context.Background(), "ORD_1"))
	assert.NoError(t, service.UpdateNotifyStatus(context.Background(), "ORD_1", models.NotifyStatusNotified))
	assert.NoError(t, mock.ExpectationsWereMet())
}
