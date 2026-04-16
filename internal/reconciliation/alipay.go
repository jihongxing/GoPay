package reconciliation

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"time"
)

// AlipayReconciler 支付宝对账器
type AlipayReconciler struct {
	billDownloader *AlipayBillDownloader
	orderRepo      OrderRepository
}

// NewAlipayReconciler 创建支付宝对账器
func NewAlipayReconciler() *AlipayReconciler {
	return &AlipayReconciler{
		billDownloader: NewAlipayBillDownloader(),
		orderRepo:      NewOrderRepository(),
	}
}

// Reconcile 执行支付宝对账
func (r *AlipayReconciler) Reconcile(ctx context.Context, date time.Time) (*ReconcileResult, error) {
	// 1. 下载支付宝账单
	billData, err := r.billDownloader.Download(ctx, date)
	if err != nil {
		return nil, fmt.Errorf("download alipay bill failed: %w", err)
	}

	// 2. 解析账单
	externalRecords, err := r.parseBill(billData)
	if err != nil {
		return nil, fmt.Errorf("parse alipay bill failed: %w", err)
	}

	// 3. 获取内部订单
	internalOrders, err := r.orderRepo.GetOrdersByDate(ctx, date, "alipay")
	if err != nil {
		return nil, fmt.Errorf("get internal orders failed: %w", err)
	}

	// 4. 双向比对
	result := r.compare(externalRecords, internalOrders)
	result.Date = date
	result.Channel = "alipay"
	result.CreatedAt = time.Now()

	return result, nil
}

// parseBill 解析支付宝账单
func (r *AlipayReconciler) parseBill(data []byte) ([]BillRecord, error) {
	var records []BillRecord

	if len(data) == 0 {
		return records, nil
	}

	// 创建 CSV reader
	reader := csv.NewReader(strings.NewReader(string(data)))
	reader.Comma = ','
	reader.LazyQuotes = true // 允许不规范的引号
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1 // 允许可变字段数

	// 支付宝账单格式：
	// 交易号,商户订单号,交易创建时间,付款时间,最近修改时间,交易来源地,类型,交易对方,商品名称,金额（元）,收支,交易状态,服务费（元）,成功退款（元）,备注,资金状态

	// 跳过表头（支付宝账单前5行是说明，第6行是表头）
	for i := 0; i < 6; i++ {
		_, err := reader.Read()
		if err == io.EOF {
			// 如果文件行数不足6行，返回空结果
			return records, nil
		}
		if err != nil {
			return nil, err
		}
	}

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		// 解析每一行
		if len(row) < 12 {
			continue
		}

		record := BillRecord{
			TransactionID: row[0],  // 交易号
			OrderNo:       row[1],  // 商户订单号
			Amount:        parseAlipayAmount(row[9]), // 金额（元）
			Status:        row[11], // 交易状态
			Channel:       "alipay",
		}

		records = append(records, record)
	}

	return records, nil
}

// parseAlipayAmount 解析支付宝金额（元转分）
func parseAlipayAmount(amountStr string) int64 {
	var amount float64
	fmt.Sscanf(amountStr, "%f", &amount)
	return int64(amount * 100)
}

// compare 比对内外部数据
func (r *AlipayReconciler) compare(external []BillRecord, internal []Order) *ReconcileResult {
	result := &ReconcileResult{
		TotalOrders: len(external),
	}

	// 构建内部订单映射
	internalMap := make(map[string]Order)
	for _, order := range internal {
		internalMap[order.OrderNo] = order
	}

	// 构建外部订单映射
	externalMap := make(map[string]BillRecord)
	for _, record := range external {
		externalMap[record.OrderNo] = record
	}

	// 检查外部订单
	for _, record := range external {
		if order, exists := internalMap[record.OrderNo]; exists {
			// 订单存在，检查金额
			if order.Amount != record.Amount {
				result.AmountMismatch = append(result.AmountMismatch, AmountMismatch{
					OrderNo:        record.OrderNo,
					InternalAmount: order.Amount,
					ExternalAmount: record.Amount,
					Difference:     order.Amount - record.Amount,
				})
			} else {
				result.MatchedOrders++
			}
		} else {
			// 长款：外部有但内部无
			result.MissingOrders = append(result.MissingOrders, record.OrderNo)
		}
	}

	// 检查内部订单
	for _, order := range internal {
		if _, exists := externalMap[order.OrderNo]; !exists {
			// 短款：内部有但外部无（严重问题）
			result.ExtraOrders = append(result.ExtraOrders, order.OrderNo)
		}
	}

	// 设置状态
	if len(result.MissingOrders) == 0 && len(result.ExtraOrders) == 0 && len(result.AmountMismatch) == 0 {
		result.Status = "success"
	} else {
		result.Status = "failed"
	}

	return result
}

// AlipayBillDownloader 支付宝账单下载器
type AlipayBillDownloader struct{}

func NewAlipayBillDownloader() *AlipayBillDownloader {
	return &AlipayBillDownloader{}
}

func (d *AlipayBillDownloader) Download(ctx context.Context, date time.Time) ([]byte, error) {
	// 调用支付宝 API 下载账单
	// https://opendocs.alipay.com/open/028wob
	//
	// 实现步骤：
	// 1. 构建请求参数（账单日期、账单类型等）
	// 2. 使用应用私钥签名请求
	// 3. 调用支付宝 API
	// 4. 下载并解析账单文件
	//
	// 示例代码：
	// billDate := date.Format("2006-01-02")
	// req := alipay.NewAlipayDataDataserviceBillDownloadurlQueryRequest()
	// req.BizContent = fmt.Sprintf(`{
	//     "bill_type": "trade",
	//     "bill_date": "%s"
	// }`, billDate)
	//
	// resp, err := client.Execute(req)
	// if err != nil {
	//     return nil, fmt.Errorf("download bill failed: %w", err)
	// }
	//
	// // 下载账单文件
	// billData, err := downloadBillFile(resp.BillDownloadUrl)
	// return billData, err

	return nil, fmt.Errorf("alipay bill download not implemented: please integrate alipay SDK")
}
