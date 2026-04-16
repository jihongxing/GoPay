package wechat

import (
	"context"
	"crypto/rsa"
	"fmt"
	"strconv"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/option"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/native"
	"github.com/wechatpay-apiv3/wechatpay-go/services/refunddomestic"
	"github.com/wechatpay-apiv3/wechatpay-go/utils"

	"gopay/pkg/channel"
	"gopay/pkg/logger"
)

var newWechatClient = core.NewClient
var newWechatWebhookHandler = func(mchID, apiV3Key, mchCertSerialNo string, mchPrivateKey string) (wechatWebhookHandler, error) {
	return NewWebhookHandler(mchID, apiV3Key, mchCertSerialNo, mchPrivateKey)
}

type wechatWebhookHandler interface {
	HandleWebhook(ctx context.Context, req *channel.WebhookRequest) (*channel.WebhookResponse, error)
}

// Provider 微信支付 Provider
type Provider struct {
	mchID         string
	serialNo      string
	apiV3Key      string
	privateKey    *rsa.PrivateKey
	client        *core.Client
	nativeService *native.NativeApiService
	refundService *refunddomestic.RefundsApiService
}

// Config 微信支付配置
type Config struct {
	MchID          string `json:"mch_id"`           // 商户号
	SerialNo       string `json:"serial_no"`        // 证书序列号
	APIV3Key       string `json:"api_v3_key"`       // APIv3密钥
	PrivateKeyPath string `json:"private_key_path"` // 私钥文件路径
}

// NewProvider 创建微信支付 Provider
func NewProvider(cfg *Config) (*Provider, error) {
	// 加载商户私钥
	privateKey, err := loadPrivateKey(cfg.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load private key: %w", err)
	}

	// 创建微信支付客户端
	ctx := context.Background()
	opts := []core.ClientOption{
		option.WithWechatPayAutoAuthCipher(cfg.MchID, cfg.SerialNo, privateKey, cfg.APIV3Key),
	}

	client, err := newWechatClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create wechat pay client: %w", err)
	}

	// 创建 Native 支付服务
	nativeService := native.NativeApiService{Client: client}
	refundService := refunddomestic.RefundsApiService{Client: client}

	logger.Info("Wechat pay provider initialized successfully, mchID=%s", cfg.MchID)

	return &Provider{
		mchID:         cfg.MchID,
		serialNo:      cfg.SerialNo,
		apiV3Key:      cfg.APIV3Key,
		privateKey:    privateKey,
		client:        client,
		nativeService: &nativeService,
		refundService: &refundService,
	}, nil
}

// Name 返回渠道名称
func (p *Provider) Name() string {
	return "wechat_native"
}

// CreateOrder 创建支付订单
func (p *Provider) CreateOrder(ctx context.Context, req *channel.CreateOrderRequest) (*channel.CreateOrderResponse, error) {
	logger.Info("Creating wechat native order: orderID=%s, amount=%d", req.OrderID, req.Amount)

	// 构建请求参数
	nativeReq := native.PrepayRequest{
		Appid:       core.String(req.ExtraData["app_id"]), // 微信公众号/小程序 AppID
		Mchid:       core.String(p.mchID),
		Description: core.String(req.Subject),
		OutTradeNo:  core.String(req.BizOrderNo),
		NotifyUrl:   core.String(req.NotifyURL),
		Amount: &native.Amount{
			Total:    core.Int64(req.Amount),
			Currency: core.String("CNY"),
		},
	}

	// 调用微信支付 API
	resp, result, err := p.nativeService.Prepay(ctx, nativeReq)
	if err != nil {
		logger.Error("Failed to create wechat order: %v", err)
		return nil, fmt.Errorf("wechat prepay failed: %w", err)
	}

	// 检查响应
	if result.Response.StatusCode != 200 {
		logger.Error("Wechat prepay failed: status=%d", result.Response.StatusCode)
		return nil, fmt.Errorf("wechat prepay failed: status=%d", result.Response.StatusCode)
	}

	logger.Info("Wechat order created successfully: orderID=%s, codeUrl=%s", req.OrderID, *resp.CodeUrl)

	return &channel.CreateOrderResponse{
		PlatformTradeNo: req.BizOrderNo, // 微信使用商户订单号作为唯一标识
		PayURL:          *resp.CodeUrl,
		QRCode:          *resp.CodeUrl,
		ExtraData:       map[string]string{},
	}, nil
}

