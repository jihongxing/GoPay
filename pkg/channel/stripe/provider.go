package stripe

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gopay/pkg/channel"
	"gopay/pkg/logger"

	stripe "github.com/stripe/stripe-go/v85"
)

// Config Stripe 配置
type Config struct {
	SecretKey     string `json:"secret_key"`     // Stripe Secret Key (sk_live_xxx)
	WebhookSecret string `json:"webhook_secret"` // Webhook 签名密钥 (whsec_xxx)
	Currency      string `json:"currency"`       // 默认货币 (usd, cny 等)
	SuccessURL    string `json:"success_url"`    // 支付成功跳转 URL
	CancelURL     string `json:"cancel_url"`     // 支付取消跳转 URL
}

// Provider Stripe 支付 Provider
type Provider struct {
	client        *stripe.Client
	webhookSecret string
	currency      string
	successURL    string
	cancelURL     string
}

// NewProvider 创建 Stripe Provider
func NewProvider(cfg *Config) (*Provider, error) {
	if cfg.SecretKey == "" {
		return nil, fmt.Errorf("stripe secret_key is required")
	}
	if cfg.WebhookSecret == "" {
		return nil, fmt.Errorf("stripe webhook_secret is required")
	}

	currency := cfg.Currency
	if currency == "" {
		currency = "usd"
	}

	successURL := cfg.SuccessURL
	if successURL == "" {
		successURL = "https://example.com/success?session_id={CHECKOUT_SESSION_ID}"
	}
	cancelURL := cfg.CancelURL
	if cancelURL == "" {
		cancelURL = "https://example.com/cancel"
	}

	sc := stripe.NewClient(cfg.SecretKey)

	logger.Info("Stripe provider initialized, currency=%s", currency)

	return &Provider{
		client:        sc,
		webhookSecret: cfg.WebhookSecret,
		currency:      currency,
		successURL:    successURL,
		cancelURL:     cancelURL,
	}, nil
}

// Name 返回渠道名称
func (p *Provider) Name() string {
	return "stripe"
}

// CreateOrder 创建支付订单（Stripe Checkout Session）
func (p *Provider) CreateOrder(ctx context.Context, req *channel.CreateOrderRequest) (*channel.CreateOrderResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}

	params := &stripe.CheckoutSessionCreateParams{
		Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
		LineItems: []*stripe.CheckoutSessionCreateLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionCreateLineItemPriceDataParams{
					Currency: stripe.String(p.currency),
					ProductData: &stripe.CheckoutSessionCreateLineItemPriceDataProductDataParams{
						Name:        stripe.String(req.Subject),
						Description: stripe.String(req.Description),
					},
					UnitAmount: stripe.Int64(req.Amount),
				},
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL: stripe.String(p.successURL),
		CancelURL:  stripe.String(p.cancelURL),
		Metadata: map[string]string{
			"order_id":     req.OrderID,
			"biz_order_no": req.BizOrderNo,
		},
	}

	session, err := p.client.V1CheckoutSessions.Create(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("stripe create checkout session failed: %w", err)
	}

	return &channel.CreateOrderResponse{
		PlatformTradeNo: session.ID,
		PayURL:          session.URL,
	}, nil
}

// QueryOrder 查询订单状态
func (p *Provider) QueryOrder(ctx context.Context, req *channel.QueryOrderRequest) (*channel.QueryOrderResponse, error) {
	if req == nil || req.PlatformTradeNo == "" {
		return nil, fmt.Errorf("platform_trade_no is required")
	}

	session, err := p.client.V1CheckoutSessions.Retrieve(ctx, req.PlatformTradeNo, nil)
	if err != nil {
		return nil, fmt.Errorf("stripe retrieve session failed: %w", err)
	}

	status := mapCheckoutStatus(session.PaymentStatus)

	resp := &channel.QueryOrderResponse{
		Status:          status,
		PlatformTradeNo: session.ID,
	}

	if session.AmountTotal > 0 {
		resp.PaidAmount = session.AmountTotal
	}

	return resp, nil
}

// HandleWebhook 处理 Stripe Webhook
func (p *Provider) HandleWebhook(ctx context.Context, req *channel.WebhookRequest) (*channel.WebhookResponse, error) {
	if req == nil || len(req.RawBody) == 0 {
		return nil, fmt.Errorf("webhook request body is required")
	}

	sigHeader := req.Headers["Stripe-Signature"]
	if sigHeader == "" {
		return nil, fmt.Errorf("missing Stripe-Signature header")
	}

	// Verify webhook signature
	if err := verifyWebhookSignature(req.RawBody, sigHeader, p.webhookSecret); err != nil {
		return nil, fmt.Errorf("webhook signature verification failed: %w", err)
	}

	// Parse event
	var event stripe.Event
	if err := json.Unmarshal(req.RawBody, &event); err != nil {
		return nil, fmt.Errorf("failed to parse webhook event: %w", err)
	}

	switch event.Type {
	case "checkout.session.completed":
		return p.handleCheckoutCompleted(event)
	case "charge.refunded":
		return p.handleChargeRefunded(event)
	default:
		logger.Info("Stripe webhook: unhandled event type %s", event.Type)
		return &channel.WebhookResponse{
			Success:      true,
			ResponseBody: []byte(`{"received": true}`),
		}, nil
	}
}

