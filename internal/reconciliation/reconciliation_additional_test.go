package reconciliation

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

type fakeDownloader struct {
	data []byte
	err  error
}

func (f *fakeDownloader) Download(ctx context.Context, date time.Time) ([]byte, error) {
	return f.data, f.err
}

type fakeOrderRepo struct {
	orders []Order
	err    error
}

func (f *fakeOrderRepo) GetOrdersByDate(ctx context.Context, date time.Time, channel string) ([]Order, error) {
	return f.orders, f.err
}

type fakeAlertNotifier struct {
	messages []string
}

func (f *fakeAlertNotifier) SendAlert(ctx context.Context, message string) error {
	f.messages = append(f.messages, message)
	return nil
}

func TestWechatReconciler_Reconcile_Success(t *testing.T) {
	date := time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC)
	csvData := []byte(strings.Join([]string{
		"交易时间,公众账号ID,商户号,特约商户号,设备号,微信订单号,商户订单号,用户标识,交易类型,交易状态,付款银行,货币种类,应结订单金额,代金券金额,微信退款单号,商户退款单号,退款金额,充值券退款金额,退款类型,退款状态,商品名称,商户数据包,手续费,费率,订单金额,申请退款金额,费率备注",
		"2026-04-16 10:30:00,wx1234567890,1234567890,,,4200001234567890,ORDER_1,oABC123,NATIVE,SUCCESS,CMB,CNY,100.00,0.00,,,0.00,0.00,,,测试商品,,0.60,0.60%,100.00,0.00,",
	}, "\n"))

	reconciler := &WechatReconciler{
		billDownloader: &fakeDownloader{data: csvData},
		orderRepo: &fakeOrderRepo{orders: []Order{
			{OrderNo: "ORDER_1", Amount: 10000, Status: "paid"},
		}},
	}

	result, err := reconciler.Reconcile(context.Background(), date)
	assert.NoError(t, err)
	assert.Equal(t, "wechat", result.Channel)
	assert.Equal(t, "success", result.Status)
	assert.Equal(t, 1, result.TotalOrders)
	assert.Equal(t, 1, result.MatchedOrders)
}

func TestWechatReconciler_Reconcile_DownloadError(t *testing.T) {
	reconciler := &WechatReconciler{
		billDownloader: &fakeDownloader{err: assert.AnError},
		orderRepo:      &fakeOrderRepo{},
	}

	result, err := reconciler.Reconcile(context.Background(), time.Now())
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "download wechat bill failed")
}

func TestAlipayReconciler_Reconcile_Success(t *testing.T) {
	date := time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC)
	data := []byte(strings.Join([]string{
		"支付宝交易记录明细查询",
		"账务日期：2026-04-16",
		"账号：test@example.com",
		"起始日期：2026-04-16 00:00:00",
		"结束日期：2026-04-16 23:59:59",
		"交易号,商户订单号,交易创建时间,付款时间,最近修改时间,交易来源地,类型,交易对方,商品名称,金额（元）,收支,交易状态,服务费（元）,成功退款（元）,备注,资金状态",
		"2026041622001234567890,ORDER_1,2026-04-16 10:30:00,2026-04-16 10:30:01,2026-04-16 10:30:01,其他,即时到账交易,buyer@example.com,测试商品,100.00,收入,交易成功,0.60,0.00,,已收入",
	}, "\n"))

	reconciler := &AlipayReconciler{
		billDownloader: &fakeDownloader{data: data},
		orderRepo: &fakeOrderRepo{orders: []Order{
			{OrderNo: "ORDER_1", Amount: 10000, Status: "paid"},
		}},
	}

	result, err := reconciler.Reconcile(context.Background(), date)
	assert.NoError(t, err)
	assert.Equal(t, "alipay", result.Channel)
	assert.Equal(t, "success", result.Status)
	assert.Equal(t, 1, result.TotalOrders)
	assert.Equal(t, 1, result.MatchedOrders)
}

func TestReconciliationService_ReconcileAll(t *testing.T) {
	date := time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC)
	wechatData := []byte(strings.Join([]string{
		"交易时间,公众账号ID,商户号,特约商户号,设备号,微信订单号,商户订单号,用户标识,交易类型,交易状态,付款银行,货币种类,应结订单金额,代金券金额,微信退款单号,商户退款单号,退款金额,充值券退款金额,退款类型,退款状态,商品名称,商户数据包,手续费,费率,订单金额,申请退款金额,费率备注",
		"2026-04-16 10:30:00,wx1234567890,1234567890,,,4200001234567890,ORDER_1,oABC123,NATIVE,SUCCESS,CMB,CNY,100.00,0.00,,,0.00,0.00,,,测试商品,,0.60,0.60%,100.00,0.00,",
	}, "\n"))
	alipayData := []byte(strings.Join([]string{
		"支付宝交易记录明细查询",
		"账务日期：2026-04-16",
		"账号：test@example.com",
		"起始日期：2026-04-16 00:00:00",
		"结束日期：2026-04-16 23:59:59",
		"交易号,商户订单号,交易创建时间,付款时间,最近修改时间,交易来源地,类型,交易对方,商品名称,金额（元）,收支,交易状态,服务费（元）,成功退款（元）,备注,资金状态",
		"2026041622001234567890,ORDER_1,2026-04-16 10:30:00,2026-04-16 10:30:01,2026-04-16 10:30:01,其他,即时到账交易,buyer@example.com,测试商品,100.00,收入,交易成功,0.60,0.00,,已收入",
	}, "\n"))

	service := &ReconciliationService{
		wechatReconciler: &WechatReconciler{
			billDownloader: &fakeDownloader{data: wechatData},
			orderRepo:      &fakeOrderRepo{orders: []Order{{OrderNo: "ORDER_1", Amount: 10000, Status: "paid"}}},
		},
		alipayReconciler: &AlipayReconciler{
			billDownloader: &fakeDownloader{data: alipayData},
			orderRepo:      &fakeOrderRepo{orders: []Order{{OrderNo: "ORDER_1", Amount: 10000, Status: "paid"}}},
		},
		reportGenerator: &ReportGenerator{reportDir: t.TempDir()},
	}

	results, err := service.ReconcileAll(context.Background(), date)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, "wechat", results[0].Channel)
	assert.Equal(t, "alipay", results[1].Channel)
}

