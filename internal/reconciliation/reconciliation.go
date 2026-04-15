package reconciliation

import (
	"context"
	"time"
)

// ReconciliationService 对账服务
type ReconciliationService struct {
	wechatReconciler *WechatReconciler
	alipayReconciler *AlipayReconciler
	reportGenerator  *ReportGenerator
}

// NewReconciliationService 创建对账服务
func NewReconciliationService() *ReconciliationService {
	return &ReconciliationService{
		wechatReconciler: NewWechatReconciler(),
		alipayReconciler: NewAlipayReconciler(),
		reportGenerator:  NewReportGenerator(),
	}
}

// ReconcileResult 对账结果
type ReconcileResult struct {
	Date           time.Time         // 对账日期
	Channel        string            // 支付渠道
	TotalOrders    int               // 总订单数
	MatchedOrders  int               // 匹配订单数
	MissingOrders  []string          // 长款（外部有但内部无）
	ExtraOrders    []string          // 短款（内部有但外部无）
	AmountMismatch []AmountMismatch  // 金额不匹配
	Status         string            // 对账状态
	CreatedAt      time.Time         // 创建时间
}

// AmountMismatch 金额不匹配记录
type AmountMismatch struct {
	OrderNo        string  // 订单号
	InternalAmount int64   // 内部金额
	ExternalAmount int64   // 外部金额
	Difference     int64   // 差额
}

// BillRecord 账单记录
type BillRecord struct {
	TransactionID string    // 交易ID
	OrderNo       string    // 订单号
	Amount        int64     // 金额（分）
	Status        string    // 状态
	PaidAt        time.Time // 支付时间
	Channel       string    // 渠道
}

// Reconcile 执行对账
func (s *ReconciliationService) Reconcile(ctx context.Context, date time.Time, channel string) (*ReconcileResult, error) {
	switch channel {
	case "wechat":
		return s.wechatReconciler.Reconcile(ctx, date)
	case "alipay":
		return s.alipayReconciler.Reconcile(ctx, date)
	default:
		return nil, ErrUnsupportedChannel
	}
}

// ReconcileAll 对所有渠道执行对账
func (s *ReconciliationService) ReconcileAll(ctx context.Context, date time.Time) ([]*ReconcileResult, error) {
	var results []*ReconcileResult

	// 微信对账
	wechatResult, err := s.wechatReconciler.Reconcile(ctx, date)
	if err != nil {
		return nil, err
	}
	results = append(results, wechatResult)

	// 支付宝对账
	alipayResult, err := s.alipayReconciler.Reconcile(ctx, date)
	if err != nil {
		return nil, err
	}
	results = append(results, alipayResult)

	return results, nil
}

// GenerateReport 生成对账报告
func (s *ReconciliationService) GenerateReport(ctx context.Context, result *ReconcileResult) (string, error) {
	return s.reportGenerator.Generate(ctx, result)
}

// ScheduleReconciliation 定时对账任务
func (s *ReconciliationService) ScheduleReconciliation(ctx context.Context) error {
	// 每天凌晨 2 点执行前一天的对账
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			yesterday := time.Now().AddDate(0, 0, -1)
			results, err := s.ReconcileAll(ctx, yesterday)
			if err != nil {
				// 记录错误日志
				continue
			}

			// 生成报告
			for _, result := range results {
				_, err := s.GenerateReport(ctx, result)
				if err != nil {
					// 记录错误日志
					continue
				}

				// 如果有差异，发送告警
				if len(result.MissingOrders) > 0 || len(result.ExtraOrders) > 0 || len(result.AmountMismatch) > 0 {
					s.sendAlert(result)
				}
			}
		}
	}
}

// sendAlert 发送告警
func (s *ReconciliationService) sendAlert(result *ReconcileResult) {
	// TODO: 实现告警逻辑（邮件、钉钉、飞书等）
}

var (
	ErrUnsupportedChannel = NewError("unsupported channel")
	ErrBillDownloadFailed = NewError("bill download failed")
	ErrBillParseFailed    = NewError("bill parse failed")
)

func NewError(msg string) error {
	return &ReconciliationError{Message: msg}
}

type ReconciliationError struct {
	Message string
}

func (e *ReconciliationError) Error() string {
	return e.Message
}
