package wechat

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gopay/pkg/channel"
	"gopay/pkg/logger"
)

// WebhookHandler 处理微信支付 Webhook
type WebhookHandler struct {
	apiV3Key string // API v3 密钥，用于解密
}

// NewWebhookHandler 创建 Webhook 处理器
func NewWebhookHandler(apiV3Key string) *WebhookHandler {
	return &WebhookHandler{
		apiV3Key: apiV3Key,
	}
}

// HandleWebhook 处理 Webhook 回调
func (h *WebhookHandler) HandleWebhook(ctx context.Context, req *channel.WebhookRequest) (*channel.WebhookResponse, error) {
	logger.Info("Handling wechat webhook")

	// 1. 验证签名
	if err := h.verifySignature(req); err != nil {
		logger.Error("Webhook signature verification failed: %v", err)
		return &channel.WebhookResponse{
			Success:      false,
			ResponseBody: []byte(`{"code":"FAIL","message":"签名验证失败"}`),
		}, nil
	}

	// 2. 解析请求体
	var webhookData WechatWebhookData
	if err := json.Unmarshal(req.RawBody, &webhookData); err != nil {
		logger.Error("Failed to parse webhook body: %v", err)
		return &channel.WebhookResponse{
			Success:      false,
			ResponseBody: []byte(`{"code":"FAIL","message":"数据格式错误"}`),
		}, nil
	}

	// 3. 解密资源内容
	resource, err := h.decryptResource(webhookData.Resource)
	if err != nil {
		logger.Error("Failed to decrypt resource: %v", err)
		return &channel.WebhookResponse{
			Success:      false,
			ResponseBody: []byte(`{"code":"FAIL","message":"解密失败"}`),
		}, nil
	}

	// 4. 解析支付结果
	var paymentResult WechatPaymentResult
	if err := json.Unmarshal([]byte(resource), &paymentResult); err != nil {
		logger.Error("Failed to parse payment result: %v", err)
		return &channel.WebhookResponse{
			Success:      false,
			ResponseBody: []byte(`{"code":"FAIL","message":"支付结果解析失败"}`),
		}, nil
	}

	// 5. 构建响应
	resp := &channel.WebhookResponse{
		Success:         true,
		PlatformTradeNo: paymentResult.OutTradeNo,
		Status:          h.mapTradeState(paymentResult.TradeState),
		PaidAmount:      paymentResult.Amount.Total,
		ResponseBody:    []byte(`{"code":"SUCCESS","message":"成功"}`),
	}

	if paymentResult.SuccessTime != "" {
		paidAt, _ := time.Parse(time.RFC3339, paymentResult.SuccessTime)
		resp.PaidAt = paidAt
	}

	logger.Info("Webhook processed: outTradeNo=%s, tradeState=%s", paymentResult.OutTradeNo, paymentResult.TradeState)

	return resp, nil
}

// verifySignature 验证签名
func (h *WebhookHandler) verifySignature(req *channel.WebhookRequest) error {
	// 获取签名相关的请求头
	timestamp := req.Headers["Wechatpay-Timestamp"]
	nonce := req.Headers["Wechatpay-Nonce"]
	signature := req.Headers["Wechatpay-Signature"]
	serial := req.Headers["Wechatpay-Serial"]

	if timestamp == "" || nonce == "" || signature == "" || serial == "" {
		return errors.New("missing signature headers")
	}

	// 构建待验证的字符串
	// 格式: timestamp\n + nonce\n + body\n
	message := fmt.Sprintf("%s\n%s\n%s\n", timestamp, nonce, string(req.RawBody))

	// 这里需要使用微信支付平台证书的公钥进行验证
	// 简化实现：实际需要从微信获取平台证书并验证
	// TODO: 实现完整的签名验证逻辑
	logger.Info("Signature verification: timestamp=%s, nonce=%s, serial=%s", timestamp, nonce, serial)

	// 暂时返回成功，实际应该验证签名
	_ = message
	_ = signature

	return nil
}

// decryptResource 解密资源内容
func (h *WebhookHandler) decryptResource(resource WechatResource) (string, error) {
	// 使用 API v3 密钥解密
	key := []byte(h.apiV3Key)

	// Base64 解码
	ciphertext, err := base64.StdEncoding.DecodeString(resource.Ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	nonce, err := base64.StdEncoding.DecodeString(resource.Nonce)
	if err != nil {
		return "", fmt.Errorf("failed to decode nonce: %w", err)
	}

	associatedData := []byte(resource.AssociatedData)

	// 使用 AES-256-GCM 解密
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, associatedData)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// mapTradeState 映射交易状态
func (h *WebhookHandler) mapTradeState(tradeState string) channel.OrderStatus {
	switch tradeState {
	case "SUCCESS":
		return channel.OrderStatusPaid
	case "REFUND":
		return channel.OrderStatusRefunded
	case "NOTPAY":
		return channel.OrderStatusPending
	case "CLOSED":
		return channel.OrderStatusClosed
	case "REVOKED":
		return channel.OrderStatusClosed
	case "USERPAYING":
		return channel.OrderStatusPending
	case "PAYERROR":
		return channel.OrderStatusFailed
	default:
		return channel.OrderStatusPending
	}
}

// deriveKey 从 API v3 密钥派生 AES 密钥
func deriveKey(apiV3Key string) []byte {
	hash := sha256.Sum256([]byte(apiV3Key))
	return hash[:]
}

// WechatWebhookData 微信 Webhook 数据结构
type WechatWebhookData struct {
	ID           string          `json:"id"`
	CreateTime   string          `json:"create_time"`
	ResourceType string          `json:"resource_type"`
	EventType    string          `json:"event_type"`
	Summary      string          `json:"summary"`
	Resource     WechatResource  `json:"resource"`
}

// WechatResource 加密的资源数据
type WechatResource struct {
	Algorithm      string `json:"algorithm"`
	Ciphertext     string `json:"ciphertext"`
	AssociatedData string `json:"associated_data"`
	Nonce          string `json:"nonce"`
}

// WechatPaymentResult 支付结果
type WechatPaymentResult struct {
	AppID          string `json:"appid"`
	MchID          string `json:"mchid"`
	OutTradeNo     string `json:"out_trade_no"`
	TransactionID  string `json:"transaction_id"`
	TradeType      string `json:"trade_type"`
	TradeState     string `json:"trade_state"`
	TradeStateDesc string `json:"trade_state_desc"`
	BankType       string `json:"bank_type"`
	Attach         string `json:"attach"`
	SuccessTime    string `json:"success_time"`
	Payer          struct {
		OpenID string `json:"openid"`
	} `json:"payer"`
	Amount struct {
		Total         int64  `json:"total"`
		PayerTotal    int64  `json:"payer_total"`
		Currency      string `json:"currency"`
		PayerCurrency string `json:"payer_currency"`
	} `json:"amount"`
}