func TestReportGenerator_GenerateExtraOrders(t *testing.T) {
	dir := t.TempDir()
	generator := &ReportGenerator{reportDir: dir}
	result := &ReconcileResult{
		Date:          time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC),
		Channel:       "wechat",
		TotalOrders:   1,
		MatchedOrders: 0,
		ExtraOrders:   []string{"ORDER_2"},
		Status:        "failed",
		CreatedAt:     time.Now(),
	}

	path, err := generator.Generate(context.Background(), result)
	assert.NoError(t, err)
	assert.FileExists(t, path)

	content, err := os.ReadFile(path)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "短款明细")
}

func TestReportGenerator_GenerateExcel_NotImplemented(t *testing.T) {
	generator := NewReportGenerator()
	_, err := generator.GenerateExcel(context.Background(), &ReconcileResult{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not implemented")
}

func TestDBOrderRepository_GetOrdersByDate(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := NewDBOrderRepository(db)
	now := time.Date(2026, 4, 16, 10, 30, 0, 0, time.UTC)
	rows := sqlmock.NewRows([]string{"order_no", "amount", "status", "paid_at"}).
		AddRow("ORDER_1", 10000, "paid", now)
	mock.ExpectQuery("SELECT order_no, amount, status, paid_at FROM orders").
		WithArgs("wechat%", time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC), time.Date(2026, 4, 17, 0, 0, 0, 0, time.UTC)).
		WillReturnRows(rows)

	orders, err := repo.GetOrdersByDate(context.Background(), time.Date(2026, 4, 16, 12, 0, 0, 0, time.UTC), "wechat")
	assert.NoError(t, err)
	assert.Len(t, orders, 1)
	assert.Equal(t, "ORDER_1", orders[0].OrderNo)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestScheduler_ReconcileDate_WithDifference(t *testing.T) {
	date := time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC)
	wechatData := []byte(strings.Join([]string{
		"交易时间,公众账号ID,商户号,特约商户号,设备号,微信订单号,商户订单号,用户标识,交易类型,交易状态,付款银行,货币种类,应结订单金额,代金券金额,微信退款单号,商户退款单号,退款金额,充值券退款金额,退款类型,退款状态,商品名称,商户数据包,手续费,费率,订单金额,申请退款金额,费率备注",
		"2026-04-16 10:30:00,wx1234567890,1234567890,,,4200001234567890,ORDER_1,oABC123,NATIVE,SUCCESS,CMB,CNY,100.00,0.00,,,0.00,0.00,,,测试商品,,0.60,0.60%,100.00,0.00,",
	}, "\n"))
	alipayData := []byte(strings.Join([]string{
		"支付宝交易记录明细查询",
		"账务日期：2026-04-16",
		"账号：test@example.com",
		"起始日期：2026-04-16 00:00:00",
		"结束日期：2026-04-16 23:59:59",
		"交易号,商户订单号,交易创建时间,付款时间,最近修改时间,交易来源地,类型,交易对方,商品名称,金额（元）,收支,交易状态,服务费（元）,成功退款（元）,备注,资金状态",
		"2026041622001234567890,ORDER_2,2026-04-16 10:30:00,2026-04-16 10:30:01,2026-04-16 10:30:01,其他,即时到账交易,buyer@example.com,测试商品,100.00,收入,交易成功,0.60,0.00,,已收入",
	}, "\n"))

	notifier := &fakeAlertNotifier{}
	scheduler := &Scheduler{
		service: &ReconciliationService{
			wechatReconciler: &WechatReconciler{
				billDownloader: &fakeDownloader{data: wechatData},
				orderRepo:      &fakeOrderRepo{orders: []Order{{OrderNo: "ORDER_1", Amount: 10000, Status: "paid"}}},
			},
			alipayReconciler: &AlipayReconciler{
				billDownloader: &fakeDownloader{data: alipayData},
				orderRepo:      &fakeOrderRepo{orders: []Order{}},
			},
			reportGenerator: &ReportGenerator{reportDir: t.TempDir()},
		},
		alertNotifier: notifier,
		stopCh:        make(chan struct{}),
	}

	err := scheduler.reconcileDate(context.Background(), date)
	assert.Error(t, err)
	assert.Len(t, notifier.messages, 1)
	assert.Contains(t, notifier.messages[0], "对账异常告警")
	assert.Contains(t, notifier.messages[0], "alipay")
}

func TestScheduler_StartAlreadyRunning(t *testing.T) {
	scheduler := &Scheduler{
		running: true,
		stopCh:  make(chan struct{}),
	}

	err := scheduler.Start(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")
}
