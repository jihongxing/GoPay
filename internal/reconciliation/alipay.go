package reconciliation

import (
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/smartwalle/alipay/v3"
)

// AlipayReconciler 支付宝对账器
type AlipayReconciler struct {
	billDownloader BillDownloader
	orderRepo      OrderRepository
}

// NewAlipayReconciler 创建支付宝对账器
func NewAlipayReconciler(db ...*sql.DB) *AlipayReconciler {
	var repo OrderRepository = NewOrderRepository()
	if len(db) > 0 && db[0] != nil {
		repo = NewDBOrderRepository(db[0])
	}

	return &AlipayReconciler{
		billDownloader: NewAlipayBillDownloader(),
		orderRepo:      repo,
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

// ReconcileByApp 按应用维度执行支付宝对账
func (r *AlipayReconciler) ReconcileByApp(ctx context.Context, date time.Time, appID string) (*ReconcileResult, error) {
	billData, err := r.billDownloader.Download(ctx, date)
	if err != nil {
		return nil, fmt.Errorf("download alipay bill failed: %w", err)
	}

	externalRecords, err := r.parseBill(billData)
	if err != nil {
		return nil, fmt.Errorf("parse alipay bill failed: %w", err)
	}

	internalOrders, err := r.orderRepo.GetOrdersByDateAndApp(ctx, date, "alipay", appID)
	if err != nil {
		return nil, fmt.Errorf("get internal orders failed: %w", err)
	}

	result := r.compare(externalRecords, internalOrders)
	result.Date = date
	result.Channel = "alipay"
	result.AppID = appID
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
			TransactionID: row[0],                    // 交易号
			OrderNo:       row[1],                    // 商户订单号
			Amount:        parseAlipayAmount(row[9]), // 金额（元）
			Status:        row[11],                   // 交易状态
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
type AlipayBillDownloader struct {
	client     *alipay.Client
	httpClient *http.Client
}

func NewAlipayBillDownloader() *AlipayBillDownloader {
	client, _ := newAlipayBillClientFromEnv()
	return &AlipayBillDownloader{
		client: client,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (d *AlipayBillDownloader) Download(ctx context.Context, date time.Time) ([]byte, error) {
	if d.client == nil {
		return nil, fmt.Errorf("alipay bill downloader is not configured")
	}

	billDate := date.Format("2006-01-02")
	resp, err := d.client.BillDownloadURLQuery(ctx, alipay.BillDownloadURLQuery{
		BillType: "trade",
		BillDate: billDate,
	})
	if err != nil {
		return nil, fmt.Errorf("request alipay bill url failed: %w", err)
	}
	if !resp.IsSuccess() {
		return nil, fmt.Errorf("alipay bill url query failed: %s - %s", resp.Code, resp.Msg)
	}
	if resp.BillDownloadURL == "" {
		return nil, fmt.Errorf("alipay bill download url is empty")
	}

	fileResp, err := d.httpClient.Get(resp.BillDownloadURL)
	if err != nil {
		return nil, fmt.Errorf("download alipay bill file failed: %w", err)
	}
	defer fileResp.Body.Close()

	body, err := io.ReadAll(fileResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read alipay bill file failed: %w", err)
	}
	if bytes.HasPrefix(body, []byte{0x1f, 0x8b}) {
		reader, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("open gzip bill failed: %w", err)
		}
		defer reader.Close()
		return io.ReadAll(reader)
	}

	return body, nil
}

func newAlipayBillClientFromEnv() (*alipay.Client, error) {
	appID := os.Getenv("ALIPAY_APP_ID")
	privateKey, err := loadEnvValueOrFile("ALIPAY_APP_PRIVATE_KEY", "ALIPAY_APP_PRIVATE_KEY_PATH")
	if err != nil {
		return nil, err
	}
	if appID == "" || privateKey == "" {
		return nil, fmt.Errorf("alipay bill downloader env is incomplete")
	}

	isProduction := true
	var opts []alipay.OptionFunc
	if gateway := os.Getenv("ALIPAY_GATEWAY_URL"); gateway != "" {
		isProduction = strings.Contains(gateway, "openapi.alipay.com")
		if isProduction {
			opts = append(opts, alipay.WithProductionGateway(gateway))
		} else {
			opts = append(opts, alipay.WithSandboxGateway(gateway))
		}
	}

	client, err := alipay.New(appID, privateKey, isProduction, opts...)
	if err != nil {
		return nil, err
	}

	publicKey, err := loadEnvValueOrFile("ALIPAY_PUBLIC_KEY", "ALIPAY_PUBLIC_KEY_PATH")
	if err != nil {
		return nil, err
	}
	if publicKey != "" {
		if err := client.LoadAliPayPublicKey(publicKey); err != nil {
			return nil, err
		}
	}

	return client, nil
}

func loadEnvValueOrFile(valueKey, pathKey string) (string, error) {
	if value := os.Getenv(valueKey); value != "" {
		return value, nil
	}

	path := os.Getenv(pathKey)
	if path == "" {
		return "", nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s failed: %w", pathKey, err)
	}
	return strings.TrimSpace(string(content)), nil
}
