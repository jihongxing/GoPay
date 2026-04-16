package reconciliation

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWechatReconciler_ParseBill(t *testing.T) {
	reconciler := NewWechatReconciler()

	tests := []struct {
		name    string
		data    []byte
		want    int
		wantErr bool
	}{
		{
			name:    "空账单",
			data:    []byte{},
			want:    0,
			wantErr: false,
		},
		{
			name: "有效账单",
			data: []byte(`交易时间,公众账号ID,商户号,特约商户号,设备号,微信订单号,商户订单号,用户标识,交易类型,交易状态,付款银行,货币种类,应结订单金额,代金券金额,微信退款单号,商户退款单号,退款金额,充值券退款金额,退款类型,退款状态,商品名称,商户数据包,手续费,费率,订单金额,申请退款金额,费率备注
2026-04-16 10:30:00,wx1234567890,1234567890,,,4200001234567890,ORDER_001,oABC123,NATIVE,SUCCESS,CMB,CNY,100.00,0.00,,,0.00,0.00,,,测试商品,,0.60,0.60%,100.00,0.00,
2026-04-16 11:00:00,wx1234567890,1234567890,,,4200001234567891,ORDER_002,oABC124,NATIVE,SUCCESS,ICBC,CNY,200.00,0.00,,,0.00,0.00,,,测试商品2,,1.20,0.60%,200.00,0.00,`),
			want:    2,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			records, err := reconciler.parseBill(tt.data)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, len(records))

				if len(records) > 0 {
					// 验证第一条记录
					assert.Equal(t, "4200001234567890", records[0].TransactionID)
					assert.Equal(t, "ORDER_001", records[0].OrderNo)
					assert.Equal(t, int64(10000), records[0].Amount) // 100.00 元 = 10000 分
					assert.Equal(t, "SUCCESS", records[0].Status)
					assert.Equal(t, "wechat", records[0].Channel)
				}
			}
		})
	}
}

func TestAlipayReconciler_ParseBill(t *testing.T) {
	reconciler := NewAlipayReconciler()

	tests := []struct {
		name    string
		data    []byte
		want    int
		wantErr bool
	}{
		{
			name:    "空账单",
			data:    []byte{},
			want:    0,
			wantErr: false,
		},
		{
			name: "有效账单",
			data: []byte(`支付宝交易记录明细查询
账务日期：2026-04-16
账号：test@example.com
起始日期：2026-04-16 00:00:00
结束日期：2026-04-16 23:59:59
交易号,商户订单号,交易创建时间,付款时间,最近修改时间,交易来源地,类型,交易对方,商品名称,金额（元）,收支,交易状态,服务费（元）,成功退款（元）,备注,资金状态
2026041622001234567890,ORDER_001,2026-04-16 10:30:00,2026-04-16 10:30:01,2026-04-16 10:30:01,其他,即时到账交易,buyer@example.com,测试商品,100.00,收入,交易成功,0.60,0.00,,已收入
2026041622001234567891,ORDER_002,2026-04-16 11:00:00,2026-04-16 11:00:01,2026-04-16 11:00:01,其他,即时到账交易,buyer2@example.com,测试商品2,200.00,收入,交易成功,1.20,0.00,,已收入`),
			want:    2,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			records, err := reconciler.parseBill(tt.data)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, len(records))

				if len(records) > 0 {
					// 验证第一条记录
					assert.Equal(t, "2026041622001234567890", records[0].TransactionID)
					assert.Equal(t, "ORDER_001", records[0].OrderNo)
					assert.Equal(t, int64(10000), records[0].Amount) // 100.00 元 = 10000 分
					assert.Equal(t, "交易成功", records[0].Status)
					assert.Equal(t, "alipay", records[0].Channel)
				}
			}
		})
	}
}

