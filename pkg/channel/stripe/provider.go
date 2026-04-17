package stripe

import (
	"context"
	"fmt"

	"gopay/pkg/channel"
	"gopay/pkg/logger"
)

// Config Stripe 配置
type Config struct {
	SecretKey     string `json:"secret_key"`     // Stripe Secret Key (sk_live_xxx)
	WebhookSecret string `json:"webhook_secret"` // Webhook 签名密钥 (whsec_xxx)
	Currency      string `json:"currency"`       // 默认货币 (usd, cny 等)
}

// Provider Stripe 支付 Provider
type Provider struct {
	secretKey     string
	webhookSecret string
	currency      string
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

	logger.Info("Stripe provider initialized, currency=%s", currency)

	return &Provider{
		secretKey:     cfg.SecretKey,
		webhookSecret: cfg.WebhookSecret,
		currency:      currency,
	}, nil
}

// Name 返回渠道名称
func (p *Provider) Name() string {
	return "stripe"
}

// CreateOrder 创建支付订单（Stripe Checkout Session）
func (p *Provider) CreateOrder(ctx context.Context, req *channel.CreateOrderRequest) (*channel.CreateOrderResponse, error) {
	// TODO: 集成 Stripe Go SDK (github.com/stripe/stripe-go/v76)
	// 创建 Checkout Session 并返回支付链接
	return nil, fmt.Errorf("stripe CreateOrder not yet implemented")
}

// QueryOrder 查询订单状态
func (p *Provider) QueryOrder(ctx context.Context, req *channel.QueryOrderRequest) (*channel.QueryOrderResponse, error) {
	// TODO: 通过 Stripe API 查询 PaymentIntent 状态
	return nil, fmt.Errorf("stripe QueryOrder not yet implemented")
}

// HandleWebhook 处理 Stripe Webhook
func (p *Provider) HandleWebhook(ctx context.Context, req *channel.WebhookRequest) (*channel.WebhookResponse, error) {
	// TODO: 验证 Stripe webhook 签名并解析事件
	return nil, fmt.Errorf("stripe HandleWebhook not yet implemented")
}

// Refund 发起退款
func (p *Provider) Refund(ctx context.Context, req *channel.RefundRequest) (*channel.RefundResponse, error) {
	// TODO: 通过 Stripe API 创建退款
	return nil, fmt.Errorf("stripe Refund not yet implemented")
}

// QueryRefund 查询退款状态
func (p *Provider) QueryRefund(ctx context.Context, req *channel.RefundRequest) (*channel.RefundResponse, error) {
	// TODO: 通过 Stripe API 查询退款状态
	return nil, fmt.Errorf("stripe QueryRefund not yet implemented")
}

// Close 关闭资源
func (p *Provider) Close() error {
	return nil
}
