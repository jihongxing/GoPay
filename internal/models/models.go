package models

import (
	"fmt"
	"time"
)

// App 应用表模型
type App struct {
	ID          int64     `json:"id" db:"id"`
	AppID       string    `json:"app_id" db:"app_id"`
	AppName     string    `json:"app_name" db:"app_name"`
	AppSecret   string    `json:"-" db:"app_secret"` // 不返回给前端
	CallbackURL string    `json:"callback_url" db:"callback_url"`
	Status      string    `json:"status" db:"status"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// Order 订单表模型
type Order struct {
	ID             int64      `json:"id" db:"id"`
	OrderNo        string     `json:"order_no" db:"order_no"`
	AppID          string     `json:"app_id" db:"app_id"`
	OutTradeNo     string     `json:"out_trade_no" db:"out_trade_no"`
	Channel        string     `json:"channel" db:"channel"`
	Amount         int64      `json:"amount" db:"amount"`
	Currency       string     `json:"currency" db:"currency"`
	Subject        string     `json:"subject" db:"subject"`
	Body           string     `json:"body" db:"body"`
	Status         string     `json:"status" db:"status"`
	NotifyStatus   string     `json:"notify_status" db:"notify_status"`
	RetryCount     int        `json:"retry_count" db:"retry_count"`
	ChannelOrderNo string     `json:"channel_order_no" db:"channel_order_no"`
	PayURL         string     `json:"pay_url" db:"pay_url"`
	PaidAt         *time.Time `json:"paid_at" db:"paid_at"`
	NotifiedAt     *time.Time `json:"notified_at" db:"notified_at"`
	ExpiresAt      time.Time  `json:"expires_at" db:"expires_at"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}

// Transaction 交易流水表模型
type Transaction struct {
	ID             int64     `json:"id" db:"id"`
	TransactionNo  string    `json:"transaction_no" db:"transaction_no"`
	OrderNo        string    `json:"order_no" db:"order_no"`
	Channel        string    `json:"channel" db:"channel"`
	ChannelOrderNo string    `json:"channel_order_no" db:"channel_order_no"`
	Type           string    `json:"type" db:"type"`
	Amount         int64     `json:"amount" db:"amount"`
	Status         string    `json:"status" db:"status"`
	RawRequest     string    `json:"raw_request" db:"raw_request"`
	RawResponse    string    `json:"raw_response" db:"raw_response"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

// ChannelConfig 支付渠道配置表模型
type ChannelConfig struct {
	ID        int64     `json:"id" db:"id"`
	AppID     string    `json:"app_id" db:"app_id"`
	Channel   string    `json:"channel" db:"channel"`
	Config    string    `json:"config" db:"config"` // JSONB 存储为字符串
	Status    string    `json:"status" db:"status"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// NotifyLog 通知日志表模型
type NotifyLog struct {
	ID             int64     `json:"id" db:"id"`
	OrderNo        string    `json:"order_no" db:"order_no"`
	CallbackURL    string    `json:"callback_url" db:"callback_url"`
	RequestBody    string    `json:"request_body" db:"request_body"`
	ResponseStatus int       `json:"response_status" db:"response_status"`
	ResponseBody   string    `json:"response_body" db:"response_body"`
	Success        bool      `json:"success" db:"success"`
	ErrorMsg       string    `json:"error_msg" db:"error_msg"`
	DurationMs     int       `json:"duration_ms" db:"duration_ms"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

// 订单状态常量
const (
	OrderStatusPending  = "pending"
	OrderStatusPaid     = "paid"
	OrderStatusClosed   = "closed"
	OrderStatusRefunded = "refunded"
)

// 通知状态常量
const (
	NotifyStatusPending     = "pending"
	NotifyStatusNotified    = "notified"
	NotifyStatusFailedNotify = "failed_notify"
)

// 支付渠道常量
const (
	ChannelWechatNative = "wechat_native" // 微信 Native 扫码支付
	ChannelWechatJSAPI  = "wechat_jsapi"  // 微信 JSAPI 支付（公众号/小程序）
	ChannelWechatH5     = "wechat_h5"     // 微信 H5 支付（手机网页）
	ChannelWechatApp    = "wechat_app"    // 微信 APP 支付
	ChannelAlipayQR     = "alipay_qr"     // 支付宝扫码支付（PC 网站）
	ChannelAlipayWap    = "alipay_wap"    // 支付宝手机网站支付（H5）
	ChannelAlipayApp    = "alipay_app"    // 支付宝 APP 支付
	ChannelAlipayFace   = "alipay_face"   // 支付宝当面付（线下扫码）
)

// 交易类型常量
const (
	TransactionTypePayment = "payment"
	TransactionTypeRefund  = "refund"
)

// ConfigAuditLog 配置变更审计日志表模型
type ConfigAuditLog struct {
	ID           int64     `json:"id" db:"id"`
	Operator     string    `json:"operator" db:"operator"`
	Action       string    `json:"action" db:"action"`
	ResourceType string    `json:"resource_type" db:"resource_type"`
	ResourceID   string    `json:"resource_id" db:"resource_id"`
	OldValue     string    `json:"old_value" db:"old_value"` // JSONB 存储为字符串
	NewValue     string    `json:"new_value" db:"new_value"` // JSONB 存储为字符串
	IPAddress    string    `json:"ip_address" db:"ip_address"`
	UserAgent    string    `json:"user_agent" db:"user_agent"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// 审计日志操作类型常量
const (
	AuditActionCreate  = "create"
	AuditActionUpdate  = "update"
	AuditActionDelete  = "delete"
	AuditActionDisable = "disable"
)

// 审计日志资源类型常量
const (
	AuditResourceApp           = "app"
	AuditResourceChannelConfig = "channel_config"
)

// IsPaid 判断订单是否已支付
func (o *Order) IsPaid() bool {
	return o.Status == OrderStatusPaid && o.PaidAt != nil
}

// Validate 验证订单字段
func (o *Order) Validate() error {
	if o.Amount <= 0 {
		return fmt.Errorf("amount must be greater than 0")
	}
	if o.Subject == "" {
		return fmt.Errorf("subject is required")
	}
	if o.AppID == "" {
		return fmt.Errorf("app_id is required")
	}
	if o.OutTradeNo == "" {
		return fmt.Errorf("out_trade_no is required")
	}
	if o.Channel == "" {
		return fmt.Errorf("channel is required")
	}
	return nil
}

// ValidateApp 验证应用字段
func (a *App) Validate() error {
	if a.AppID == "" {
		return fmt.Errorf("app_id is required")
	}
	if a.AppName == "" {
		return fmt.Errorf("app_name is required")
	}
	if a.AppSecret == "" {
		return fmt.Errorf("app_secret is required")
	}
	if a.CallbackURL == "" {
		return fmt.Errorf("callback_url is required")
	}
	return nil
}

// ValidateChannelConfig 验证渠道配置字段
func (c *ChannelConfig) Validate() error {
	if c.AppID == "" {
		return fmt.Errorf("app_id is required")
	}
	if c.Channel == "" {
		return fmt.Errorf("channel is required")
	}
	if c.Config == "" {
		return fmt.Errorf("config is required")
	}
	return nil
}
