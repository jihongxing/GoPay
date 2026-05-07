package reconciliation

import (
	"context"
	"fmt"
	"time"
)

// ReconcileByApp 按应用维度执行对账
func (s *ReconciliationService) ReconcileByApp(ctx context.Context, date time.Time, channel, appID string) (*ReconcileResult, error) {
	if appID == "" {
		return nil, fmt.Errorf("app_id is required")
	}

	var result *ReconcileResult
	var err error

	switch channel {
	case "wechat":
		result, err = s.wechatReconciler.ReconcileByApp(ctx, date, appID)
	case "alipay":
		result, err = s.alipayReconciler.ReconcileByApp(ctx, date, appID)
	default:
		return nil, ErrUnsupportedChannel
	}

	if err != nil {
		return nil, err
	}

	// 设置 app_id
	result.AppID = appID

	return result, nil
}

// ReconcileAllByApp 对所有渠道按应用维度执行对账
func (s *ReconciliationService) ReconcileAllByApp(ctx context.Context, date time.Time, appID string) ([]*ReconcileResult, error) {
	if appID == "" {
		return nil, fmt.Errorf("app_id is required")
	}

	var results []*ReconcileResult

	// 微信对账
	wechatResult, err := s.wechatReconciler.ReconcileByApp(ctx, date, appID)
	if err != nil {
		return nil, fmt.Errorf("wechat reconciliation failed: %w", err)
	}
	wechatResult.AppID = appID
	results = append(results, wechatResult)

	// 支付宝对账
	alipayResult, err := s.alipayReconciler.ReconcileByApp(ctx, date, appID)
	if err != nil {
		return nil, fmt.Errorf("alipay reconciliation failed: %w", err)
	}
	alipayResult.AppID = appID
	results = append(results, alipayResult)

	return results, nil
}

// GetReportsByApp 获取指定应用的对账报告列表
func (s *ReconciliationService) GetReportsByApp(ctx context.Context, appID string, startDate, endDate time.Time, page, pageSize int) ([]*ReconciliationReport, int, error) {
	if s.db == nil {
		return nil, 0, fmt.Errorf("database connection is required")
	}

	offset := (page - 1) * pageSize

	// 查询总数
	var total int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM reconciliation_reports
		WHERE app_id = $1 AND date BETWEEN $2 AND $3
	`, appID, startDate, endDate).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count reports: %w", err)
	}

	// 查询报告列表
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, date, channel, app_id, total_orders, matched_orders,
		       long_orders, short_orders, amount_mismatch, status, file_path,
		       created_at, updated_at
		FROM reconciliation_reports
		WHERE app_id = $1 AND date BETWEEN $2 AND $3
		ORDER BY date DESC, channel
		LIMIT $4 OFFSET $5
	`, appID, startDate, endDate, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query reports: %w", err)
	}
	defer rows.Close()

	var reports []*ReconciliationReport
	for rows.Next() {
		var report ReconciliationReport
		err := rows.Scan(
			&report.ID,
			&report.Date,
			&report.Channel,
			&report.AppID,
			&report.TotalOrders,
			&report.MatchedOrders,
			&report.LongOrders,
			&report.ShortOrders,
			&report.AmountMismatch,
			&report.Status,
			&report.FilePath,
			&report.CreatedAt,
			&report.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan report: %w", err)
		}
		reports = append(reports, &report)
	}

	return reports, total, nil
}

