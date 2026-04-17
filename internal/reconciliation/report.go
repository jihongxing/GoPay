package reconciliation

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
)

// ReportGenerator 报告生成器
type ReportGenerator struct {
	reportDir string
}

// NewReportGenerator 创建报告生成器
func NewReportGenerator() *ReportGenerator {
	return &ReportGenerator{
		reportDir: "./reports",
	}
}

// Generate 生成对账报告
func (g *ReportGenerator) Generate(ctx context.Context, result *ReconcileResult) (string, error) {
	// 确保报告目录存在
	if err := os.MkdirAll(g.reportDir, 0755); err != nil {
		return "", fmt.Errorf("create report dir failed: %w", err)
	}

	// 生成报告文件名
	filename := fmt.Sprintf("reconciliation_%s_%s.csv",
		result.Channel,
		result.Date.Format("20060102"))
	filepath := filepath.Join(g.reportDir, filename)

	// 创建 CSV 文件
	file, err := os.Create(filepath)
	if err != nil {
		return "", fmt.Errorf("create report file failed: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 写入报告头
	g.writeHeader(writer, result)

	// 写入汇总信息
	g.writeSummary(writer, result)

	// 写入长款明细
	if len(result.MissingOrders) > 0 {
		g.writeMissingOrders(writer, result)
	}

	// 写入短款明细
	if len(result.ExtraOrders) > 0 {
		g.writeExtraOrders(writer, result)
	}

	// 写入金额不匹配明细
	if len(result.AmountMismatch) > 0 {
		g.writeAmountMismatch(writer, result)
	}

	return filepath, nil
}

// writeHeader 写入报告头
func (g *ReportGenerator) writeHeader(writer *csv.Writer, result *ReconcileResult) {
	writer.Write([]string{"GoPay 对账报告"})
	writer.Write([]string{"对账日期", result.Date.Format("2006-01-02")})
	writer.Write([]string{"支付渠道", result.Channel})
	writer.Write([]string{"生成时间", result.CreatedAt.Format("2006-01-02 15:04:05")})
	writer.Write([]string{"对账状态", result.Status})
	writer.Write([]string{""}) // 空行
}

// writeSummary 写入汇总信息
func (g *ReportGenerator) writeSummary(writer *csv.Writer, result *ReconcileResult) {
	writer.Write([]string{"汇总信息"})
	writer.Write([]string{"总订单数", fmt.Sprintf("%d", result.TotalOrders)})
	writer.Write([]string{"匹配订单数", fmt.Sprintf("%d", result.MatchedOrders)})
	writer.Write([]string{"长款订单数", fmt.Sprintf("%d", len(result.MissingOrders))})
	writer.Write([]string{"短款订单数", fmt.Sprintf("%d", len(result.ExtraOrders))})
	writer.Write([]string{"金额不匹配数", fmt.Sprintf("%d", len(result.AmountMismatch))})
	writer.Write([]string{""}) // 空行
}

// writeMissingOrders 写入长款明细
func (g *ReportGenerator) writeMissingOrders(writer *csv.Writer, result *ReconcileResult) {
	writer.Write([]string{"长款明细（外部有但内部无）"})
	writer.Write([]string{"订单号"})

	for _, orderNo := range result.MissingOrders {
		writer.Write([]string{orderNo})
	}

	writer.Write([]string{""}) // 空行
}

// writeExtraOrders 写入短款明细
func (g *ReportGenerator) writeExtraOrders(writer *csv.Writer, result *ReconcileResult) {
	writer.Write([]string{"短款明细（内部有但外部无）- 严重问题！"})
	writer.Write([]string{"订单号"})

	for _, orderNo := range result.ExtraOrders {
		writer.Write([]string{orderNo})
	}

	writer.Write([]string{""}) // 空行
}

// writeAmountMismatch 写入金额不匹配明细
func (g *ReportGenerator) writeAmountMismatch(writer *csv.Writer, result *ReconcileResult) {
	writer.Write([]string{"金额不匹配明细"})
	writer.Write([]string{"订单号", "内部金额（分）", "外部金额（分）", "差额（分）"})

	for _, mismatch := range result.AmountMismatch {
		writer.Write([]string{
			mismatch.OrderNo,
			fmt.Sprintf("%d", mismatch.InternalAmount),
			fmt.Sprintf("%d", mismatch.ExternalAmount),
			fmt.Sprintf("%d", mismatch.Difference),
		})
	}

	writer.Write([]string{""}) // 空行
}

// GenerateExcel 生成 Excel 报告
// 需要安装 excelize 库: go get github.com/xuri/excelize/v2
func (g *ReportGenerator) GenerateExcel(ctx context.Context, result *ReconcileResult) (string, error) {
	return "", fmt.Errorf("excel report generation not implemented: install github.com/xuri/excelize/v2")
}
