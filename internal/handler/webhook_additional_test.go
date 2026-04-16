package handler

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"gopay/internal/models"
	"gopay/internal/service"
	"gopay/pkg/channel"
)

type webhookTestChannelManager struct {
	provider channel.PaymentChannel
	err      error
}

func (m *webhookTestChannelManager) GetProvider(appID, channelName string) (channel.PaymentChannel, error) {
	return m.provider, m.err
}

type webhookErrorBody struct{}

func (w *webhookErrorBody) Read(_ []byte) (int, error) {
	return 0, errors.New("read body failed")
}

func (w *webhookErrorBody) Close() error {
	return nil
}

func setupWebhookTestEnv(t *testing.T, db *sql.DB, cm webhookChannelManager) {
	t.Helper()

	InitServices(service.NewOrderService(db, cm))
	InitWebhookServices(cm, service.NewNotifyService(db, service.NewOrderService(db, cm)))

	t.Cleanup(func() {
		InitServices(nil)
		InitWebhookServices(nil, nil)
	})
}

func newWebhookOrderRows(orderNo, appID, outTradeNo, channelName string) *sqlmock.Rows {
	now := time.Now()
	paidAt := now.Add(-1 * time.Hour)

	return sqlmock.NewRows([]string{
		"id", "order_no", "app_id", "out_trade_no", "channel", "amount", "currency",
		"subject", "body", "status", "notify_status", "retry_count",
		"channel_order_no", "pay_url", "paid_at", "notified_at",
		"expires_at", "created_at", "updated_at",
	}).AddRow(
		1, orderNo, appID, outTradeNo, channelName, 10000, "CNY",
		"测试商品", "测试描述", models.OrderStatusPending, models.NotifyStatusPending, 0,
		"PLAT_001", "http://pay.url", &paidAt, nil,
		now.Add(2*time.Hour), now, now,
	)
}