// GetAppReconciliationStats 获取应用对账统计数据
func (s *ReconciliationService) GetAppReconciliationStats(ctx context.Context, appID string, startDate, endDate time.Time) (*AppReconciliationStats, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database connection is required")
	}

	stats := &AppReconciliationStats{
		AppID:     appID,
		StartDate: startDate,
		EndDate:   endDate,
	}

	// 查询统计数据
	err := s.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) as report_count,
			COALESCE(SUM(total_orders), 0) as total_orders,
			COALESCE(SUM(matched_orders), 0) as matched_orders,
			COALESCE(SUM(long_orders), 0) as total_long_orders,
			COALESCE(SUM(short_orders), 0) as total_short_orders,
			COALESCE(SUM(amount_mismatch), 0) as total_amount_mismatch
		FROM reconciliation_reports
		WHERE app_id = $1 AND date BETWEEN $2 AND $3
	`, appID, startDate, endDate).Scan(
		&stats.ReportCount,
		&stats.TotalOrders,
		&stats.MatchedOrders,
		&stats.TotalLongOrders,
		&stats.TotalShortOrders,
		&stats.TotalAmountMismatch,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query stats: %w", err)
	}

	// 计算匹配率
	if stats.TotalOrders > 0 {
		stats.MatchRate = float64(stats.MatchedOrders) / float64(stats.TotalOrders) * 100
	}

	return stats, nil
}

// GetAllAppsReconciliationSummary 获取所有应用的对账汇总
func (s *ReconciliationService) GetAllAppsReconciliationSummary(ctx context.Context, date time.Time) ([]*AppReconciliationSummary, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database connection is required")
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT
			COALESCE(app_id, 'unknown') as app_id,
			COUNT(*) as report_count,
			SUM(total_orders) as total_orders,
			SUM(matched_orders) as matched_orders,
			SUM(long_orders) as long_orders,
			SUM(short_orders) as short_orders,
			SUM(amount_mismatch) as amount_mismatch
		FROM reconciliation_reports
		WHERE date = $1
		GROUP BY app_id
		ORDER BY total_orders DESC
	`, date)
	if err != nil {
		return nil, fmt.Errorf("failed to query summary: %w", err)
	}
	defer rows.Close()

	var summaries []*AppReconciliationSummary
	for rows.Next() {
		var summary AppReconciliationSummary
		err := rows.Scan(
			&summary.AppID,
			&summary.ReportCount,
			&summary.TotalOrders,
			&summary.MatchedOrders,
			&summary.LongOrders,
			&summary.ShortOrders,
			&summary.AmountMismatch,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan summary: %w", err)
		}

		// 计算匹配率
		if summary.TotalOrders > 0 {
			summary.MatchRate = float64(summary.MatchedOrders) / float64(summary.TotalOrders) * 100
		}

		summaries = append(summaries, &summary)
	}

	return summaries, nil
}

// ReconciliationReport 对账报告模型
type ReconciliationReport struct {
	ID             int64     `json:"id"`
	Date           time.Time `json:"date"`
	Channel        string    `json:"channel"`
	AppID          string    `json:"app_id"`
	TotalOrders    int       `json:"total_orders"`
	MatchedOrders  int       `json:"matched_orders"`
	LongOrders     int       `json:"long_orders"`
	ShortOrders    int       `json:"short_orders"`
	AmountMismatch int       `json:"amount_mismatch"`
	Status         string    `json:"status"`
	FilePath       string    `json:"file_path"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// AppReconciliationStats 应用对账统计
type AppReconciliationStats struct {
	AppID               string    `json:"app_id"`
	StartDate           time.Time `json:"start_date"`
	EndDate             time.Time `json:"end_date"`
	ReportCount         int       `json:"report_count"`
	TotalOrders         int       `json:"total_orders"`
	MatchedOrders       int       `json:"matched_orders"`
	TotalLongOrders     int       `json:"total_long_orders"`
	TotalShortOrders    int       `json:"total_short_orders"`
	TotalAmountMismatch int       `json:"total_amount_mismatch"`
	MatchRate           float64   `json:"match_rate"`
}

// AppReconciliationSummary 应用对账汇总
type AppReconciliationSummary struct {
	AppID          string  `json:"app_id"`
	ReportCount    int     `json:"report_count"`
	TotalOrders    int     `json:"total_orders"`
	MatchedOrders  int     `json:"matched_orders"`
	LongOrders     int     `json:"long_orders"`
	ShortOrders    int     `json:"short_orders"`
	AmountMismatch int     `json:"amount_mismatch"`
	MatchRate      float64 `json:"match_rate"`
}
