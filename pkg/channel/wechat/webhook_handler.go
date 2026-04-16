package wechat

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/wechatpay-apiv3/wechatpay-go/core/auth/verifiers"
	"github.com/wechatpay-apiv3/wechatpay-go/core/downloader"
	"github.com/wechatpay-apiv3/wechatpay-go/core/notify"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
	"github.com/wechatpay-apiv3/wechatpay-go/services/refunddomestic"
	"gopay/pkg/channel"
	"gopay/pkg/logger"
)

// WebhookHandler 处理微信支付 Webhook
type WebhookHandler struct {
	handler *notify.Handler // 官方 SDK 的通知处理器
}

// NewWebhookHandler 创建 Webhook 处理器
// 使用微信支付官方 SDK 进行签名验证和解密
func NewWebhookHandler(mchID, apiV3Key, mchCertSerialNo string, mchPrivateKey string) (*WebhookHandler, error) {
	// 1. 使用「证书下载器」下载微信支付平台证书
	certDownloader := downloader.MgrInstance().GetCertificateVisitor(mchID)

	// 2. 使用「证书校验器」校验微信支付应答签名
	verifier := verifiers.NewSHA256WithRSAVerifier(certDownloader)

	// 3. 创建通知处理器
	handler, err := notify.NewRSANotifyHandler(apiV3Key, verifier)
	if err != nil {
		logger.Error("Failed to create notify handler: %v", err)
		return nil, err
	}

	return &WebhookHandler{
		handler: handler,
	}, nil
}

// HandleWebhook 处理 Webhook 回调
func (h *WebhookHandler) HandleWebhook(ctx context.Context, req *channel.WebhookRequest) (*channel.WebhookResponse, error) {
	logger.Info("Handling wechat webhook")

	// 构建标准 HTTP 请求对象，设置请求体
	httpReq, err := http.NewRequestWithContext(ctx, "POST", "", bytes.NewReader(req.RawBody))
	if err != nil {
		logger.Error("Failed to create http request: %v", err)
		return &channel.WebhookResponse{
			Success:      false,
			ResponseBody: []byte(`{"code":"FAIL","message":"请求构建失败"}`),
		}, nil
	}

	// 设置请求头（微信支付签名验证需要）
	httpReq.Header.Set("Wechatpay-Timestamp", req.Headers["Wechatpay-Timestamp"])
	httpReq.Header.Set("Wechatpay-Nonce", req.Headers["Wechatpay-Nonce"])
	httpReq.Header.Set("Wechatpay-Signature", req.Headers["Wechatpay-Signature"])
	httpReq.Header.Set("Wechatpay-Serial", req.Headers["Wechatpay-Serial"])

	var envelope struct {
		EventType string `json:"event_type"`
	}
	if err := json.Unmarshal(req.RawBody, &envelope); err != nil {
		logger.Error("Failed to inspect wechat webhook event: %v", err)
		return &channel.WebhookResponse{
			Success:      false,
			ResponseBody: []byte(`{"code":"FAIL","message":"签名验证失败"}`),
		}, nil
	}

	if strings.Contains(strings.ToUpper(envelope.EventType), "REFUND") {
		refund := new(refunddomestic.Refund)
		_, err = h.handler.ParseNotifyRequest(ctx, httpReq, refund)
		if err != nil {
			logger.Error("Wechat refund webhook verification failed: %v", err)
			return &channel.WebhookResponse{
				Success:      false,
				ResponseBody: []byte(`{"code":"FAIL","message":"签名验证失败"}`),
			}, nil
		}

		resp := &channel.WebhookResponse{
			Success:         true,
			PlatformTradeNo: valueOrEmpty(refund.OutTradeNo),
			Status:          channel.OrderStatusRefund,
			ResponseBody:    []byte(`{"code":"SUCCESS","message":"成功"}`),
		}
		if refund.SuccessTime != nil {
			resp.PaidAt = *refund.SuccessTime
		} else if refund.CreateTime != nil {
			resp.PaidAt = *refund.CreateTime
		}
		if refund.Amount != nil && refund.Amount.Refund != nil {
			resp.PaidAmount = *refund.Amount.Refund
		}
		logger.Info("Wechat refund webhook processed: outTradeNo=%s", resp.PlatformTradeNo)
		return resp, nil
	}

	// 使用官方 SDK 验证签名并解密
	// ParseNotifyRequest 会自动：
	// 1. 验证 RSA-SHA256 签名（使用微信平台证书公钥）
	// 2. 验证时间戳（防重放攻击）
	// 3. 解密 AES-256-GCM 加密的资源内容
	transaction := new(payments.Transaction)
	_, err = h.handler.ParseNotifyRequest(ctx, httpReq, transaction)
	if err != nil {
		logger.Error("Webhook signature verification or decryption failed: %v", err)
		return &channel.WebhookResponse{
			Success:      false,
			ResponseBody: []byte(`{"code":"FAIL","message":"签名验证失败"}`),
		}, nil
	}

	logger.Info("Webhook signature verified successfully: outTradeNo=%s, tradeState=%s",
		*transaction.OutTradeNo, *transaction.TradeState)

	// 构建响应
	resp := &channel.WebhookResponse{
		Success:         true,
		PlatformTradeNo: *transaction.OutTradeNo,
		Status:          mapTradeState(*transaction.TradeState),
		PaidAmount:      int64(*transaction.Amount.Total),
		ResponseBody:    []byte(`{"code":"SUCCESS","message":"成功"}`),
	}

	if transaction.SuccessTime != nil {
		paidAt, _ := time.Parse(time.RFC3339, *transaction.SuccessTime)
		resp.PaidAt = paidAt
	}

	logger.Info("Webhook processed: outTradeNo=%s, tradeState=%s", *transaction.OutTradeNo, *transaction.TradeState)

	return resp, nil
}

// mapTradeState 映射交易状态
func mapTradeState(tradeState string) channel.OrderStatus {
	switch tradeState {
	case "SUCCESS":
		return channel.OrderStatusPaid
	case "REFUND":
		return channel.OrderStatusRefund
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
	default:
		return channel.OrderStatusPending
	}
}

func valueOrEmpty(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
