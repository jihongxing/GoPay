package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"gopay/pkg/channel"
)

type fakePaymentChannel struct {
	closeCalled bool
}

func (f *fakePaymentChannel) Name() string { return "fake" }
func (f *fakePaymentChannel) CreateOrder(ctx context.Context, req *channel.CreateOrderRequest) (*channel.CreateOrderResponse, error) {
	return nil, nil
}
func (f *fakePaymentChannel) QueryOrder(ctx context.Context, req *channel.QueryOrderRequest) (*channel.QueryOrderResponse, error) {
	return nil, nil
}
func (f *fakePaymentChannel) HandleWebhook(ctx context.Context, req *channel.WebhookRequest) (*channel.WebhookResponse, error) {
	return nil, nil
}
func (f *fakePaymentChannel) Close() error {
	f.closeCalled = true
	return nil
}

func TestChannelManager_CacheHitAndReload(t *testing.T) {
	manager := NewChannelManager(nil)
	fake := &fakePaymentChannel{}
	manager.providers["app-1_wechat_native"] = fake

	provider, err := manager.GetProvider("app-1", "wechat_native")
	assert.NoError(t, err)
	assert.Same(t, fake, provider)

	err = manager.ReloadProvider("app-1", "wechat_native")
	assert.NoError(t, err)
	assert.True(t, fake.closeCalled)
	_, exists := manager.providers["app-1_wechat_native"]
	assert.False(t, exists)
}

func TestChannelManager_GetProvider_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	manager := NewChannelManager(db)
	mock.ExpectQuery("SELECT (.+) FROM channel_configs WHERE app_id = \\$1 AND channel = \\$2").
		WithArgs("app-1", "wechat_native").
		WillReturnError(sql.ErrNoRows)

	provider, err := manager.GetProvider("app-1", "wechat_native")
	assert.Error(t, err)
	assert.Nil(t, provider)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestChannelManager_GetProvider_Inactive(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	manager := NewChannelManager(db)
	now := time.Now()
	rows := sqlmock.NewRows([]string{"id", "app_id", "channel", "config", "status", "created_at", "updated_at"}).
		AddRow(1, "app-1", "wechat_native", `{"mch_id":"123","serial_no":"abc","api_v3_key":"key","private_key_path":"missing.pem"}`, "disabled", now, now)
	mock.ExpectQuery("SELECT (.+) FROM channel_configs WHERE app_id = \\$1 AND channel = \\$2").
		WithArgs("app-1", "wechat_native").
		WillReturnRows(rows)

	provider, err := manager.GetProvider("app-1", "wechat_native")
	assert.Error(t, err)
	assert.Nil(t, provider)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestChannelManager_GetProvider_CreateError(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	manager := NewChannelManager(db)
	now := time.Now()
	rows := sqlmock.NewRows([]string{"id", "app_id", "channel", "config", "status", "created_at", "updated_at"}).
		AddRow(1, "app-1", "wechat_native", `{"mch_id":"123","serial_no":"abc","api_v3_key":"key","private_key_path":"missing.pem"}`, "active", now, now)
	mock.ExpectQuery("SELECT (.+) FROM channel_configs WHERE app_id = \\$1 AND channel = \\$2").
		WithArgs("app-1", "wechat_native").
		WillReturnRows(rows)

	provider, err := manager.GetProvider("app-1", "wechat_native")
	assert.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "failed to create wechat native provider")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestChannelManager_GetProvider_InvalidJSON(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	manager := NewChannelManager(db)
	now := time.Now()
	rows := sqlmock.NewRows([]string{"id", "app_id", "channel", "config", "status", "created_at", "updated_at"}).
		AddRow(1, "app-1", "alipay_qr", `invalid-json`, "active", now, now)
	mock.ExpectQuery("SELECT (.+) FROM channel_configs WHERE app_id = \\$1 AND channel = \\$2").
		WithArgs("app-1", "alipay_qr").
		WillReturnRows(rows)

	provider, err := manager.GetProvider("app-1", "alipay_qr")
	assert.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "failed to unmarshal alipay config")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestChannelManager_Close(t *testing.T) {
	manager := NewChannelManager(nil)
	fake := &fakePaymentChannel{}
	manager.providers["app-1_wechat_native"] = fake

	assert.NoError(t, manager.Close())
	assert.True(t, fake.closeCalled)
	assert.Empty(t, manager.providers)
}
