package alert

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"gopay/internal/models"
	"gopay/pkg/logger"
)

// AlertLevel 告警级别
type AlertLevel string

const (
	AlertLevelInfo     AlertLevel = "info"
	AlertLevelWarning  AlertLevel = "warning"
	AlertLevelError    AlertLevel = "error"
	AlertLevelCritical AlertLevel = "critical"
)

// AlertMessage 告警消息
type AlertMessage struct {
	Level     AlertLevel        `json:"level"`
	Title     string            `json:"title"`
	Content   string            `json:"content"`
	Details   map[string]string `json:"details,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// AlertManager 告警管理器
type AlertManager struct {
	webhookURL string
	httpClient *http.Client
}

// NewAlertManager 创建告警管理器
func NewAlertManager(webhookURL string) *AlertManager {
	return &AlertManager{
		webhookURL: webhookURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// SendAlert 发送告警
func (am *AlertManager) SendAlert(msg *AlertMessage) error {
	if am.webhookURL == "" {
		logger.Error("Alert webhook URL not configured")
		return fmt.Errorf("alert webhook URL not configured")
	}

	msg.Timestamp = time.Now()

	// 序列化消息
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal alert message: %w", err)
	}

	// 发送 HTTP 请求
	req, err := http.NewRequest("POST", am.webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create alert request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := am.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send alert: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("alert webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// AlertNotifyFailed 通知失败告警
func (am *AlertManager) AlertNotifyFailed(order *models.Order) {
	msg := &AlertMessage{
		Level:   AlertLevelError,
		Title:   "订单通知失败",
		Content: fmt.Sprintf("订单 %s 通知失败，已重试 %d 次", order.OrderNo, order.RetryCount),
		Details: map[string]string{
			"order_no":     order.OrderNo,
			"out_trade_no": order.OutTradeNo,
			"app_id":       order.AppID,
			"amount":       fmt.Sprintf("%d", order.Amount),
			"retry_count":  fmt.Sprintf("%d", order.RetryCount),
		},
	}

	if err := am.SendAlert(msg); err != nil {
		logger.Error("Failed to send notify failed alert: %v", err)
	} else {
		logger.Info("Notify failed alert sent: orderNo=%s", order.OrderNo)
	}
}

// AlertPaymentAbnormal 支付异常告警
func (am *AlertManager) AlertPaymentAbnormal(orderNo string, reason string, details map[string]string) {
	msg := &AlertMessage{
		Level:   AlertLevelWarning,
		Title:   "支付异常",
		Content: fmt.Sprintf("订单 %s 支付异常: %s", orderNo, reason),
		Details: details,
	}

	if err := am.SendAlert(msg); err != nil {
		logger.Error("Failed to send payment abnormal alert: %v", err)
	}
}

// AlertSystemError 系统错误告警
func (am *AlertManager) AlertSystemError(component string, err error, details map[string]string) {
	msg := &AlertMessage{
		Level:   AlertLevelCritical,
		Title:   "系统错误",
		Content: fmt.Sprintf("组件 %s 发生错误: %v", component, err),
		Details: details,
	}

	if sendErr := am.SendAlert(msg); sendErr != nil {
		logger.Error("Failed to send system error alert: %v", sendErr)
	}
}

// AlertHighRetryRate 高重试率告警
func (am *AlertManager) AlertHighRetryRate(rate float64, count int) {
	msg := &AlertMessage{
		Level:   AlertLevelWarning,
		Title:   "通知重试率过高",
		Content: fmt.Sprintf("当前通知重试率: %.2f%%, 重试订单数: %d", rate*100, count),
		Details: map[string]string{
			"retry_rate":  fmt.Sprintf("%.2f%%", rate*100),
			"retry_count": fmt.Sprintf("%d", count),
		},
	}

	if err := am.SendAlert(msg); err != nil {
		logger.Error("Failed to send high retry rate alert: %v", err)
	}
}

// DingTalkAlertManager 钉钉告警管理器
type DingTalkAlertManager struct {
	webhookURL string
	httpClient *http.Client
}

// NewDingTalkAlertManager 创建钉钉告警管理器
func NewDingTalkAlertManager(webhookURL string) *DingTalkAlertManager {
	return &DingTalkAlertManager{
		webhookURL: webhookURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// DingTalkMessage 钉钉消息格式
type DingTalkMessage struct {
	MsgType string `json:"msgtype"`
	Text    struct {
		Content string `json:"content"`
	} `json:"text"`
}

// SendAlert 发送钉钉告警
func (dm *DingTalkAlertManager) SendAlert(msg *AlertMessage) error {
	if dm.webhookURL == "" {
		return fmt.Errorf("dingtalk webhook URL not configured")
	}

	// 构建钉钉消息
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("[%s] %s\n%s\n时间: %s",
		msg.Level, msg.Title, msg.Content, msg.Timestamp.Format("2006-01-02 15:04:05")))

	if len(msg.Details) > 0 {
		builder.WriteString("\n详情:")
		for k, v := range msg.Details {
			builder.WriteString(fmt.Sprintf("\n  %s: %s", k, v))
		}
	}

	dingMsg := DingTalkMessage{
		MsgType: "text",
	}
	dingMsg.Text.Content = builder.String()

	body, err := json.Marshal(dingMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal dingtalk message: %w", err)
	}

	req, err := http.NewRequest("POST", dm.webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create dingtalk request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := dm.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send dingtalk alert: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("dingtalk webhook returned status %d", resp.StatusCode)
	}

	return nil
}
