package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestOrder_Validate 测试订单验证
func TestOrder_Validate(t *testing.T) {
	tests := []struct {
		name    string
		order   Order
		wantErr bool
	}{
		{
			name: "有效订单",
			order: Order{
				AppID:      "test_app_id",
				OutTradeNo: "ORDER_001",
				Amount:     100,
				Subject:    "测试商品",
				Channel:    "wechat_native",
				Status:     "pending",
			},
			wantErr: false,
		},
		{
			name: "金额为0",
			order: Order{
				AppID:      "test_app_id",
				OutTradeNo: "ORDER_002",
				Amount:     0,
				Subject:    "测试商品",
				Channel:    "wechat_native",
			},
			wantErr: true,
		},
		{
			name: "缺少标题",
			order: Order{
				AppID:      "test_app_id",
				OutTradeNo: "ORDER_003",
				Amount:     100,
				Subject:    "",
				Channel:    "wechat_native",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.order.Validate()
			if tt.wantErr {
				assert.Error(t, err, "应该返回错误")
			} else {
				assert.NoError(t, err, "不应该返回错误")
			}
		})
	}
}

// TestOrder_IsPaid 测试订单是否已支付
func TestOrder_IsPaid(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name   string
		order  Order
		expect bool
	}{
		{
			name: "已支付订单",
			order: Order{
				Status: "paid",
				PaidAt: &now,
			},
			expect: true,
		},
		{
			name: "待支付订单",
			order: Order{
				Status: "pending",
			},
			expect: false,
		},
		{
			name: "失败订单",
			order: Order{
				Status: "failed",
			},
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.order.IsPaid()
			assert.Equal(t, tt.expect, result, "IsPaid() 返回值不符合预期")
		})
	}
}
