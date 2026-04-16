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

func TestConfigService_ListApps(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	service := NewConfigService(db)

	countRows := sqlmock.NewRows([]string{"count"}).AddRow(2)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM apps WHERE 1=1 AND status = \\$1").
		WithArgs("active").
		WillReturnRows(countRows)

	now := time.Now()
	rows := sqlmock.NewRows([]string{"id", "app_id", "app_name", "app_secret", "callback_url", "status", "created_at", "updated_at"}).
		AddRow(1, "app-1", "App 1", "secret-1", "https://callback-1", "active", now, now).
		AddRow(2, "app-2", "App 2", "secret-2", "https://callback-2", "active", now, now)
	mock.ExpectQuery("SELECT id, app_id, app_name, app_secret, callback_url, status, created_at, updated_at").
		WithArgs("active", 10, 10).
		WillReturnRows(rows)

	apps, total, err := service.ListApps(context.Background(), 2, 10, "active")
	assert.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, apps, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestConfigService_GetApp_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	service := NewConfigService(db)
	mock.ExpectQuery("SELECT (.+) FROM apps WHERE app_id = \\$1").
		WithArgs("missing-app").
		WillReturnError(sql.ErrNoRows)

	app, err := service.GetApp(context.Background(), "missing-app")
	assert.Error(t, err)
	assert.Nil(t, app)
	assert.Contains(t, err.Error(), "app not found")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestConfigService_CreateApp(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	service := NewConfigService(db)
	now := time.Now()
	app := &models.App{
		AppID:       "app-1",
		AppName:     "App 1",
		AppSecret:   "secret-1",
		CallbackURL: "https://callback-1",
		Status:      "active",
	}

	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO apps \\(app_id, app_name, app_secret, callback_url, status\\)").
		WithArgs(app.AppID, app.AppName, app.AppSecret, app.CallbackURL, app.Status).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(7, now, now))
	mock.ExpectExec("INSERT INTO config_audit_logs").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = service.CreateApp(context.Background(), app, "operator", "127.0.0.1", "ua")
	assert.NoError(t, err)
	assert.Equal(t, int64(7), app.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestConfigService_UpdateApp(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	service := NewConfigService(db)
	now := time.Now()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT (.+) FROM apps WHERE app_id = \\$1").
		WithArgs("app-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "app_id", "app_name", "app_secret", "callback_url", "status", "created_at", "updated_at"}).
			AddRow(1, "app-1", "App 1", "secret-1", "https://callback-1", "active", now, now))
	mock.ExpectExec("UPDATE apps SET app_name = \\$1 WHERE app_id = \\$2").
		WithArgs("New App", "app-1").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO config_audit_logs").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = service.UpdateApp(context.Background(), "app-1", map[string]interface{}{"app_name": "New App"}, "operator", "127.0.0.1", "ua")
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestConfigService_ListChannelConfigs(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	service := NewConfigService(db)
	now := time.Now()
	rows := sqlmock.NewRows([]string{"id", "app_id", "channel", "config", "status", "created_at", "updated_at"}).
		AddRow(1, "app-1", "wechat_native", `{"mch_id":"123"}`, "active", now, now)
	mock.ExpectQuery("SELECT id, app_id, channel, config, status, created_at, updated_at FROM channel_configs").
		WithArgs("app-1").
		WillReturnRows(rows)

	configs, err := service.ListChannelConfigs(context.Background(), "app-1")
	assert.NoError(t, err)
	assert.Len(t, configs, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestConfigService_GetChannelConfig_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	service := NewConfigService(db)
	mock.ExpectQuery("SELECT (.+) FROM channel_configs WHERE id = \\$1").
		WithArgs(int64(99)).
		WillReturnError(sql.ErrNoRows)

	config, err := service.GetChannelConfig(context.Background(), 99)
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "channel config not found")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestConfigService_CreateChannelConfig(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	service := NewConfigService(db)
	now := time.Now()
	cfg := &models.ChannelConfig{
		AppID:   "app-1",
		Channel: "wechat_native",
		Config:  `{"mch_id":"123"}`,
		Status:  "active",
	}

	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO channel_configs \\(app_id, channel, config, status\\)").
		WithArgs(cfg.AppID, cfg.Channel, cfg.Config, cfg.Status).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(9, now, now))
	mock.ExpectExec("INSERT INTO config_audit_logs").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = service.CreateChannelConfig(context.Background(), cfg, "operator", "127.0.0.1", "ua")
	assert.NoError(t, err)
	assert.Equal(t, int64(9), cfg.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestConfigService_UpdateChannelConfig(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	service := NewConfigService(db)
	now := time.Now()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT (.+) FROM channel_configs WHERE id = \\$1").
		WithArgs(int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "app_id", "channel", "config", "status", "created_at", "updated_at"}).
			AddRow(9, "app-1", "wechat_native", `{"mch_id":"123"}`, "active", now, now))
	mock.ExpectExec("UPDATE channel_configs SET status = \\$1 WHERE id = \\$2").
		WithArgs("disabled", int64(9)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO config_audit_logs").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = service.UpdateChannelConfig(context.Background(), 9, map[string]interface{}{"status": "disabled"}, "operator", "127.0.0.1", "ua")
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