func (p *Provider) handleCheckoutCompleted(event stripe.Event) (*channel.WebhookResponse, error) {
	var session stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
		return nil, fmt.Errorf("failed to parse checkout session: %w", err)
	}

	orderID := session.Metadata["order_id"]

	return &channel.WebhookResponse{
		Success:         true,
		OrderID:         orderID,
		PlatformTradeNo: session.ID,
		Status:          channel.OrderStatusPaid,
		PaidAmount:      session.AmountTotal,
		PaidAt:          time.Now(),
		ResponseBody:    []byte(`{"received": true}`),
	}, nil
}

func (p *Provider) handleChargeRefunded(event stripe.Event) (*channel.WebhookResponse, error) {
	// Extract charge data from raw JSON
	var chargeData map[string]interface{}
	if err := json.Unmarshal(event.Data.Raw, &chargeData); err != nil {
		return nil, fmt.Errorf("failed to parse charge data: %w", err)
	}

	chargeID, _ := chargeData["id"].(string)
	metadata, _ := chargeData["metadata"].(map[string]interface{})
	orderID, _ := metadata["order_id"].(string)

	return &channel.WebhookResponse{
		Success:         true,
		OrderID:         orderID,
		PlatformTradeNo: chargeID,
		Status:          channel.OrderStatusRefund,
		ResponseBody:    []byte(`{"received": true}`),
	}, nil
}

// Refund 发起退款
func (p *Provider) Refund(ctx context.Context, req *channel.RefundRequest) (*channel.RefundResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("refund request is required")
	}

	// Retrieve the checkout session to get the PaymentIntent
	session, err := p.client.V1CheckoutSessions.Retrieve(ctx, req.PlatformNo, nil)
	if err != nil {
		return nil, fmt.Errorf("stripe retrieve session for refund failed: %w", err)
	}

	if session.PaymentIntent == nil {
		return nil, fmt.Errorf("no payment intent found for session %s", req.PlatformNo)
	}

	refundParams := &stripe.RefundCreateParams{
		PaymentIntent: stripe.String(session.PaymentIntent.ID),
		Amount:        stripe.Int64(req.Amount),
	}
	if req.Reason != "" {
		refundParams.Reason = stripe.String(string(stripe.RefundReasonRequestedByCustomer))
	}

	refund, err := p.client.V1Refunds.Create(ctx, refundParams)
	if err != nil {
		return nil, fmt.Errorf("stripe create refund failed: %w", err)
	}

	return &channel.RefundResponse{
		RefundNo:         req.RefundNo,
		PlatformTradeNo:  session.PaymentIntent.ID,
		PlatformRefundNo: refund.ID,
		Status:           mapRefundStatus(refund.Status),
		Amount:           refund.Amount,
	}, nil
}

// QueryRefund 查询退款状态
func (p *Provider) QueryRefund(ctx context.Context, req *channel.RefundRequest) (*channel.RefundResponse, error) {
	if req == nil || req.RefundNo == "" {
		return nil, fmt.Errorf("refund_no is required")
	}

	// Use the platform refund no if available, otherwise list refunds
	refund, err := p.client.V1Refunds.Retrieve(ctx, req.RefundNo, nil)
	if err != nil {
		return nil, fmt.Errorf("stripe retrieve refund failed: %w", err)
	}

	return &channel.RefundResponse{
		RefundNo:         req.RefundNo,
		PlatformRefundNo: refund.ID,
		Status:           mapRefundStatus(refund.Status),
		Amount:           refund.Amount,
	}, nil
}

// Close 关闭资源
func (p *Provider) Close() error {
	return nil
}

// mapCheckoutStatus 映射 Stripe Checkout 支付状态
func mapCheckoutStatus(status stripe.CheckoutSessionPaymentStatus) channel.OrderStatus {
	switch status {
	case stripe.CheckoutSessionPaymentStatusPaid:
		return channel.OrderStatusPaid
	case stripe.CheckoutSessionPaymentStatusUnpaid:
		return channel.OrderStatusPending
	case stripe.CheckoutSessionPaymentStatusNoPaymentRequired:
		return channel.OrderStatusPaid
	default:
		return channel.OrderStatusPending
	}
}

// mapRefundStatus 映射 Stripe 退款状态
func mapRefundStatus(status stripe.RefundStatus) channel.RefundStatus {
	switch status {
	case stripe.RefundStatusSucceeded:
		return channel.RefundStatusSuccess
	case stripe.RefundStatusPending:
		return channel.RefundStatusProcessing
	case stripe.RefundStatusFailed:
		return channel.RefundStatusFailed
	default:
		return channel.RefundStatusPending
	}
}

// verifyWebhookSignature 验证 Stripe Webhook 签名
// Stripe 使用 HMAC-SHA256 签名，格式: t=timestamp,v1=signature
func verifyWebhookSignature(payload []byte, sigHeader, secret string) error {
	parts := strings.Split(sigHeader, ",")
	var timestamp string
	var signatures []string

	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "t":
			timestamp = kv[1]
		case "v1":
			signatures = append(signatures, kv[1])
		}
	}

	if timestamp == "" || len(signatures) == 0 {
		return fmt.Errorf("invalid signature header format")
	}

	// Check timestamp tolerance (5 minutes)
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp: %w", err)
	}
	if time.Now().Unix()-ts > 300 {
		return fmt.Errorf("webhook timestamp too old")
	}

	// Compute expected signature
	signedPayload := fmt.Sprintf("%s.%s", timestamp, string(payload))
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signedPayload))
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	for _, sig := range signatures {
		if hmac.Equal([]byte(sig), []byte(expectedSig)) {
			return nil
		}
	}

	return fmt.Errorf("signature mismatch")
}
