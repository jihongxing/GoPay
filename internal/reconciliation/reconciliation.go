package reconciliation

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// ReconciliationService 对账服务
type ReconciliationService struct {
	db               *sql.DB
	wechatReconciler *WechatReconciler
	alipayReconciler *AlipayReconciler
	reportGenerator  *ReportGenerator
	alertNotifier    AlertNotifier
}

// BillDownloader 账单下载器接口
type BillDownloader interface {
	Download(ctx context.Context, date time.Time) ([]byte, error)
}

// NewReconciliationService 创建对账服务
func NewReconciliationService(db ...*sql.DB) *ReconciliationService {
	var conn *sql.DB
	if len(db) > 0 {
		conn = db[0]
	}

	return &ReconciliationService{
		db:               conn,
		wechatReconciler: NewWechatReconciler(db...),
		alipayReconciler: NewAlipayReconciler(db...),
		reportGenerator:  NewReportGenerator(),
	}
}

// ReconcileResult 对账结果
type ReconcileResult struct {
	Date           time.Time        // 对账日期
	Channel        string           // 支付渠道
	AppID          string           // 应用ID（可选，用于按应用维度对账）
	TotalOrders    int              // 总订单数
	MatchedOrders  int              // 匹配订单数
	MissingOrders  []string         // 长款（外部有但内部无）
	ExtraOrders    []string         // 短款（内部有但外部无）
	AmountMismatch []AmountMismatch // 金额不匹配
	Status         string           // 对账状态
	CreatedAt      time.Time        // 创建时间
}

// AmountMismatch 金额不匹配记录
type AmountMismatch struct {
	OrderNo        string // 订单号
	InternalAmount int64  // 内部金额
	ExternalAmount int64  // 外部金额
	Difference     int64  // 差额
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
	if result == nil {
		return "", fmt.Errorf("result is required")
	}

	if result.CreatedAt.IsZero() {
		result.CreatedAt = time.Now()
	}

	path, err := s.reportGenerator.Generate(ctx, result)
	if err != nil {
		return "", err
	}

	if s.db == nil {
		return path, nil
	}

	reportID, err := s.saveReport(ctx, result, path)
	if err != nil {
		return "", err
	}

	if err := s.saveReportDetails(ctx, reportID, result); err != nil {
		return "", err
	}

	return path, nil
}

func (s *ReconciliationService) saveReport(ctx context.Context, result *ReconcileResult, path string) (int64, error) {
	var id int64

	// 根据是否有 app_id 使用不同的 SQL
	var err error
	if result.AppID != "" {
		err = s.db.QueryRowContext(ctx, `
			INSERT INTO reconciliation_reports (
				date, channel, app_id, total_orders, matched_orders,
				long_orders, short_orders, amount_mismatch, status, file_path, created_at, updated_at
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NOW(),NOW())
			ON CONFLICT (date, channel, COALESCE(app_id, '')) DO UPDATE SET
				total_orders = EXCLUDED.total_orders,
				matched_orders = EXCLUDED.matched_orders,
				long_orders = EXCLUDED.long_orders,
				short_orders = EXCLUDED.short_orders,
				amount_mismatch = EXCLUDED.amount_mismatch,
				status = EXCLUDED.status,
				file_path = EXCLUDED.file_path,
				updated_at = NOW()
			RETURNING id
		`,
			result.Date.Format("2006-01-02"),
			result.Channel,
			result.AppID,
			result.TotalOrders,
			result.MatchedOrders,
			len(result.MissingOrders),
			len(result.ExtraOrders),
			len(result.AmountMismatch),
			result.Status,
			path,
		).Scan(&id)
	} else {
		err = s.db.QueryRowContext(ctx, `
			INSERT INTO reconciliation_reports (
				date, channel, total_orders, matched_orders,
				long_orders, short_orders, amount_mismatch, status, file_path, created_at, updated_at
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,NOW(),NOW())
			ON CONFLICT (date, channel, COALESCE(app_id, '')) DO UPDATE SET
				total_orders = EXCLUDED.total_orders,
				matched_orders = EXCLUDED.matched_orders,
				long_orders = EXCLUDED.long_orders,
				short_orders = EXCLUDED.short_orders,
				amount_mismatch = EXCLUDED.amount_mismatch,
				status = EXCLUDED.status,
				file_path = EXCLUDED.file_path,
				updated_at = NOW()
			RETURNING id
		`,
			result.Date.Format("2006-01-02"),
			result.Channel,
			result.TotalOrders,
			result.MatchedOrders,
			len(result.MissingOrders),
			len(result.ExtraOrders),
			len(result.AmountMismatch),
			result.Status,
			path,
		).Scan(&id)
	}

	if err != nil {
		return 0, fmt.Errorf("save reconciliation report failed: %w", err)
	}
	return id, nil
}

func (s *ReconciliationService) saveReportDetails(ctx context.Context, reportID int64, result *ReconcileResult) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin report detail tx failed: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM reconciliation_details WHERE report_id = $1`, reportID); err != nil {
		return fmt.Errorf("clear reconciliation details failed: %w", err)
	}

	for _, orderNo := range result.MissingOrders {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO reconciliation_details (report_id, order_no, type)
			VALUES ($1, $2, 'long')
		`, reportID, orderNo); err != nil {
			return fmt.Errorf("save long detail failed: %w", err)
		}
	}

	for _, orderNo := range result.ExtraOrders {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO reconciliation_details (report_id, order_no, type)
			VALUES ($1, $2, 'short')
		`, reportID, orderNo); err != nil {
			return fmt.Errorf("save short detail failed: %w", err)
		}
	}

	for _, mismatch := range result.AmountMismatch {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO reconciliation_details (
				report_id, order_no, type, internal_amount, external_amount, diff
			) VALUES ($1, $2, 'amount_mismatch', $3, $4, $5)
		`, reportID, mismatch.OrderNo, mismatch.InternalAmount, mismatch.ExternalAmount, mismatch.Difference); err != nil {
			return fmt.Errorf("save amount mismatch detail failed: %w", err)
		}
	}

	return tx.Commit()
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

// SetAlertNotifier 设置告警通知器
func (s *ReconciliationService) SetAlertNotifier(notifier AlertNotifier) {
	s.alertNotifier = notifier
}

// sendAlert 发送告警
func (s *ReconciliationService) sendAlert(result *ReconcileResult) {
	if s.alertNotifier == nil {
		return
	}

	msg := fmt.Sprintf(
		"⚠️ 对账异常告警\n渠道: %s\n日期: %s\n长款: %d笔\n短款: %d笔\n金额不匹配: %d笔",
		result.Channel,
		result.Date.Format("2006-01-02"),
		len(result.MissingOrders),
		len(result.ExtraOrders),
		len(result.AmountMismatch),
	)

	if err := s.alertNotifier.SendAlert(context.Background(), msg); err != nil {
		fmt.Printf("Failed to send reconciliation alert: %v\n", err)
	}
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
