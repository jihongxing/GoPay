package wechat

import (
	"context"
	"fmt"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/app"

	"gopay/pkg/channel"
	"gopay/pkg/logger"
)

// AppProvider 微信 APP 支付 Provider（原生应用）
type AppProvider struct {
	*Provider
	appService *app.AppApiService
}

// NewAppProvider 创建微信 APP 支付 Provider
func NewAppProvider(cfg *Config) (*AppProvider, error) {
	// 创建基础 Provider
	baseProvider, err := NewProvider(cfg)
	if err != nil {
		return nil, err
	}

	// 创建 APP 支付服务
	appService := &app.AppApiService{Client: baseProvider.client}

	logger.Info("Wechat APP pay provider initialized successfully, mchID=%s", cfg.MchID)

	return &AppProvider{
		Provider:   baseProvider,
		appService: appService,
	}, nil
}

// Name 返回渠道名称
func (p *AppProvider) Name() string {
	return "wechat_app"
}

// CreateOrder 创建支付订单
func (p *AppProvider) CreateOrder(ctx context.Context, req *channel.CreateOrderRequest) (*channel.CreateOrderResponse, error) {
	logger.Info("Creating wechat APP order: orderID=%s, amount=%d", req.OrderID, req.Amount)

	// 验证必需参数
	appID := firstNonEmpty(req.ExtraData["app_id"], p.appID)
	if appID == "" {
		return nil, fmt.Errorf("app_id is required for APP payment")
	}

	// 构建请求参数
	appReq := app.PrepayRequest{
		Appid:       core.String(appID),
		Mchid:       core.String(p.mchID),
		Description: core.String(req.Subject),
		OutTradeNo:  core.String(req.BizOrderNo),
		NotifyUrl:   core.String(req.NotifyURL),
		Amount: &app.Amount{
			Total:    core.Int64(req.Amount),
			Currency: core.String("CNY"),
		},
	}

	// 可选参数：场景信息
	if deviceID := req.ExtraData["device_id"]; deviceID != "" {
		appReq.SceneInfo = &app.SceneInfo{
			DeviceId: core.String(deviceID),
		}
	}

	// 调用微信支付 API（获取 prepay_id 和调起支付参数）
	resp, result, err := p.appService.PrepayWithRequestPayment(ctx, appReq)
	if err != nil {
		logger.Error("Failed to create wechat APP order: %v", err)
		return nil, fmt.Errorf("wechat APP prepay failed: %w", err)
	}

	// 检查响应
	if result.Response.StatusCode != 200 {
		logger.Error("Wechat APP prepay failed: status=%d", result.Response.StatusCode)
		return nil, fmt.Errorf("wechat APP prepay failed: status=%d", result.Response.StatusCode)
	}

	logger.Info("Wechat APP order created successfully: orderID=%s, prepayID=%s", req.OrderID, *resp.PrepayId)

	// 返回 prepay_id 和调起支付所需的参数
	return &channel.CreateOrderResponse{
		PlatformTradeNo: req.BizOrderNo,
		PrepayID:        *resp.PrepayId,
		ExtraData: map[string]string{
			"prepay_id":  *resp.PrepayId,
			"partner_id": *resp.PartnerId,
			"package":    *resp.Package,
			"timestamp":  *resp.TimeStamp,
			"nonce_str":  *resp.NonceStr,
			"sign":       *resp.Sign,
		},
	}, nil
}

// QueryOrder 查询订单状态
func (p *AppProvider) QueryOrder(ctx context.Context, req *channel.QueryOrderRequest) (*channel.QueryOrderResponse, error) {
	// APP 使用与 Native 相同的查询接口
	return p.Provider.QueryOrder(ctx, req)
}

// HandleWebhook 处理支付平台的回调通知
func (p *AppProvider) HandleWebhook(ctx context.Context, req *channel.WebhookRequest) (*channel.WebhookResponse, error) {
	// APP 使用与 Native 相同的回调处理逻辑
	return p.Provider.HandleWebhook(ctx, req)
}