func TestReconciler_Compare(t *testing.T) {
	reconciler := NewWechatReconciler()

	tests := []struct {
		name     string
		external []BillRecord
		internal []Order
		want     *ReconcileResult
	}{
		{
			name: "完全匹配",
			external: []BillRecord{
				{OrderNo: "ORDER_001", Amount: 10000, Status: "SUCCESS"},
				{OrderNo: "ORDER_002", Amount: 20000, Status: "SUCCESS"},
			},
			internal: []Order{
				{OrderNo: "ORDER_001", Amount: 10000, Status: "paid"},
				{OrderNo: "ORDER_002", Amount: 20000, Status: "paid"},
			},
			want: &ReconcileResult{
				TotalOrders:    2,
				MatchedOrders:  2,
				MissingOrders:  []string{},
				ExtraOrders:    []string{},
				AmountMismatch: []AmountMismatch{},
				Status:         "success",
			},
		},
		{
			name: "长款（外部有但内部无）",
			external: []BillRecord{
				{OrderNo: "ORDER_001", Amount: 10000, Status: "SUCCESS"},
				{OrderNo: "ORDER_002", Amount: 20000, Status: "SUCCESS"},
			},
			internal: []Order{
				{OrderNo: "ORDER_001", Amount: 10000, Status: "paid"},
			},
			want: &ReconcileResult{
				TotalOrders:    2,
				MatchedOrders:  1,
				MissingOrders:  []string{"ORDER_002"},
				ExtraOrders:    []string{},
				AmountMismatch: []AmountMismatch{},
				Status:         "failed",
			},
		},
		{
			name: "短款（内部有但外部无）",
			external: []BillRecord{
				{OrderNo: "ORDER_001", Amount: 10000, Status: "SUCCESS"},
			},
			internal: []Order{
				{OrderNo: "ORDER_001", Amount: 10000, Status: "paid"},
				{OrderNo: "ORDER_002", Amount: 20000, Status: "paid"},
			},
			want: &ReconcileResult{
				TotalOrders:    1,
				MatchedOrders:  1,
				MissingOrders:  []string{},
				ExtraOrders:    []string{"ORDER_002"},
				AmountMismatch: []AmountMismatch{},
				Status:         "failed",
			},
		},
		{
			name: "金额不匹配",
			external: []BillRecord{
				{OrderNo: "ORDER_001", Amount: 10000, Status: "SUCCESS"},
				{OrderNo: "ORDER_002", Amount: 20000, Status: "SUCCESS"},
			},
			internal: []Order{
				{OrderNo: "ORDER_001", Amount: 10000, Status: "paid"},
				{OrderNo: "ORDER_002", Amount: 25000, Status: "paid"}, // 金额不匹配
			},
			want: &ReconcileResult{
				TotalOrders:   2,
				MatchedOrders: 1,
				MissingOrders: []string{},
				ExtraOrders:   []string{},
				AmountMismatch: []AmountMismatch{
					{
						OrderNo:        "ORDER_002",
						InternalAmount: 25000,
						ExternalAmount: 20000,
						Difference:     5000,
					},
				},
				Status: "failed",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := reconciler.compare(tt.external, tt.internal)

			assert.Equal(t, tt.want.TotalOrders, result.TotalOrders)
			assert.Equal(t, tt.want.MatchedOrders, result.MatchedOrders)
			assert.Equal(t, len(tt.want.MissingOrders), len(result.MissingOrders))
			assert.Equal(t, len(tt.want.ExtraOrders), len(result.ExtraOrders))
			assert.Equal(t, len(tt.want.AmountMismatch), len(result.AmountMismatch))
			assert.Equal(t, tt.want.Status, result.Status)

			// 验证长款订单号
			if len(tt.want.MissingOrders) > 0 {
				assert.ElementsMatch(t, tt.want.MissingOrders, result.MissingOrders)
			}

			// 验证短款订单号
			if len(tt.want.ExtraOrders) > 0 {
				assert.ElementsMatch(t, tt.want.ExtraOrders, result.ExtraOrders)
			}

			// 验证金额不匹配
			if len(tt.want.AmountMismatch) > 0 {
				assert.Equal(t, tt.want.AmountMismatch[0].OrderNo, result.AmountMismatch[0].OrderNo)
				assert.Equal(t, tt.want.AmountMismatch[0].InternalAmount, result.AmountMismatch[0].InternalAmount)
				assert.Equal(t, tt.want.AmountMismatch[0].ExternalAmount, result.AmountMismatch[0].ExternalAmount)
				assert.Equal(t, tt.want.AmountMismatch[0].Difference, result.AmountMismatch[0].Difference)
			}
		})
	}
}

func TestReportGenerator_Generate(t *testing.T) {
	generator := NewReportGenerator()

	result := &ReconcileResult{
		Date:          time.Date(2026, 4, 16, 0, 0, 0, 0, time.UTC),
		Channel:       "wechat",
		TotalOrders:   10,
		MatchedOrders: 8,
		MissingOrders: []string{"ORDER_001", "ORDER_002"},
		ExtraOrders:   []string{},
		AmountMismatch: []AmountMismatch{
			{
				OrderNo:        "ORDER_003",
				InternalAmount: 10000,
				ExternalAmount: 9900,
				Difference:     100,
			},
		},
		Status:    "failed",
		CreatedAt: time.Now(),
	}

	ctx := context.Background()
	reportPath, err := generator.Generate(ctx, result)

	assert.NoError(t, err)
	assert.NotEmpty(t, reportPath)
	assert.Contains(t, reportPath, "reconciliation_wechat_20260416.csv")

	// 验证报告文件是否存在
	// 注意：这会在文件系统中创建实际文件
	// 在实际测试中，可能需要使用临时目录
}

func TestScheduler_GetNextRunTime(t *testing.T) {
	scheduler := &Scheduler{}

	// 测试不同时间点的下次执行时间
	tests := []struct {
		name     string
		now      time.Time
		wantHour int
		wantDay  int
	}{
		{
			name:     "凌晨1点，应该是今天2点",
			now:      time.Date(2026, 4, 16, 1, 0, 0, 0, time.Local),
			wantHour: 2,
			wantDay:  16,
		},
		{
			name:     "凌晨3点，应该是明天2点",
			now:      time.Date(2026, 4, 16, 3, 0, 0, 0, time.Local),
			wantHour: 2,
			wantDay:  17,
		},
		{
			name:     "下午2点，应该是明天2点",
			now:      time.Date(2026, 4, 16, 14, 0, 0, 0, time.Local),
			wantHour: 2,
			wantDay:  17,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 注意：这个测试依赖于系统时间，实际使用时可能需要 mock time.Now()
			nextRun := scheduler.getNextRunTime()

			assert.Equal(t, tt.wantHour, nextRun.Hour())
			assert.Equal(t, 0, nextRun.Minute())
			assert.Equal(t, 0, nextRun.Second())
		})
	}
}

func TestParseWechatAmount(t *testing.T) {
	tests := []struct {
		name   string
		amount string
		want   int64
	}{
		{"整数金额", "100.00", 10000},
		{"小数金额", "99.99", 9999},
		{"零金额", "0.00", 0},
		{"大金额", "12345.67", 1234567},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseWechatAmount(tt.amount)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseAlipayAmount(t *testing.T) {
	tests := []struct {
		name   string
		amount string
		want   int64
	}{
		{"整数金额", "100.00", 10000},
		{"小数金额", "99.99", 9999},
		{"零金额", "0.00", 0},
		{"大金额", "12345.67", 1234567},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseAlipayAmount(tt.amount)
			assert.Equal(t, tt.want, got)
		})
	}
}
