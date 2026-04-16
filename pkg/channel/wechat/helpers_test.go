package wechat

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"gopay/pkg/channel"
)

type fakeWechatWebhookHandler struct {
	resp *channel.WebhookResponse
	err  error
}

func (f *fakeWechatWebhookHandler) HandleWebhook(ctx context.Context, req *channel.WebhookRequest) (*channel.WebhookResponse, error) {
	return f.resp, f.err
}

func TestConvertTradeState(t *testing.T) {
	tests := map[string]channel.OrderStatus{
		"SUCCESS": channel.OrderStatusPaid,
		"NOTPAY":  channel.OrderStatusPending,
		"CLOSED":  channel.OrderStatusClosed,
		"REFUND":  channel.OrderStatusRefund,
		"UNKNOWN": channel.OrderStatusPending,
	}

	for input, want := range tests {
		assert.Equal(t, want, convertTradeState(input))
	}
}

func TestParseWechatTime(t *testing.T) {
	got, err := parseWechatTime("2026-04-16T10:34:56+08:00")
	assert.NoError(t, err)
	assert.Equal(t, 2026, got.Year())
	assert.Equal(t, time.April, got.Month())
	assert.Equal(t, 16, got.Day())
}

func TestMapTradeState(t *testing.T) {
	tests := map[string]channel.OrderStatus{
		"SUCCESS": channel.OrderStatusPaid,
		"REFUND":  channel.OrderStatusRefund,
		"NOTPAY":  channel.OrderStatusPending,
		"CLOSED":  channel.OrderStatusClosed,
		"UNKNOWN": channel.OrderStatusPending,
	}

	for input, want := range tests {
		assert.Equal(t, want, mapTradeState(input))
	}
}

func TestProviderNames(t *testing.T) {
	assert.Equal(t, "wechat_native", (&Provider{}).Name())
	assert.Equal(t, "wechat_app", (&AppProvider{}).Name())
	assert.Equal(t, "wechat_jsapi", (&JSAPIProvider{}).Name())
	assert.Equal(t, "wechat_h5", (&H5Provider{}).Name())
}

func TestConstructors_ErrorPaths(t *testing.T) {
	cfg := &Config{
		MchID:          "mch",
		SerialNo:       "serial",
		APIV3Key:       "key",
		PrivateKeyPath: "missing.pem",
	}

	_, err := NewProvider(cfg)
	assert.Error(t, err)
	_, err = NewAppProvider(cfg)
	assert.Error(t, err)
	_, err = NewJSAPIProvider(cfg)
	assert.Error(t, err)
	_, err = NewH5Provider(cfg)
	assert.Error(t, err)
}

func TestConstructors_Success(t *testing.T) {
	origNewClient := newWechatClient
	newWechatClient = func(ctx context.Context, opts ...core.ClientOption) (*core.Client, error) {
		return &core.Client{}, nil
	}
	t.Cleanup(func() { newWechatClient = origNewClient })

	privateKey := mustGenerateRSAKey(t)
	privPath := filepath.Join(t.TempDir(), "wechat_private.pem")
	assert.NoError(t, os.WriteFile(privPath, pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: mustMarshalPKCS8(t, privateKey),
	}), 0600))

	cfg := &Config{
		MchID:          "mch123",
		SerialNo:       "serial123",
		APIV3Key:       "12345678901234567890123456789012",
		PrivateKeyPath: privPath,
	}

	base, err := NewProvider(cfg)
	assert.NoError(t, err)
	assert.Equal(t, "wechat_native", base.Name())
	assert.NoError(t, base.Close())

	app, err := NewAppProvider(cfg)
	assert.NoError(t, err)
	assert.Equal(t, "wechat_app", app.Name())

	jsapi, err := NewJSAPIProvider(cfg)
	assert.NoError(t, err)
	assert.Equal(t, "wechat_jsapi", jsapi.Name())

	h5, err := NewH5Provider(cfg)
	assert.NoError(t, err)
	assert.Equal(t, "wechat_h5", h5.Name())
}

func TestProvider_HandleWebhook_Delegates(t *testing.T) {
	orig := newWechatWebhookHandler
	newWechatWebhookHandler = func(mchID, apiV3Key, mchCertSerialNo string, mchPrivateKey string) (wechatWebhookHandler, error) {
		return &fakeWechatWebhookHandler{
			resp: &channel.WebhookResponse{
				Success:         true,
				PlatformTradeNo: "OUT_1",
				Status:          channel.OrderStatusPaid,
				ResponseBody:    []byte("ok"),
			},
		}, nil
	}
	t.Cleanup(func() { newWechatWebhookHandler = orig })

	provider := &Provider{mchID: "mch", apiV3Key: "key", serialNo: "serial"}
	resp, err := provider.HandleWebhook(context.Background(), &channel.WebhookRequest{RawBody: []byte("ok")})
	assert.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, "OUT_1", resp.PlatformTradeNo)
}

func TestProvider_HandleWebhook_HandlerError(t *testing.T) {
	orig := newWechatWebhookHandler
	newWechatWebhookHandler = func(mchID, apiV3Key, mchCertSerialNo string, mchPrivateKey string) (wechatWebhookHandler, error) {
		return nil, assert.AnError
	}
	t.Cleanup(func() { newWechatWebhookHandler = orig })

	provider := &Provider{mchID: "mch", apiV3Key: "key", serialNo: "serial"}
	resp, err := provider.HandleWebhook(context.Background(), &channel.WebhookRequest{RawBody: []byte("ok")})
	assert.Error(t, err)
	assert.False(t, resp.Success)
	assert.Contains(t, string(resp.ResponseBody), "FAIL")
}

func mustGenerateRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.NoError(t, err)
	return key
}

func mustMarshalPKCS8(t *testing.T, key *rsa.PrivateKey) []byte {
	t.Helper()

	encoded, err := x509.MarshalPKCS8PrivateKey(key)
	assert.NoError(t, err)
	return encoded
}