func TestParseOutTradeNoFromWebhook(t *testing.T) {
	tests := []struct {
		name    string
		body    []byte
		want    string
		wantErr bool
		fn      func([]byte) (string, error)
	}{
		{
			name:    "wechat ok",
			body:    []byte(`{"out_trade_no":"OUT_001"}`),
			want:    "OUT_001",
			fn:      parseOutTradeNoFromWechatWebhook,
		},
		{
			name:    "wechat missing field",
			body:    []byte(`{"foo":"bar"}`),
			wantErr: true,
			fn:      parseOutTradeNoFromWechatWebhook,
		},
		{
			name:    "wechat invalid json",
			body:    []byte(`{`),
			wantErr: true,
			fn:      parseOutTradeNoFromWechatWebhook,
		},
		{
			name:    "alipay ok",
			body:    []byte(`{"out_trade_no":"OUT_002"}`),
			want:    "OUT_002",
			fn:      parseOutTradeNoFromAlipayWebhook,
		},
		{
			name:    "alipay missing field",
			body:    []byte(`{"foo":"bar"}`),
			wantErr: true,
			fn:      parseOutTradeNoFromAlipayWebhook,
		},
		{
			name:    "alipay invalid json",
			body:    []byte(`{`),
			wantErr: true,
			fn:      parseOutTradeNoFromAlipayWebhook,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.fn(tt.body)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, got)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWechatWebhook_Branches(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		body           io.ReadCloser
		status         int
		wantBody       string
		orderSetup     func(mock sqlmock.Sqlmock)
		manager        webhookChannelManager
	}{
		{
			name:   "read body error",
			body:   &webhookErrorBody{},
			status: http.StatusBadRequest,
			wantBody: `{"code":"FAIL","message":"invalid request"}`,
		},
		{
			name:     "invalid webhook payload",
			body:     io.NopCloser(strings.NewReader(`{"foo":"bar"}`)),
			status:   http.StatusBadRequest,
			wantBody: `{"code":"FAIL","message":"invalid webhook data"}`,
		},
		{
			name: "order not found",
			body: io.NopCloser(strings.NewReader(`{"out_trade_no":"OUT_404"}`)),
			orderSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT (.+) FROM orders WHERE out_trade_no = \\$1").
					WithArgs("OUT_404").
					WillReturnError(sql.ErrNoRows)
			},
			status:   http.StatusBadRequest,
			wantBody: `{"code":"FAIL","message":"order not found"}`,
		},
		{
			name: "provider lookup error",
			body: io.NopCloser(strings.NewReader(`{"out_trade_no":"OUT_001"}`)),
			orderSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT (.+) FROM orders WHERE out_trade_no = \\$1").
					WithArgs("OUT_001").
					WillReturnRows(newWebhookOrderRows("ORD_001", "test_app", "OUT_001", models.ChannelWechatNative))
			},
			manager:  &webhookTestChannelManager{err: errors.New("provider failed")},
			status:   http.StatusInternalServerError,
			wantBody: `{"code":"FAIL","message":"系统错误"}`,
		},
		{
			name: "handle webhook error",
			body: io.NopCloser(strings.NewReader(`{"out_trade_no":"OUT_001"}`)),
			orderSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT (.+) FROM orders WHERE out_trade_no = \\$1").
					WithArgs("OUT_001").
					WillReturnRows(newWebhookOrderRows("ORD_001", "test_app", "OUT_001", models.ChannelWechatNative))
			},
			manager: &webhookTestChannelManager{
				provider: &MockPaymentChannel{
					handleWebhookFunc: func(context.Context, *channel.WebhookRequest) (*channel.WebhookResponse, error) {
						return nil, errors.New("handle failed")
					},
				},
			},
			status:   http.StatusInternalServerError,
			wantBody: `{"code":"FAIL","message":"处理失败"}`,
		},
		{
			name: "verification failed",
			body: io.NopCloser(strings.NewReader(`{"out_trade_no":"OUT_001"}`)),
			orderSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT (.+) FROM orders WHERE out_trade_no = \\$1").
					WithArgs("OUT_001").
					WillReturnRows(newWebhookOrderRows("ORD_001", "test_app", "OUT_001", models.ChannelWechatNative))
			},
			manager: &webhookTestChannelManager{
				provider: &MockPaymentChannel{
					handleWebhookFunc: func(context.Context, *channel.WebhookRequest) (*channel.WebhookResponse, error) {
						return &channel.WebhookResponse{
							Success:      false,
							ResponseBody: []byte(`{"code":"FAIL"}`),
						}, nil
					},
				},
			},
			status:   http.StatusOK,
			wantBody: `{"code":"FAIL"}`,
		},
		{
			name: "success pending",
			body: io.NopCloser(strings.NewReader(`{"out_trade_no":"OUT_001"}`)),
			orderSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT (.+) FROM orders WHERE out_trade_no = \\$1").
					WithArgs("OUT_001").
					WillReturnRows(newWebhookOrderRows("ORD_001", "test_app", "OUT_001", models.ChannelWechatNative))
			},
			manager: &webhookTestChannelManager{
				provider: &MockPaymentChannel{
					handleWebhookFunc: func(context.Context, *channel.WebhookRequest) (*channel.WebhookResponse, error) {
						return &channel.WebhookResponse{
							Success:         true,
							Status:          channel.OrderStatusPending,
							PlatformTradeNo: "WX_001",
							ResponseBody:    []byte("success"),
						}, nil
					},
				},
			},
			status:   http.StatusOK,
			wantBody: "success",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			assert.NoError(t, err)
			defer db.Close()

			if tt.orderSetup != nil {
				tt.orderSetup(mock)
			}

			manager := tt.manager
			if manager == nil {
				manager = &webhookTestChannelManager{
					provider: &MockPaymentChannel{},
				}
			}

			setupWebhookTestEnv(t, db, manager)

			router := gin.New()
			router.POST("/webhook/wechat", WechatWebhook)

			req := httptest.NewRequest(http.MethodPost, "/webhook/wechat", nil)
			req.Body = tt.body
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Test", "1")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.status, w.Code)
			assert.Equal(t, tt.wantBody, w.Body.String())
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestAlipayWebhook_Branches(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		body       io.ReadCloser
		status     int
		wantBody   string
		orderSetup func(mock sqlmock.Sqlmock)
		manager    webhookChannelManager
	}{
		{
			name:   "read body error",
			body:   &webhookErrorBody{},
			status: http.StatusOK,
			wantBody: "failure",
		},
		{
			name:   "invalid webhook payload",
			body:   io.NopCloser(strings.NewReader(`{"foo":"bar"}`)),
			status: http.StatusOK,
			wantBody: "failure",
		},
		{
			name: "order not found",
			body: io.NopCloser(strings.NewReader(`{"out_trade_no":"OUT_404"}`)),
			orderSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT (.+) FROM orders WHERE out_trade_no = \\$1").
					WithArgs("OUT_404").
					WillReturnError(sql.ErrNoRows)
			},
			status:   http.StatusOK,
			wantBody: "failure",
		},
		{
			name: "provider lookup error",
			body: io.NopCloser(strings.NewReader(`{"out_trade_no":"OUT_001"}`)),
			orderSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT (.+) FROM orders WHERE out_trade_no = \\$1").
					WithArgs("OUT_001").
					WillReturnRows(newWebhookOrderRows("ORD_001", "test_app", "OUT_001", models.ChannelAlipayQR))
			},
			manager:  &webhookTestChannelManager{err: errors.New("provider failed")},
			status:   http.StatusOK,
			wantBody: "failure",
		},
		{
			name: "handle webhook error",
			body: io.NopCloser(strings.NewReader(`{"out_trade_no":"OUT_001"}`)),
			orderSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT (.+) FROM orders WHERE out_trade_no = \\$1").
					WithArgs("OUT_001").
					WillReturnRows(newWebhookOrderRows("ORD_001", "test_app", "OUT_001", models.ChannelAlipayQR))
			},
			manager: &webhookTestChannelManager{
				provider: &MockPaymentChannel{
					handleWebhookFunc: func(context.Context, *channel.WebhookRequest) (*channel.WebhookResponse, error) {
						return nil, errors.New("handle failed")
					},
				},
			},
			status:   http.StatusOK,
			wantBody: "failure",
		},
		{
			name: "verification failed",
			body: io.NopCloser(strings.NewReader(`{"out_trade_no":"OUT_001"}`)),
			orderSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT (.+) FROM orders WHERE out_trade_no = \\$1").
					WithArgs("OUT_001").
					WillReturnRows(newWebhookOrderRows("ORD_001", "test_app", "OUT_001", models.ChannelAlipayQR))
			},
			manager: &webhookTestChannelManager{
				provider: &MockPaymentChannel{
					handleWebhookFunc: func(context.Context, *channel.WebhookRequest) (*channel.WebhookResponse, error) {
						return &channel.WebhookResponse{
							Success:      false,
							ResponseBody: []byte("fail"),
						}, nil
					},
				},
			},
			status:   http.StatusOK,
			wantBody: "fail",
		},
		{
			name: "success pending",
			body: io.NopCloser(strings.NewReader(`{"out_trade_no":"OUT_001"}`)),
			orderSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT (.+) FROM orders WHERE out_trade_no = \\$1").
					WithArgs("OUT_001").
					WillReturnRows(newWebhookOrderRows("ORD_001", "test_app", "OUT_001", models.ChannelAlipayQR))
			},
			manager: &webhookTestChannelManager{
				provider: &MockPaymentChannel{
					handleWebhookFunc: func(context.Context, *channel.WebhookRequest) (*channel.WebhookResponse, error) {
						return &channel.WebhookResponse{
							Success:         true,
							Status:          channel.OrderStatusPending,
							PlatformTradeNo: "ALI_001",
							ResponseBody:    []byte("success"),
						}, nil
					},
				},
			},
			status:   http.StatusOK,
			wantBody: "success",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			assert.NoError(t, err)
			defer db.Close()

			if tt.orderSetup != nil {
				tt.orderSetup(mock)
			}

			manager := tt.manager
			if manager == nil {
				manager = &webhookTestChannelManager{
					provider: &MockPaymentChannel{},
				}
			}

			setupWebhookTestEnv(t, db, manager)

			router := gin.New()
			router.POST("/webhook/alipay", AlipayWebhook)

			req := httptest.NewRequest(http.MethodPost, "/webhook/alipay", nil)
			req.Body = tt.body
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("X-Test", "1")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.status, w.Code)
			assert.Equal(t, tt.wantBody, w.Body.String())
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
