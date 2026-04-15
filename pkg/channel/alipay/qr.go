package alipay

import (
	"context"
	"fmt"

	"github.com/smartwalle/alipay/v3"

	"gopay/pkg/channel"
	"gopay/pkg/logger"
)

// QRProvider 支付宝扫码支付 Provider（PC 网站）
type QRProvider struct {
	*Provider
}

// NewQRProvider 创建支付宝扫码支付 Provider
func NewQRProvider(cfg *Config) (*QRProvider, error) {
	// 创建基础 Provider
	baseProvider, err := NewProvider(cfg)
	if err != nil {
		return nil, err
	}

	logger.Info("Alipay QR pay provider initialized successfully, appID=%s", cfg.AppID)

	return &QRProvider{
		Provider: baseProvider,
	}, nil
}

// Name 返回渠道名称
func (p *QRProvider) Name() string {
	return "alipay_qr"
}

// CreateOrder 创建支付订单
func (p *QRProvider) CreateOrder(ctx context.Context, req *channel.CreateOrderRequest) (*channel.CreateOrderResponse, error) {
	logger.Info("Creating alipay QR order: orderID=%s, amount=%d", req.OrderID, req.Amount)

	// 构建请求参数
	qrReq := alipay.TradePrecreate{
		Trade: alipay.Trade{
			Subject:     req.Subject,
			OutTradeNo:  req.BizOrderNo,
			TotalAmount: formatAmount(req.Amount),
			ProductCode: "FACE_TO_FACE_PAYMENT",
		},
		NotifyURL: req.NotifyURL,
	}

	// 可选参数：商品描述
	if req.Description != "" {
		qrReq.Body = req.Description
	}

	// 调用支付宝预下单接口
	resp, err := p.client.TradePrecreate(ctx, qrReq)
	if err != nil {
		logger.Error("Failed to create alipay QR order: %v", err)
		return nil, fmt.Errorf("alipay QR precreate failed: %w", err)
	}

	// 检查响应
	if !resp.IsSuccess() {
		logger.Error("Alipay QR precreate failed: code=%s, msg=%s", resp.Code, resp.Msg)
		return nil, fmt.Errorf("alipay QR precreate failed: %s - %s", resp.Code, resp.Msg)
	}

	logger.Info("Alipay QR order created successfully: orderID=%s, qrCode=%s", req.OrderID, resp.QRCode)

	// 返回二维码链接
	return &channel.CreateOrderResponse{
		PlatformTradeNo: req.BizOrderNo,
		PayURL:          resp.QRCode,
		QRCode:          resp.QRCode,
		ExtraData: map[string]string{
			"qr_code": resp.QRCode,
		},
	}, nil
}

// QueryOrder 查询订单状态
func (p *QRProvider) QueryOrder(ctx context.Context, req *channel.QueryOrderRequest) (*channel.QueryOrderResponse, error) {
	// QR 使用与基础 Provider 相同的查询接口
	return p.Provider.QueryOrder(ctx, req)
}

// HandleWebhook 处理支付平台的回调通知
func (p *QRProvider) HandleWebhook(ctx context.Context, req *channel.WebhookRequest) (*channel.WebhookResponse, error) {
	// QR 使用与基础 Provider 相同的回调处理逻辑
	return p.Provider.HandleWebhook(ctx, req)
}
