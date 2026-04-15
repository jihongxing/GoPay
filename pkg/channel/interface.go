package channel

import (
	"context"
	"time"
)

// PaymentChannel 支付渠道接口
type PaymentChannel interface {
	// Name 返回渠道名称
	Name() string

	// CreateOrder 创建支付订单
	CreateOrder(ctx context.Context, req *CreateOrderRequest) (*CreateOrderResponse, error)

	// QueryOrder 查询订单状态
	QueryOrder(ctx context.Context, req *QueryOrderRequest) (*QueryOrderResponse, error)

	// HandleWebhook 处理支付平台的回调通知
	HandleWebhook(ctx context.Context, req *WebhookRequest) (*WebhookResponse, error)

	// Close 关闭资源
	Close() error
}

// CreateOrderRequest 创建订单请求
type CreateOrderRequest struct {
	OrderID     string            // GoPay 内部订单 ID
	BizOrderNo  string            // 业务订单号
	Amount      int64             // 金额（分）
	Subject     string            // 订单标题
	Description string            // 订单描述
	NotifyURL   string            // 支付平台回调地址
	ExtraData   map[string]string // 额外参数
}

// CreateOrderResponse 创建订单响应
type CreateOrderResponse struct {
	PlatformTradeNo string            // 第三方平台交易号
	PayURL          string            // 支付链接（Native、H5）
	QRCode          string            // 二维码内容（Native）
	PrepayID        string            // 预支付 ID（JSAPI、APP）
	ExtraData       map[string]string // 额外返回数据（JSAPI 调起支付参数等）
}

// QueryOrderRequest 查询订单请求
type QueryOrderRequest struct {
	OrderID         string // GoPay 内部订单 ID
	PlatformTradeNo string // 第三方平台交易号
}

// QueryOrderResponse 查询订单响应
type QueryOrderResponse struct {
	Status          OrderStatus // 订单状态
	PlatformTradeNo string      // 第三方平台交易号
	PaidAmount      int64       // 实付金额（分）
	PaidAt          *time.Time  // 支付时间
	ExtraData       map[string]string
}

// WebhookRequest Webhook 请求
type WebhookRequest struct {
	RawBody []byte            // 原始请求体
	Headers map[string]string // 请求头
}

// WebhookResponse Webhook 响应
type WebhookResponse struct {
	Success         bool       // 是否成功
	OrderID         string     // GoPay 内部订单 ID
	PlatformTradeNo string     // 第三方平台交易号
	Status          OrderStatus // 订单状态
	PaidAmount      int64      // 实付金额（分）
	PaidAt          time.Time  // 支付时间
	ResponseBody    []byte     // 返回给支付平台的响应体
}

// OrderStatus 订单状态
type OrderStatus string

const (
	OrderStatusPending OrderStatus = "PENDING" // 待支付
	OrderStatusPaid    OrderStatus = "PAID"    // 已支付
	OrderStatusClosed  OrderStatus = "CLOSED"  // 已关闭
	OrderStatusRefund  OrderStatus = "REFUND"  // 已退款
)

// NotifyStatus 通知状态
type NotifyStatus string

const (
	NotifyStatusPending NotifyStatus = "PENDING"       // 待通知
	NotifyStatusSuccess NotifyStatus = "SUCCESS"       // 通知成功
	NotifyStatusFailed  NotifyStatus = "FAILED_NOTIFY" // 通知失败
)
