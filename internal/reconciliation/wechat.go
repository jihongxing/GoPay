package reconciliation

import (
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/option"
	"github.com/wechatpay-apiv3/wechatpay-go/utils"
)

// WechatReconciler 微信对账器
type WechatReconciler struct {
	billDownloader BillDownloader
	orderRepo      OrderRepository
}

// NewWechatReconciler 创建微信对账器
func NewWechatReconciler(db ...*sql.DB) *WechatReconciler {
	var repo OrderRepository = NewOrderRepository()
	if len(db) > 0 && db[0] != nil {
		repo = NewDBOrderRepository(db[0])
	}

	return &WechatReconciler{
		billDownloader: NewWechatBillDownloader(),
		orderRepo:      repo,
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

	if len(data) == 0 {
		return records, nil
	}

	// 创建 CSV reader
	reader := csv.NewReader(strings.NewReader(string(data)))
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
		// 微信账单格式：交易时间,公众账号ID,商户号,特约商户号,设备号,微信订单号,商户订单号,用户标识,交易类型,交易状态,付款银行,货币种类,应结订单金额,代金券金额,微信退款单号,商户退款单号,退款金额,充值券退款金额,退款类型,退款状态,商品名称,商户数据包,手续费,费率,订单金额,申请退款金额,费率备注
		if len(row) < 8 {
			continue
		}

		record := BillRecord{
			TransactionID: row[5],                     // 微信订单号
			OrderNo:       row[6],                     // 商户订单号
			Amount:        parseWechatAmount(row[12]), // 应结订单金额
			Status:        row[9],                     // 交易状态
			Channel:       "wechat",
		}

		records = append(records, record)
	}

	return records, nil
}

// parseWechatAmount 解析微信金额（元转分）
func parseWechatAmount(amountStr string) int64 {
	var amount float64
	fmt.Sscanf(amountStr, "%f", &amount)
	return int64(amount * 100)
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
type WechatBillDownloader struct {
	client     *core.Client
	httpClient *http.Client
}

func NewWechatBillDownloader() *WechatBillDownloader {
	client, _ := newWechatBillClientFromEnv()
	return &WechatBillDownloader{
		client: client,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (d *WechatBillDownloader) Download(ctx context.Context, date time.Time) ([]byte, error) {
	if d.client == nil {
		return nil, fmt.Errorf("wechat bill downloader is not configured")
	}

	billDate := date.Format("2006-01-02")
	apiURL := os.Getenv("WECHAT_API_URL")
	if apiURL == "" {
		apiURL = "https://api.mch.weixin.qq.com"
	}
	requestURL := fmt.Sprintf("%s/v3/bill/tradebill?bill_date=%s&bill_type=ALL",
		strings.TrimRight(apiURL, "/"), billDate)

	result, err := d.client.Get(ctx, requestURL)
	if err != nil {
		return nil, fmt.Errorf("request wechat bill url failed: %w", err)
	}
	defer result.Response.Body.Close()

	var resp struct {
		DownloadURL string `json:"download_url"`
	}
	if err := json.NewDecoder(result.Response.Body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("decode wechat bill response failed: %w", err)
	}
	if resp.DownloadURL == "" {
		return nil, fmt.Errorf("wechat bill download url is empty")
	}

	fileResp, err := d.httpClient.Get(resp.DownloadURL)
	if err != nil {
		return nil, fmt.Errorf("download wechat bill file failed: %w", err)
	}
	defer fileResp.Body.Close()

	body, err := io.ReadAll(fileResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read wechat bill file failed: %w", err)
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

func newWechatBillClientFromEnv() (*core.Client, error) {
	mchID := os.Getenv("WECHAT_MCH_ID")
	serialNo := os.Getenv("WECHAT_SERIAL_NO")
	apiV3Key := os.Getenv("WECHAT_API_V3_KEY")
	privateKeyPath := os.Getenv("WECHAT_PRIVATE_KEY_PATH")
	if mchID == "" || serialNo == "" || apiV3Key == "" || privateKeyPath == "" {
		return nil, fmt.Errorf("wechat bill downloader env is incomplete")
	}

	privateKey, err := utils.LoadPrivateKeyWithPath(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("load wechat private key failed: %w", err)
	}

	return core.NewClient(context.Background(), option.WithWechatPayAutoAuthCipher(mchID, serialNo, privateKey, apiV3Key))
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

// NewOrderRepository 创建订单仓储
func NewOrderRepository() OrderRepository {
	return &orderRepository{}
}

type orderRepository struct{}

func (r *orderRepository) GetOrdersByDate(ctx context.Context, date time.Time, channel string) ([]Order, error) {
	// 这个方法需要在初始化时注入数据库连接
	// 当前返回空列表，实际使用时需要通过构造函数注入 *sql.DB
	return []Order{}, nil
}
