package alipay

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopay/pkg/channel"
)

func TestConvertTradeStatus(t *testing.T) {
	tests := map[string]channel.OrderStatus{
		"TRADE_SUCCESS":  channel.OrderStatusPaid,
		"TRADE_FINISHED": channel.OrderStatusPaid,
		"WAIT_BUYER_PAY": channel.OrderStatusPending,
		"TRADE_CLOSED":   channel.OrderStatusClosed,
		"UNKNOWN":        channel.OrderStatusPending,
	}

	for input, want := range tests {
		assert.Equal(t, want, convertTradeStatus(input))
	}
}

func TestParseAndFormatAmount(t *testing.T) {
	assert.Equal(t, int64(10000), parseAmount("100.00"))
	assert.Equal(t, "100.00", formatAmount(10000))
}

func TestProviderNames(t *testing.T) {
	assert.Equal(t, "alipay", (&Provider{}).Name())
	assert.Equal(t, "alipay_app", (&AppProvider{}).Name())
	assert.Equal(t, "alipay_qr", (&QRProvider{}).Name())
	assert.Equal(t, "alipay_wap", (&WapProvider{}).Name())
	assert.Equal(t, "alipay_face", (&FaceProvider{}).Name())
}

func TestConstructors_ErrorPaths(t *testing.T) {
	cfg := &Config{
		AppID:           "",
		PrivateKey:      "",
		AlipayPublicKey: "",
		IsProduction:    false,
	}

	_, err := NewProvider(cfg)
	assert.Error(t, err)
	_, err = NewAppProvider(cfg)
	assert.Error(t, err)
	_, err = NewQRProvider(cfg)
	assert.Error(t, err)
	_, err = NewWapProvider(cfg)
	assert.Error(t, err)
	_, err = NewFaceProvider(cfg)
	assert.Error(t, err)
}

func TestConstructors_Success(t *testing.T) {
	privateKey, publicKey := mustGenerateAlipayKeys(t)

	cfg := &Config{
		AppID:           "2021001234567890",
		PrivateKey:      privateKey,
		AlipayPublicKey: publicKey,
		IsProduction:    false,
	}

	base, err := NewProvider(cfg)
	assert.NoError(t, err)
	assert.Equal(t, "alipay", base.Name())
	assert.NoError(t, base.Close())

	app, err := NewAppProvider(cfg)
	assert.NoError(t, err)
	assert.Equal(t, "alipay_app", app.Name())

	qr, err := NewQRProvider(cfg)
	assert.NoError(t, err)
	assert.Equal(t, "alipay_qr", qr.Name())

	wap, err := NewWapProvider(cfg)
	assert.NoError(t, err)
	assert.Equal(t, "alipay_wap", wap.Name())

	face, err := NewFaceProvider(cfg)
	assert.NoError(t, err)
	assert.Equal(t, "alipay_face", face.Name())
}

func TestHandleWebhook_ParseErrorPaths(t *testing.T) {
	provider := &Provider{}
	resp, err := provider.HandleWebhook(context.Background(), &channel.WebhookRequest{RawBody: []byte("%")})
	assert.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, []byte("failure"), resp.ResponseBody)
}

func TestWrapperHandleWebhook_ParseErrorPaths(t *testing.T) {
	wrappers := []struct {
		name string
		h    interface {
			HandleWebhook(context.Context, *channel.WebhookRequest) (*channel.WebhookResponse, error)
		}
	}{
		{"app", &AppProvider{Provider: &Provider{}}},
		{"qr", &QRProvider{Provider: &Provider{}}},
		{"wap", &WapProvider{Provider: &Provider{}}},
		{"face", &FaceProvider{Provider: &Provider{}}},
	}

	for _, tt := range wrappers {
		resp, err := tt.h.HandleWebhook(context.Background(), &channel.WebhookRequest{RawBody: []byte("%")})
		assert.NoError(t, err, tt.name)
		assert.False(t, resp.Success, tt.name)
		assert.Equal(t, []byte("failure"), resp.ResponseBody, tt.name)
	}
}

func mustGenerateAlipayKeys(t *testing.T) (string, string) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.NoError(t, err)

	privBytes := x509.MarshalPKCS1PrivateKey(key)
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: privBytes})

	pubBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	assert.NoError(t, err)
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes})

	return string(privPEM), string(pubPEM)
}
