package alipay

import (
	"context"
	"fmt"

	"github.com/smartwalle/alipay/v3"

	"gopay/pkg/channel"
	"gopay/pkg/logger"
)

// FaceProvider 支付宝当面付 Provider（线下扫码）
type FaceProvider struct {
	*Provider
}

// NewFaceProvider 创建支付宝当面付 Provider
func NewFaceProvider(cfg *Config) (*FaceProvider, error) {
	// 创建基础 Provider
	baseProvider, err := NewProvider(cfg)
	if err != nil {
		return nil, err
	}

	logger.Info("Alipay Face pay provider initialized successfully, appID=%s", cfg.AppID)

	return &FaceProvider{
		Provider: baseProvider,
	}, nil
}

// Name 返回渠道名称
func (p *FaceProvider) Name() string {
	return "alipay_face"
}

// CreateOrder 创建支付订单
func (p *FaceProvider) CreateOrder(ctx context.Context, req *channel.CreateOrderRequest) (*channel.CreateOrderResponse, error) {
	logger.Info("Creating alipay Face order: orderID=%s, amount=%d", req.OrderID, req.Amount)

	// 验证必需参数：auth_code（用户付款码）
	authCode := req.ExtraData["auth_code"]
	if authCode == "" {
		return nil, fmt.Errorf("auth_code is required for Face payment")
	}

	// 构建请求参数
	faceReq := alipay.TradePay{
		Trade: alipay.Trade{
			Subject:     req.Subject,
			OutTradeNo:  req.BizOrderNo,
			TotalAmount: formatAmount(req.Amount),
			ProductCode: "FACE_TO_FACE_PAYMENT",
		},
		NotifyURL: req.NotifyURL,
		AuthCode:  authCode,
		Scene:     "bar_code", // 条码支付场景
	}

	// 可选参数：商品描述
	if req.Description != "" {
		faceReq.Body = req.Description
	}

	// 调用支付宝当面付接口
	resp, err := p.client.TradePay(ctx, faceReq)
	if err != nil {
		logger.Error("Failed to create alipay Face order: %v", err)
		return nil, fmt.Errorf("alipay Face pay failed: %w", err)
	}

	// 检查响应
	if !resp.IsSuccess() {
		logger.Error("Alipay Face pay failed: code=%s, msg=%s", resp.Code, resp.Msg)
		return nil, fmt.Errorf("alipay Face pay failed: %s - %s", resp.Code, resp.Msg)
	}

	logger.Info("Alipay Face order created successfully: orderID=%s, tradeNo=%s", req.OrderID, resp.TradeNo)

	// 当面付是同步支付，直接返回支付结果
	return &channel.CreateOrderResponse{
		PlatformTradeNo: resp.OutTradeNo,
		ExtraData: map[string]string{
			"trade_no":     resp.TradeNo,
			"trade_status": resp.TradeStatus,
			"total_amount": resp.TotalAmount,
		},
	}, nil
}

// QueryOrder 查询订单状态
func (p *FaceProvider) QueryOrder(ctx context.Context, req *channel.QueryOrderRequest) (*channel.QueryOrderResponse, error) {
	// Face 使用与基础 Provider 相同的查询接口
	return p.Provider.QueryOrder(ctx, req)
}

// HandleWebhook 处理支付平台的回调通知
func (p *FaceProvider) HandleWebhook(ctx context.Context, req *channel.WebhookRequest) (*channel.WebhookResponse, error) {
	// Face 使用与基础 Provider 相同的回调处理逻辑
	return p.Provider.HandleWebhook(ctx, req)
}
