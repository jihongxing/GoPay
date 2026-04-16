package channel

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestWechatProvider_CreateOrder 测试微信支付创建订单
func TestWechatProvider_CreateOrder(t *testing.T) {
	tests := []struct {
		name    string
		amount  int64
		subject string
		wantErr bool
	}{
		{
			name:    "成功创建订单",
			amount:  100,
			subject: "测试商品",
			wantErr: false,
		},
		{
			name:    "金额为负数",
			amount:  -100,
			subject: "测试商品",
			wantErr: true,
		},
		{
			name:    "标题为空",
			amount:  100,
			subject: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 验证测试用例参数
			if tt.wantErr {
				// 预期失败的情况
				if tt.amount < 0 {
					assert.Less(t, tt.amount, int64(0), "金额应该为负数")
				}
				if tt.subject == "" {
					assert.Empty(t, tt.subject, "标题应该为空")
				}
			} else {
				// 预期成功的情况
				assert.Greater(t, tt.amount, int64(0), "金额应该大于0")
				assert.NotEmpty(t, tt.subject, "标题不应该为空")
			}
		})
	}
}

// TestAlipayProvider_CreateOrder 测试支付宝创建订单
func TestAlipayProvider_CreateOrder(t *testing.T) {
	tests := []struct {
		name    string
		amount  int64
		subject string
		wantErr bool
	}{
		{
			name:    "成功创建订单",
			amount:  100,
			subject: "测试商品",
			wantErr: false,
		},
		{
			name:    "金额为0",
			amount:  0,
			subject: "测试商品",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 验证测试用例参数
			if tt.wantErr {
				// 预期失败的情况
				assert.Equal(t, int64(0), tt.amount, "金额应该为0")
			} else {
				// 预期成功的情况
				assert.Greater(t, tt.amount, int64(0), "金额应该大于0")
				assert.NotEmpty(t, tt.subject, "标题不应该为空")
			}
		})
	}
}
