package wechat

import (
	"context"
	"fmt"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/jsapi"

	"gopay/pkg/channel"
	"gopay/pkg/logger"
)

// JSAPIProvider 微信 JSAPI 支付 Provider（公众号/小程序）
type JSAPIProvider struct {
	*Provider
	jsapiService *jsapi.JsapiApiService
}

// NewJSAPIProvider 创建微信 JSAPI 支付 Provider
func NewJSAPIProvider(cfg *Config) (*JSAPIProvider, error) {
	// 创建基础 Provider
	baseProvider, err := NewProvider(cfg)
	if err != nil {
		return nil, err
	}

	// 创建 JSAPI 支付服务
	jsapiService := &jsapi.JsapiApiService{Client: baseProvider.client}

	logger.Info("Wechat JSAPI pay provider initialized successfully, mchID=%s", cfg.MchID)

	return &JSAPIProvider{
		Provider:     baseProvider,
		jsapiService: jsapiService,
	}, nil
}

// Name 返回渠道名称
func (p *JSAPIProvider) Name() string {
	return "wechat_jsapi"
}

// CreateOrder 创建支付订单
func (p *JSAPIProvider) CreateOrder(ctx context.Context, req *channel.CreateOrderRequest) (*channel.CreateOrderResponse, error) {
	logger.Info("Creating wechat JSAPI order: orderID=%s, amount=%d", req.OrderID, req.Amount)

	// 验证必需参数：openid
	openid := req.ExtraData["openid"]
	if openid == "" {
		return nil, fmt.Errorf("openid is required for JSAPI payment")
	}

	appID := firstNonEmpty(req.ExtraData["app_id"], p.appID)
	if appID == "" {
		return nil, fmt.Errorf("app_id is required for JSAPI payment")
	}

	// 构建请求参数
	jsapiReq := jsapi.PrepayRequest{
		Appid:       core.String(appID),
		Mchid:       core.String(p.mchID),
		Description: core.String(req.Subject),
		OutTradeNo:  core.String(req.BizOrderNo),
		NotifyUrl:   core.String(req.NotifyURL),
		Amount: &jsapi.Amount{
			Total:    core.Int64(req.Amount),
			Currency: core.String("CNY"),
		},
		Payer: &jsapi.Payer{
			Openid: core.String(openid),
		},
	}

	// 调用微信支付 API（获取 prepay_id 和调起支付参数）
	resp, result, err := p.jsapiService.PrepayWithRequestPayment(ctx, jsapiReq)
	if err != nil {
		logger.Error("Failed to create wechat JSAPI order: %v", err)
		return nil, fmt.Errorf("wechat JSAPI prepay failed: %w", err)
	}

	// 检查响应
	if result.Response.StatusCode != 200 {
		logger.Error("Wechat JSAPI prepay failed: status=%d", result.Response.StatusCode)
		return nil, fmt.Errorf("wechat JSAPI prepay failed: status=%d", result.Response.StatusCode)
	}

	logger.Info("Wechat JSAPI order created successfully: orderID=%s, prepayID=%s", req.OrderID, *resp.PrepayId)

	// 返回 prepay_id 和调起支付所需的参数
	return &channel.CreateOrderResponse{
		PlatformTradeNo: req.BizOrderNo,
		PrepayID:        *resp.PrepayId,
		ExtraData: map[string]string{
			"prepay_id": *resp.PrepayId,
			"timestamp": *resp.TimeStamp,
			"nonce_str": *resp.NonceStr,
			"package":   *resp.Package,
			"sign_type": *resp.SignType,
			"pay_sign":  *resp.PaySign,
		},
	}, nil
}

// QueryOrder 查询订单状态
func (p *JSAPIProvider) QueryOrder(ctx context.Context, req *channel.QueryOrderRequest) (*channel.QueryOrderResponse, error) {
	// JSAPI 使用与 Native 相同的查询接口
	return p.Provider.QueryOrder(ctx, req)
}

// HandleWebhook 处理支付平台的回调通知
func (p *JSAPIProvider) HandleWebhook(ctx context.Context, req *channel.WebhookRequest) (*channel.WebhookResponse, error) {
	// JSAPI 使用与 Native 相同的回调处理逻辑
	return p.Provider.HandleWebhook(ctx, req)
}
