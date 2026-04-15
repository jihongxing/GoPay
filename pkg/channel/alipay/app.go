package alipay

import (
	"context"
	"fmt"

	"github.com/smartwalle/alipay/v3"

	"gopay/pkg/channel"
	"gopay/pkg/logger"
)

// AppProvider 支付宝 APP 支付 Provider（原生应用）
type AppProvider struct {
	*Provider
}

// NewAppProvider 创建支付宝 APP 支付 Provider
func NewAppProvider(cfg *Config) (*AppProvider, error) {
	// 创建基础 Provider
	baseProvider, err := NewProvider(cfg)
	if err != nil {
		return nil, err
	}

	logger.Info("Alipay APP pay provider initialized successfully, appID=%s", cfg.AppID)

	return &AppProvider{
		Provider: baseProvider,
	}, nil
}

// Name 返回渠道名称
func (p *AppProvider) Name() string {
	return "alipay_app"
}

// CreateOrder 创建支付订单
func (p *AppProvider) CreateOrder(ctx context.Context, req *channel.CreateOrderRequest) (*channel.CreateOrderResponse, error) {
	logger.Info("Creating alipay APP order: orderID=%s, amount=%d", req.OrderID, req.Amount)

	// 构建请求参数
	appReq := alipay.TradeAppPay{
		Trade: alipay.Trade{
			Subject:     req.Subject,
			OutTradeNo:  req.BizOrderNo,
			TotalAmount: formatAmount(req.Amount),
			ProductCode: "QUICK_MSECURITY_PAY",
		},
		NotifyURL: req.NotifyURL,
	}

	// 可选参数：商品描述
	if req.Description != "" {
		appReq.Body = req.Description
	}

	// 调用支付宝 APP 支付接口
	payParam, err := p.client.TradeAppPay(appReq)
	if err != nil {
		logger.Error("Failed to create alipay APP order: %v", err)
		return nil, fmt.Errorf("alipay APP pay failed: %w", err)
	}

	logger.Info("Alipay APP order created successfully: orderID=%s", req.OrderID)

	// 返回 APP 调起支付参数
	return &channel.CreateOrderResponse{
		PlatformTradeNo: req.BizOrderNo,
		PrepayID:        req.BizOrderNo,
		ExtraData: map[string]string{
			"order_string": payParam, // APP 端需要的完整支付参数字符串
		},
	}, nil
}

// QueryOrder 查询订单状态
func (p *AppProvider) QueryOrder(ctx context.Context, req *channel.QueryOrderRequest) (*channel.QueryOrderResponse, error) {
	// APP 使用与基础 Provider 相同的查询接口
	return p.Provider.QueryOrder(ctx, req)
}

// HandleWebhook 处理支付平台的回调通知
func (p *AppProvider) HandleWebhook(ctx context.Context, req *channel.WebhookRequest) (*channel.WebhookResponse, error) {
	// APP 使用与基础 Provider 相同的回调处理逻辑
	return p.Provider.HandleWebhook(ctx, req)
}
