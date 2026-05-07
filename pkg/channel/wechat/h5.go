package wechat

import (
	"context"
	"fmt"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/h5"

	"gopay/pkg/channel"
	"gopay/pkg/logger"
)

// H5Provider 微信 H5 支付 Provider（手机网页）
type H5Provider struct {
	*Provider
	h5Service *h5.H5ApiService
}

// NewH5Provider 创建微信 H5 支付 Provider
func NewH5Provider(cfg *Config) (*H5Provider, error) {
	// 创建基础 Provider
	baseProvider, err := NewProvider(cfg)
	if err != nil {
		return nil, err
	}

	// 创建 H5 支付服务
	h5Service := &h5.H5ApiService{Client: baseProvider.client}

	logger.Info("Wechat H5 pay provider initialized successfully, mchID=%s", cfg.MchID)

	return &H5Provider{
		Provider:  baseProvider,
		h5Service: h5Service,
	}, nil
}

// Name 返回渠道名称
func (p *H5Provider) Name() string {
	return "wechat_h5"
}

// CreateOrder 创建支付订单
func (p *H5Provider) CreateOrder(ctx context.Context, req *channel.CreateOrderRequest) (*channel.CreateOrderResponse, error) {
	logger.Info("Creating wechat H5 order: orderID=%s, amount=%d", req.OrderID, req.Amount)

	// 验证必需参数
	appID := firstNonEmpty(req.ExtraData["app_id"], p.appID)
	if appID == "" {
		return nil, fmt.Errorf("app_id is required for H5 payment")
	}

	clientIP := req.ExtraData["client_ip"]
	if clientIP == "" {
		clientIP = "127.0.0.1" // 默认值
	}

	// H5 场景类型：iOS, Android, Wap
	h5Type := req.ExtraData["h5_type"]
	if h5Type == "" {
		h5Type = "Wap" // 默认为 Wap
	}

	// 构建请求参数
	h5Req := h5.PrepayRequest{
		Appid:       core.String(appID),
		Mchid:       core.String(p.mchID),
		Description: core.String(req.Subject),
		OutTradeNo:  core.String(req.BizOrderNo),
		NotifyUrl:   core.String(req.NotifyURL),
		Amount: &h5.Amount{
			Total:    core.Int64(req.Amount),
			Currency: core.String("CNY"),
		},
		SceneInfo: &h5.SceneInfo{
			PayerClientIp: core.String(clientIP),
			H5Info: &h5.H5Info{
				Type: core.String(h5Type),
			},
		},
	}

	// 调用微信支付 API
	resp, result, err := p.h5Service.Prepay(ctx, h5Req)
	if err != nil {
		logger.Error("Failed to create wechat H5 order: %v", err)
		return nil, fmt.Errorf("wechat H5 prepay failed: %w", err)
	}

	// 检查响应
	if result.Response.StatusCode != 200 {
		logger.Error("Wechat H5 prepay failed: status=%d", result.Response.StatusCode)
		return nil, fmt.Errorf("wechat H5 prepay failed: status=%d", result.Response.StatusCode)
	}

	logger.Info("Wechat H5 order created successfully: orderID=%s, h5Url=%s", req.OrderID, *resp.H5Url)

	// 返回 H5 支付链接
	return &channel.CreateOrderResponse{
		PlatformTradeNo: req.BizOrderNo,
		PayURL:          *resp.H5Url,
		ExtraData: map[string]string{
			"h5_url": *resp.H5Url,
		},
	}, nil
}

// QueryOrder 查询订单状态
func (p *H5Provider) QueryOrder(ctx context.Context, req *channel.QueryOrderRequest) (*channel.QueryOrderResponse, error) {
	// H5 使用与 Native 相同的查询接口
	return p.Provider.QueryOrder(ctx, req)
}

// HandleWebhook 处理支付平台的回调通知
func (p *H5Provider) HandleWebhook(ctx context.Context, req *channel.WebhookRequest) (*channel.WebhookResponse, error) {
	// H5 使用与 Native 相同的回调处理逻辑
	return p.Provider.HandleWebhook(ctx, req)
}