// QueryOrder 查询订单状态
func (p *Provider) QueryOrder(ctx context.Context, req *channel.QueryOrderRequest) (*channel.QueryOrderResponse, error) {
	logger.Info("Querying wechat order: orderID=%s", req.OrderID)

	// 使用商户订单号查询
	queryReq := native.QueryOrderByOutTradeNoRequest{
		OutTradeNo: core.String(req.PlatformTradeNo),
		Mchid:      core.String(p.mchID),
	}

	resp, result, err := p.nativeService.QueryOrderByOutTradeNo(ctx, queryReq)
	if err != nil {
		logger.Error("Failed to query wechat order: %v", err)
		return nil, fmt.Errorf("wechat query order failed: %w", err)
	}

	if result.Response.StatusCode != 200 {
		logger.Error("Wechat query order failed: status=%d", result.Response.StatusCode)
		return nil, fmt.Errorf("wechat query order failed: status=%d", result.Response.StatusCode)
	}

	// 转换订单状态
	status := convertTradeState(*resp.TradeState)

	queryResp := &channel.QueryOrderResponse{
		Status:          status,
		PlatformTradeNo: *resp.OutTradeNo,
		ExtraData:       map[string]string{},
	}

	// 如果已支付，填充支付信息
	if status == channel.OrderStatusPaid && resp.SuccessTime != nil {
		paidAt, _ := parseWechatTime(*resp.SuccessTime)
		queryResp.PaidAt = &paidAt
		if resp.Amount != nil && resp.Amount.PayerTotal != nil {
			queryResp.PaidAmount = *resp.Amount.PayerTotal
		}
	}

	logger.Info("Wechat order queried: orderID=%s, status=%s", req.OrderID, status)

	return queryResp, nil
}

// HandleWebhook 处理支付平台的回调通知
func (p *Provider) HandleWebhook(ctx context.Context, req *channel.WebhookRequest) (*channel.WebhookResponse, error) {
	logger.Info("Handling wechat webhook, body length=%d", len(req.RawBody))

	// 使用 WebhookHandler 处理（使用官方 SDK 验证签名）
	handler, err := newWechatWebhookHandler(p.mchID, p.apiV3Key, p.serialNo, "")
	if err != nil {
		logger.Error("Failed to create webhook handler: %v", err)
		return &channel.WebhookResponse{
			Success:      false,
			ResponseBody: []byte(`{"code":"FAIL","message":"系统错误"}`),
		}, err
	}

	return handler.HandleWebhook(ctx, req)
}

