package alipay

import (
	"context"
	"fmt"

	"github.com/smartwalle/alipay/v3"

	"gopay/pkg/channel"
	"gopay/pkg/logger"
)

// WapProvider 支付宝手机网站支付 Provider（H5）
type WapProvider struct {
	*Provider
}

// NewWapProvider 创建支付宝手机网站支付 Provider
func NewWapProvider(cfg *Config) (*WapProvider, error) {
	// 创建基础 Provider
	baseProvider, err := NewProvider(cfg)
	if err != nil {
		return nil, err
	}

	logger.Info("Alipay Wap pay provider initialized successfully, appID=%s", cfg.AppID)

	return &WapProvider{
		Provider: baseProvider,
	}, nil
}

// Name 返回渠道名称
func (p *WapProvider) Name() string {
	return "alipay_wap"
}

// CreateOrder 创建支付订单
func (p *WapProvider) CreateOrder(ctx context.Context, req *channel.CreateOrderRequest) (*channel.CreateOrderResponse, error) {
	logger.Info("Creating alipay Wap order: orderID=%s, amount=%d", req.OrderID, req.Amount)

	// 构建请求参数
	wapReq := alipay.TradeWapPay{
		Trade: alipay.Trade{
			Subject:     req.Subject,
			OutTradeNo:  req.BizOrderNo,
			TotalAmount: formatAmount(req.Amount),
			ProductCode: "QUICK_WAP_WAY",
		},
		NotifyURL: req.NotifyURL,
	}

	// 可选参数：商品描述
	if req.Description != "" {
		wapReq.Body = req.Description
	}

	// 可选参数：退出 URL
	if quitURL := req.ExtraData["quit_url"]; quitURL != "" {
		wapReq.QuitURL = quitURL
	}

	// 可选参数：返回 URL
	if returnURL := req.ExtraData["return_url"]; returnURL != "" {
		wapReq.ReturnURL = returnURL
	}

	// 调用支付宝手机网站支付接口
	payURL, err := p.client.TradeWapPay(wapReq)
	if err != nil {
		logger.Error("Failed to create alipay Wap order: %v", err)
		return nil, fmt.Errorf("alipay Wap pay failed: %w", err)
	}

	logger.Info("Alipay Wap order created successfully: orderID=%s, payURL=%s", req.OrderID, payURL.String())

	// 返回支付链接
	return &channel.CreateOrderResponse{
		PlatformTradeNo: req.BizOrderNo,
		PayURL:          payURL.String(),
		ExtraData: map[string]string{
			"pay_url": payURL.String(),
		},
	}, nil
}

// QueryOrder 查询订单状态
func (p *WapProvider) QueryOrder(ctx context.Context, req *channel.QueryOrderRequest) (*channel.QueryOrderResponse, error) {
	// Wap 使用与基础 Provider 相同的查询接口
	return p.Provider.QueryOrder(ctx, req)
}

// HandleWebhook 处理支付平台的回调通知
func (p *WapProvider) HandleWebhook(ctx context.Context, req *channel.WebhookRequest) (*channel.WebhookResponse, error) {
	// Wap 使用与基础 Provider 相同的回调处理逻辑
	return p.Provider.HandleWebhook(ctx, req)
}
