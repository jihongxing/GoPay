package stripe

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"gopay/pkg/channel"

	"github.com/stretchr/testify/assert"
	stripe "github.com/stripe/stripe-go/v85"
)

func TestNewProvider_Success(t *testing.T) {
	p, err := NewProvider(&Config{
		SecretKey:     "sk_test_123",
		WebhookSecret: "whsec_123",
		Currency:      "usd",
	})
	assert.NoError(t, err)
	assert.NotNil(t, p)
	assert.Equal(t, "stripe", p.Name())
	assert.Equal(t, "usd", p.currency)
}

func TestNewProvider_DefaultCurrency(t *testing.T) {
	p, err := NewProvider(&Config{
		SecretKey:     "sk_test_123",
		WebhookSecret: "whsec_123",
	})
	assert.NoError(t, err)
	assert.Equal(t, "usd", p.currency)
}

func TestNewProvider_MissingSecretKey(t *testing.T) {
	p, err := NewProvider(&Config{WebhookSecret: "whsec_123"})
	assert.Error(t, err)
	assert.Nil(t, p)
	assert.Contains(t, err.Error(), "secret_key")
}

func TestNewProvider_MissingWebhookSecret(t *testing.T) {
	p, err := NewProvider(&Config{SecretKey: "sk_test_123"})
	assert.Error(t, err)
	assert.Nil(t, p)
	assert.Contains(t, err.Error(), "webhook_secret")
}

func TestProvider_Close(t *testing.T) {
	p, _ := NewProvider(&Config{SecretKey: "sk_test_123", WebhookSecret: "whsec_123"})
	assert.NoError(t, p.Close())
}