// Refund 发起退款
func (p *Provider) Refund(ctx context.Context, req *channel.RefundRequest) (*channel.RefundResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("refund request is required")
	}

	amount := req.Amount
	total := req.Amount
	refundReq := refunddomestic.CreateRequest{
		OutTradeNo:  &req.PlatformNo,
		OutRefundNo: &req.RefundNo,
		Reason:      &req.Reason,
		Amount: &refunddomestic.AmountReq{
			Refund:   &amount,
			Total:    &total,
			Currency: core.String("CNY"),
		},
	}
	if req.NotifyURL != "" {
		refundReq.NotifyUrl = &req.NotifyURL
	}

	resp, result, err := p.refundService.Create(ctx, refundReq)
	if err != nil {
		return nil, fmt.Errorf("wechat refund failed: %w", err)
	}
	if result != nil && result.Response != nil && result.Response.StatusCode >= 400 {
		return nil, fmt.Errorf("wechat refund failed: status=%d", result.Response.StatusCode)
	}

	refundResp := &channel.RefundResponse{
		RefundNo:        req.RefundNo,
		PlatformTradeNo: req.PlatformNo,
		Amount:          req.Amount,
		ExtraData:       map[string]string{},
	}
	if resp != nil {
		if resp.RefundId != nil {
			refundResp.PlatformRefundNo = *resp.RefundId
		}
		if resp.OutRefundNo != nil {
			refundResp.RefundNo = *resp.OutRefundNo
		}
		if resp.OutTradeNo != nil {
			refundResp.PlatformTradeNo = *resp.OutTradeNo
		}
		if resp.SuccessTime != nil {
			refundResp.RefundedAt = resp.SuccessTime
		}
		if resp.Status != nil {
			refundResp.Status = convertRefundStatus(string(*resp.Status))
		}
		if resp.Amount != nil && resp.Amount.Refund != nil {
			refundResp.Amount = *resp.Amount.Refund
		}
		if resp.Amount != nil && resp.Amount.Total != nil {
			refundResp.ExtraData["total"] = strconv.FormatInt(*resp.Amount.Total, 10)
		}
	}

	return refundResp, nil
}

// QueryRefund 查询退款
func (p *Provider) QueryRefund(ctx context.Context, req *channel.RefundRequest) (*channel.RefundResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("refund request is required")
	}

	queryReq := refunddomestic.QueryByOutRefundNoRequest{
		OutRefundNo: &req.RefundNo,
	}
	resp, result, err := p.refundService.QueryByOutRefundNo(ctx, queryReq)
	if err != nil {
		return nil, fmt.Errorf("wechat refund query failed: %w", err)
	}
	if result != nil && result.Response != nil && result.Response.StatusCode >= 400 {
		return nil, fmt.Errorf("wechat refund query failed: status=%d", result.Response.StatusCode)
	}

	refundResp := &channel.RefundResponse{
		RefundNo:        req.RefundNo,
		PlatformTradeNo: req.PlatformNo,
		ExtraData:       map[string]string{},
	}
	if resp != nil {
		if resp.RefundId != nil {
			refundResp.PlatformRefundNo = *resp.RefundId
		}
		if resp.OutRefundNo != nil {
			refundResp.RefundNo = *resp.OutRefundNo
		}
		if resp.OutTradeNo != nil {
			refundResp.PlatformTradeNo = *resp.OutTradeNo
		}
		if resp.SuccessTime != nil {
			refundResp.RefundedAt = resp.SuccessTime
		}
		if resp.Status != nil {
			refundResp.Status = convertRefundStatus(string(*resp.Status))
		}
		if resp.Amount != nil && resp.Amount.Refund != nil {
			refundResp.Amount = *resp.Amount.Refund
		}
	}

	return refundResp, nil
}

// Close 关闭资源
func (p *Provider) Close() error {
	logger.Info("Closing wechat pay provider")
	return nil
}

// loadPrivateKey 加载商户私钥
func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	return utils.LoadPrivateKeyWithPath(path)
}

// convertTradeState 转换微信交易状态到内部状态
func convertTradeState(state string) channel.OrderStatus {
	switch state {
	case "SUCCESS":
		return channel.OrderStatusPaid
	case "NOTPAY":
		return channel.OrderStatusPending
	case "CLOSED":
		return channel.OrderStatusClosed
	case "REVOKED":
		return channel.OrderStatusClosed
	case "USERPAYING":
		return channel.OrderStatusPending
	case "PAYERROR":
		return channel.OrderStatusClosed
	case "REFUND":
		return channel.OrderStatusRefund
	default:
		return channel.OrderStatusPending
	}
}

func convertRefundStatus(status string) channel.RefundStatus {
	switch status {
	case "SUCCESS":
		return channel.RefundStatusSuccess
	case "PROCESSING":
		return channel.RefundStatusProcessing
	case "CLOSED":
		return channel.RefundStatusFailed
	case "ABNORMAL":
		return channel.RefundStatusFailed
	default:
		return channel.RefundStatusPending
	}
}
