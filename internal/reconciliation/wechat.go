package reconciliation

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"time"
)

// WechatReconciler 微信对账器
type WechatReconciler struct {
	billDownloader *WechatBillDownloader
	orderRepo      OrderRepository
}

// NewWechatReconciler 创建微信对账器
func NewWechatReconciler() *WechatReconciler {
	return &WechatReconciler{
		billDownloader: NewWechatBillDownloader(),
		orderRepo:      NewOrderRepository(),
	}
}

// Reconcile 执行微信对账
func (r *WechatReconciler) Reconcile(ctx context.Context, date time.Time) (*ReconcileResult, error) {
	// 1. 下载微信账单
	billData, err := r.billDownloader.Download(ctx, date)
	if err != nil {
		return nil, fmt.Errorf("download wechat bill failed: %w", err)
	}

	// 2. 解析账单
	externalRecords, err := r.parseBill(billData)
	if err != nil {
		return nil, fmt.Errorf("parse wechat bill failed: %w", err)
	}

	// 3. 获取内部订单
	internalOrders, err := r.orderRepo.GetOrdersByDate(ctx, date, "wechat")
	if err != nil {
		return nil, fmt.Errorf("get internal orders failed: %w", err)
	}

	// 4. 双向比对
	result := r.compare(externalRecords, internalOrders)
	result.Date = date
	result.Channel = "wechat"
	result.CreatedAt = time.Now()

	return result, nil
}

// parseBill 解析微信账单
func (r *WechatReconciler) parseBill(data []byte) ([]BillRecord, error) {
	var records []BillRecord

	reader := csv.NewReader(io.Reader(nil)) // TODO: 实现 CSV 读取
	reader.Comma = ','

	// 跳过表头
	_, err := reader.Read()
	if err != nil {
		return nil, err
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
		record := BillRecord{
			TransactionID: row[0],
			OrderNo:       row[1],
			// Amount:        parseAmount(row[2]),
			Status:  row[3],
			Channel: "wechat",
		}

		records = append(records, record)
	}

	return records, nil
}

// compare 比对内外部数据
func (r *WechatReconciler) compare(external []BillRecord, internal []Order) *ReconcileResult {
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

// WechatBillDownloader 微信账单下载器
type WechatBillDownloader struct{}

func NewWechatBillDownloader() *WechatBillDownloader {
	return &WechatBillDownloader{}
}

func (d *WechatBillDownloader) Download(ctx context.Context, date time.Time) ([]byte, error) {
	// TODO: 调用微信 API 下载账单
	// https://pay.weixin.qq.com/wiki/doc/apiv3/apis/chapter3_1_6.shtml
	return nil, nil
}

// OrderRepository 订单仓储接口
type OrderRepository interface {
	GetOrdersByDate(ctx context.Context, date time.Time, channel string) ([]Order, error)
}

type Order struct {
	OrderNo string
	Amount  int64
	Status  string
	PaidAt  time.Time
}

func NewOrderRepository() OrderRepository {
	return &orderRepository{}
}

type orderRepository struct{}

func (r *orderRepository) GetOrdersByDate(ctx context.Context, date time.Time, channel string) ([]Order, error) {
	// TODO: 从数据库查询订单
	return nil, nil
}