func TestProvider_CreateOrder_NilRequest(t *testing.T) {
	p, _ := NewProvider(&Config{SecretKey: "sk_test_123", WebhookSecret: "whsec_123"})
	resp, err := p.CreateOrder(context.Background(), nil)
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestProvider_QueryOrder_NilRequest(t *testing.T) {
	p, _ := NewProvider(&Config{SecretKey: "sk_test_123", WebhookSecret: "whsec_123"})
	resp, err := p.QueryOrder(context.Background(), nil)
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestProvider_QueryOrder_EmptyPlatformNo(t *testing.T) {
	p, _ := NewProvider(&Config{SecretKey: "sk_test_123", WebhookSecret: "whsec_123"})
	resp, err := p.QueryOrder(context.Background(), &channel.QueryOrderRequest{})
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestProvider_Refund_NilRequest(t *testing.T) {
	p, _ := NewProvider(&Config{SecretKey: "sk_test_123", WebhookSecret: "whsec_123"})
	resp, err := p.Refund(context.Background(), nil)
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestProvider_QueryRefund_NilRequest(t *testing.T) {
	p, _ := NewProvider(&Config{SecretKey: "sk_test_123", WebhookSecret: "whsec_123"})
	resp, err := p.QueryRefund(context.Background(), nil)
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestProvider_QueryRefund_EmptyRefundNo(t *testing.T) {
	p, _ := NewProvider(&Config{SecretKey: "sk_test_123", WebhookSecret: "whsec_123"})
	resp, err := p.QueryRefund(context.Background(), &channel.RefundRequest{})
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestProvider_HandleWebhook_NilRequest(t *testing.T) {
	p, _ := NewProvider(&Config{SecretKey: "sk_test_123", WebhookSecret: "whsec_123"})
	resp, err := p.HandleWebhook(context.Background(), nil)
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestProvider_HandleWebhook_EmptyBody(t *testing.T) {
	p, _ := NewProvider(&Config{SecretKey: "sk_test_123", WebhookSecret: "whsec_123"})
	resp, err := p.HandleWebhook(context.Background(), &channel.WebhookRequest{})
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestProvider_HandleWebhook_MissingSignature(t *testing.T) {
	p, _ := NewProvider(&Config{SecretKey: "sk_test_123", WebhookSecret: "whsec_123"})
	resp, err := p.HandleWebhook(context.Background(), &channel.WebhookRequest{
		RawBody: []byte(`{}`),
		Headers: map[string]string{},
	})
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "Stripe-Signature")
}

func makeSignedWebhook(t *testing.T, secret string, event stripe.Event) ([]byte, string) {
	body, err := json.Marshal(event)
	assert.NoError(t, err)

	ts := fmt.Sprintf("%d", time.Now().Unix())
	signedPayload := fmt.Sprintf("%s.%s", ts, string(body))
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signedPayload))
	sig := hex.EncodeToString(mac.Sum(nil))

	sigHeader := fmt.Sprintf("t=%s,v1=%s", ts, sig)
	return body, sigHeader
}

func TestProvider_HandleWebhook_CheckoutCompleted(t *testing.T) {
	secret := "whsec_test_secret"
	p, _ := NewProvider(&Config{SecretKey: "sk_test_123", WebhookSecret: secret})

	sessionData := map[string]interface{}{
		"id":           "cs_test_123",
		"amount_total": 5000,
		"metadata":     map[string]string{"order_id": "ORD_001"},
	}
	rawData, _ := json.Marshal(sessionData)

	event := stripe.Event{
		Type: "checkout.session.completed",
		Data: &stripe.EventData{Raw: rawData},
	}

	body, sigHeader := makeSignedWebhook(t, secret, event)

	resp, err := p.HandleWebhook(context.Background(), &channel.WebhookRequest{
		RawBody: body,
		Headers: map[string]string{"Stripe-Signature": sigHeader},
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.Equal(t, "ORD_001", resp.OrderID)
	assert.Equal(t, channel.OrderStatusPaid, resp.Status)
}

func TestProvider_HandleWebhook_UnhandledEvent(t *testing.T) {
	secret := "whsec_test_secret"
	p, _ := NewProvider(&Config{SecretKey: "sk_test_123", WebhookSecret: secret})

	event := stripe.Event{
		Type: "customer.created",
		Data: &stripe.EventData{Raw: json.RawMessage(`{}`)},
	}

	body, sigHeader := makeSignedWebhook(t, secret, event)

	resp, err := p.HandleWebhook(context.Background(), &channel.WebhookRequest{
		RawBody: body,
		Headers: map[string]string{"Stripe-Signature": sigHeader},
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Success)
}

func TestProvider_HandleWebhook_InvalidSignature(t *testing.T) {
	p, _ := NewProvider(&Config{SecretKey: "sk_test_123", WebhookSecret: "whsec_real"})

	event := stripe.Event{
		Type: "checkout.session.completed",
		Data: &stripe.EventData{Raw: json.RawMessage(`{}`)},
	}

	body, sigHeader := makeSignedWebhook(t, "whsec_wrong", event)

	resp, err := p.HandleWebhook(context.Background(), &channel.WebhookRequest{
		RawBody: body,
		Headers: map[string]string{"Stripe-Signature": sigHeader},
	})

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "signature")
}

func TestVerifyWebhookSignature_Valid(t *testing.T) {
	secret := "whsec_test"
	payload := []byte(`{"test": true}`)
	ts := fmt.Sprintf("%d", time.Now().Unix())

	signedPayload := fmt.Sprintf("%s.%s", ts, string(payload))
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signedPayload))
	sig := hex.EncodeToString(mac.Sum(nil))

	sigHeader := fmt.Sprintf("t=%s,v1=%s", ts, sig)
	err := verifyWebhookSignature(payload, sigHeader, secret)
	assert.NoError(t, err)
}

func TestVerifyWebhookSignature_ExpiredTimestamp(t *testing.T) {
	secret := "whsec_test"
	payload := []byte(`{"test": true}`)
	ts := fmt.Sprintf("%d", time.Now().Unix()-600) // 10 minutes ago

	signedPayload := fmt.Sprintf("%s.%s", ts, string(payload))
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signedPayload))
	sig := hex.EncodeToString(mac.Sum(nil))

	sigHeader := fmt.Sprintf("t=%s,v1=%s", ts, sig)
	err := verifyWebhookSignature(payload, sigHeader, secret)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too old")
}

func TestVerifyWebhookSignature_InvalidFormat(t *testing.T) {
	err := verifyWebhookSignature([]byte(`{}`), "invalid", "secret")
	assert.Error(t, err)
}

func TestMapCheckoutStatus(t *testing.T) {
	assert.Equal(t, channel.OrderStatusPaid, mapCheckoutStatus(stripe.CheckoutSessionPaymentStatusPaid))
	assert.Equal(t, channel.OrderStatusPending, mapCheckoutStatus(stripe.CheckoutSessionPaymentStatusUnpaid))
	assert.Equal(t, channel.OrderStatusPaid, mapCheckoutStatus(stripe.CheckoutSessionPaymentStatusNoPaymentRequired))
	assert.Equal(t, channel.OrderStatusPending, mapCheckoutStatus("unknown"))
}

func TestMapRefundStatus(t *testing.T) {
	assert.Equal(t, channel.RefundStatusSuccess, mapRefundStatus(stripe.RefundStatusSucceeded))
	assert.Equal(t, channel.RefundStatusProcessing, mapRefundStatus(stripe.RefundStatusPending))
	assert.Equal(t, channel.RefundStatusFailed, mapRefundStatus(stripe.RefundStatusFailed))
	assert.Equal(t, channel.RefundStatusPending, mapRefundStatus("unknown"))
}
