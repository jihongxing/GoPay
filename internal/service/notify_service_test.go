package service

import (
	"context"
	"testing"
	"time"

	"gopay/internal/models"
)

// TestNotifyService_BuildNotifyRequest 测试构建通知请求
func TestNotifyService_BuildNotifyRequest(t *testing.T) {
	service := &NotifyService{}

	paidAt := time.Now()
	order := &models.Order{
		OrderNo:        "ORD_20260416_001",
		OutTradeNo:     "TEST_ORDER_001",
		Amount:         10000,
		Status:         models.OrderStatusPaid,
		PaidAt:         &paidAt,
		Channel:        "wechat_native",
		ChannelOrderNo: "WX_20260416_001",
	}

	req := service.buildNotifyRequest(order)

	if req.OrderNo != order.OrderNo {
		t.Errorf("OrderNo = %v, want %v", req.OrderNo, order.OrderNo)
	}
	if req.OutTradeNo != order.OutTradeNo {
		t.Errorf("OutTradeNo = %v, want %v", req.OutTradeNo, order.OutTradeNo)
	}
	if req.Amount != order.Amount {
		t.Errorf("Amount = %v, want %v", req.Amount, order.Amount)
	}
	if req.Status != order.Status {
		t.Errorf("Status = %v, want %v", req.Status, order.Status)
	}
	if req.Channel != order.Channel {
		t.Errorf("Channel = %v, want %v", req.Channel, order.Channel)
	}
	if req.ChannelOrderNo != order.ChannelOrderNo {
		t.Errorf("ChannelOrderNo = %v, want %v", req.ChannelOrderNo, order.ChannelOrderNo)
	}
	if req.PaidAt == "" {
		t.Error("PaidAt should not be empty")
	}
}

// TestNotifyService_GetErrorMsg 测试获取错误信息
func TestNotifyService_GetErrorMsg(t *testing.T) {
	service := &NotifyService{}

	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "nil error",
			err:  nil,
			want: "",
		},
		{
			name: "with error",
			err:  context.DeadlineExceeded,
			want: "context deadline exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := service.getErrorMsg(tt.err)
			if got != tt.want {
				t.Errorf("getErrorMsg() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestNotifyService_WorkerPool 测试工作池限流
func TestNotifyService_WorkerPool(t *testing.T) {
	// 创建一个小容量的工作池用于测试
	service := &NotifyService{
		workerPool: make(chan struct{}, 2), // 只允许 2 个并发
	}

	order := &models.Order{
		OrderNo:    "ORD_TEST_001",
		AppID:      "test_app",
		OutTradeNo: "TEST_001",
	}

	// 提交 5 个任务，但只有 2 个能并发执行
	submitted := 0
	for i := 0; i < 5; i++ {
		select {
		case service.workerPool <- struct{}{}:
			submitted++
			<-service.workerPool // 立即释放
		default:
			// 工作池已满
		}
	}

	// 应该只有 2 个任务能提交
	if submitted != 2 {
		t.Errorf("Expected 2 tasks submitted, got %d", submitted)
	}

	// 测试 NotifyAsync 不会阻塞
	service.NotifyAsync(order)
	service.NotifyAsync(order)
	service.NotifyAsync(order) // 第三个应该被拒绝但不阻塞

	// 如果执行到这里没有阻塞，说明测试通过
}
