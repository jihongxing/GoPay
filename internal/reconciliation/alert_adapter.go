package reconciliation

import (
	"context"

	"gopay/pkg/alert"
)

// AlertManagerNotifier 将 pkg/alert 适配为对账调度器的通知器
type AlertManagerNotifier struct {
	manager *alert.AlertManager
}

// NewAlertManagerNotifier 创建适配器
func NewAlertManagerNotifier(manager *alert.AlertManager) *AlertManagerNotifier {
	return &AlertManagerNotifier{manager: manager}
}

// SendAlert 发送告警
func (n *AlertManagerNotifier) SendAlert(ctx context.Context, message string) error {
	if n == nil || n.manager == nil {
		return nil
	}

	return n.manager.SendAlert(&alert.AlertMessage{
		Level:   alert.AlertLevelError,
		Title:   "对账异常",
		Content: message,
	})
}
