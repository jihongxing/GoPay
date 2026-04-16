package alipay

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/smartwalle/alipay/v3"

	"gopay/pkg/channel"
	"gopay/pkg/logger"
)

// Provider 支付宝基础 Provider
type Provider struct {
	appID          string
	privateKey     string
	alipayPublicKey string
	isProduction   bool
	client         *alipay.Client
}

// Config 支付宝配置
type Config struct {
	AppID           string `json:"app_id"`            // 应用 ID
	PrivateKey      string `json:"private_key"`       // 应用私钥
	AlipayPublicKey string `json:"alipay_public_key"` // 支付宝公钥
	IsProduction    bool   `json:"is_production"`     // 是否生产环境
}

// NewProvider 创建支付宝基础 Provider
func NewProvider(cfg *Config) (*Provider, error) {
	// 创建支付宝客户端
	client, err := alipay.New(cfg.AppID, cfg.PrivateKey, cfg.IsProduction)
	if err != nil {
		return nil, fmt.Errorf("failed to create alipay client: %w", err)
	}

	// 加载支付宝公钥
	if err := client.LoadAliPayPublicKey(cfg.AlipayPublicKey); err != nil {
		return nil, fmt.Errorf("failed to load alipay public key: %w", err)
	}

	logger.Info("Alipay provider initialized successfully, appID=%s, isProduction=%v", cfg.AppID, cfg.IsProduction)

	return &Provider{
		appID:           cfg.AppID,
		privateKey:      cfg.PrivateKey,
		alipayPublicKey: cfg.AlipayPublicKey,
		isProduction:    cfg.IsProduction,
		client:          client,
	}, nil
}

// Name 返回渠道名称
func (p *Provider) Name() string {
	return "alipay"
}

// QueryOrder 查询订单状态（通用方法）
func (p *Provider) QueryOrder(ctx context.Context, req *channel.QueryOrderRequest) (*channel.QueryOrderResponse, error) {
	logger.Info("Querying alipay order: orderID=%s", req.OrderID)

	// 构建查询请求
	queryReq := alipay.TradeQuery{
		OutTradeNo: req.PlatformTradeNo,
	}

	// 调用支付宝查询接口
	resp, err := p.client.TradeQuery(ctx, queryReq)
	if err != nil {
		logger.Error("Failed to query alipay order: %v", err)
		return nil, fmt.Errorf("alipay query order failed: %w", err)
	}

	// 检查响应
	if !resp.IsSuccess() {
		logger.Error("Alipay query order failed: code=%s, msg=%s", resp.Code, resp.Msg)
		return nil, fmt.Errorf("alipay query order failed: %s - %s", resp.Code, resp.Msg)
	}

	// 转换订单状态
	status := convertTradeStatus(string(resp.TradeStatus))

	queryResp := &channel.QueryOrderResponse{
		Status:          status,
		PlatformTradeNo: resp.OutTradeNo,
		ExtraData:       map[string]string{},
	}

	// 如果已支付，填充支付信息
	if status == channel.OrderStatusPaid {
		// 解析支付时间
		if resp.SendPayDate != "" {
			paidAt, err := time.Parse("2006-01-02 15:04:05", resp.SendPayDate)
			if err == nil {
				queryResp.PaidAt = &paidAt
			}
		}
		// 支付宝金额单位是元，需要转换为分
		if resp.TotalAmount != "" {
			amount := parseAmount(resp.TotalAmount)
			queryResp.PaidAmount = amount
		}
	}

	logger.Info("Alipay order queried: orderID=%s, status=%s", req.OrderID, status)

	return queryResp, nil
}

// HandleWebhook 处理支付平台的回调通知（通用方法）
func (p *Provider) HandleWebhook(ctx context.Context, req *channel.WebhookRequest) (*channel.WebhookResponse, error) {
	logger.Info("Handling alipay webhook, body length=%d", len(req.RawBody))

	// 解析 URL 参数
	values, err := url.ParseQuery(string(req.RawBody))
	if err != nil {
		logger.Error("Failed to parse alipay notification: %v", err)
		return &channel.WebhookResponse{
			Success:      false,
			ResponseBody: []byte("failure"),
		}, nil
	}

	// 解析回调通知
	notification, err := p.client.DecodeNotification(values)
	if err != nil {
		logger.Error("Failed to decode alipay notification: %v", err)
		return &channel.WebhookResponse{
			Success:      false,
			ResponseBody: []byte("failure"),
		}, nil
	}

	// 验证签名
	if err := p.client.VerifySign(values); err != nil {
		logger.Error("Failed to verify alipay signature: %v", err)
		return &channel.WebhookResponse{
			Success:      false,
			ResponseBody: []byte("failure"),
		}, nil
	}

	// 获取订单号和交易状态
	outTradeNo := notification.OutTradeNo
	tradeStatus := notification.TradeStatus

	// 转换订单状态
	status := convertTradeStatus(string(tradeStatus))

	resp := &channel.WebhookResponse{
		Success:         true,
		PlatformTradeNo: outTradeNo,
		Status:          status,
		ResponseBody:    []byte("success"),
	}

	// 如果已支付，填充支付信息
	if status == channel.OrderStatusPaid {
		// 解析支付时间
		if notification.GmtPayment != "" {
			paidAt, err := time.Parse("2006-01-02 15:04:05", notification.GmtPayment)
			if err == nil {
				resp.PaidAt = paidAt
			}
		}
		// 支付宝金额单位是元，需要转换为分
		if notification.TotalAmount != "" {
			amount := parseAmount(notification.TotalAmount)
			resp.PaidAmount = amount
		}
	}

	logger.Info("Alipay webhook handled: tradeNo=%s, status=%s", outTradeNo, status)

	return resp, nil
}

// Close 关闭资源
func (p *Provider) Close() error {
	logger.Info("Closing alipay provider")
	return nil
}

// convertTradeStatus 转换支付宝交易状态到内部状态
func convertTradeStatus(status string) channel.OrderStatus {
	switch status {
	case "TRADE_SUCCESS":
		return channel.OrderStatusPaid
	case "TRADE_FINISHED":
		return channel.OrderStatusPaid
	case "WAIT_BUYER_PAY":
		return channel.OrderStatusPending
	case "TRADE_CLOSED":
		return channel.OrderStatusClosed
	default:
		return channel.OrderStatusPending
	}
}

// parseAmount 解析金额（元转分）
func parseAmount(amountStr string) int64 {
	// 支付宝金额单位是元，需要转换为分
	var amount float64
	fmt.Sscanf(amountStr, "%f", &amount)
	return int64(amount * 100)
}

// formatAmount 格式化金额（分转元）
func formatAmount(amountCents int64) string {
	return fmt.Sprintf("%.2f", float64(amountCents)/100.0)
}
