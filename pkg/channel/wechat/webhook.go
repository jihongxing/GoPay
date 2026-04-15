package wechat

import (
	"encoding/json"
	"fmt"
	"time"

	"gopay/pkg/channel"
	"gopay/pkg/logger"
)

// NotificationRequest 微信支付通知请求
type NotificationRequest struct {
	ID           string                 `json:"id"`
	CreateTime   string                 `json:"create_time"`
	EventType    string                 `json:"event_type"`
	ResourceType string                 `json:"resource_type"`
	Summary      string                 `json:"summary"`
	Resource     NotificationResource   `json:"resource"`
}

// NotificationResource 通知资源
type NotificationResource struct {
	Algorithm      string               `json:"algorithm"`
	Ciphertext     string               `json:"ciphertext"`
	AssociatedData string               `json:"associated_data"`
	Nonce          string               `json:"nonce"`
	OriginalType   string               `json:"original_type"`
	PlainText      *TransactionDetail   `json:"-"` // 解密后的明文
}

// TransactionDetail 交易详情（简化版）
type TransactionDetail struct {
	Appid          string `json:"appid"`
	Mchid          string `json:"mchid"`
	OutTradeNo     string `json:"out_trade_no"`
	TransactionId  string `json:"transaction_id"`
	TradeType      string `json:"trade_type"`
	TradeState     string `json:"trade_state"`
	TradeStateDesc string `json:"trade_state_desc"`
	BankType       string `json:"bank_type"`
	SuccessTime    string `json:"success_time"`
	Amount         struct {
		Total         int64  `json:"total"`
		PayerTotal    int64  `json:"payer_total"`
		Currency      string `json:"currency"`
		PayerCurrency string `json:"payer_currency"`
	} `json:"amount"`
}

// parseNotification 解析并验证微信支付通知
// TODO: 完整实现需要验证签名，这里先简化处理
func (p *Provider) parseNotification(req *channel.WebhookRequest) (*NotificationRequest, error) {
	logger.Info("Parsing wechat notification, body length=%d", len(req.RawBody))

	// 解析通知请求
	notifyReq := new(NotificationRequest)
	if err := json.Unmarshal(req.RawBody, notifyReq); err != nil {
		return nil, fmt.Errorf("failed to unmarshal notification: %w", err)
	}

	// TODO: 验证签名
	// 1. 从请求头获取签名信息
	// 2. 使用平台公钥验证签名
	// 3. 验证时间戳防止重放攻击

	// TODO: 解密资源内容
	// 使用 AEAD_AES_256_GCM 算法解密
	// plaintext := decryptAES256GCM(notifyReq.Resource.Ciphertext, p.apiV3Key, notifyReq.Resource.Nonce, notifyReq.Resource.AssociatedData)

	// 临时：直接解析（生产环境必须解密）
	transaction := new(TransactionDetail)
	// if err := json.Unmarshal([]byte(plaintext), transaction); err != nil {
	// 	return nil, fmt.Errorf("failed to unmarshal transaction: %w", err)
	// }
	notifyReq.Resource.PlainText = transaction

	logger.Info("Wechat notification parsed: id=%s, eventType=%s", notifyReq.ID, notifyReq.EventType)

	return notifyReq, nil
}

// parseWechatTime 解析微信时间格式
// 微信时间格式：2018-06-08T10:34:56+08:00
func parseWechatTime(timeStr string) (time.Time, error) {
	return time.Parse(time.RFC3339, timeStr)
}

// formatWechatTime 格式化为微信时间格式
func formatWechatTime(t time.Time) string {
	return t.Format(time.RFC3339)
}

