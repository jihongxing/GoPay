package client

import "time"

// CreateOrderRequest 创建订单请求
type CreateOrderRequest struct {
	AppID      string                 `json:"app_id"`
	OutTradeNo string                 `json:"out_trade_no"`
	Amount     int64                  `json:"amount"`
	Currency   string                 `json:"currency,omitempty"`
	Subject    string                 `json:"subject"`
	Body       string                 `json:"body,omitempty"`
	Channel    string                 `json:"channel"`
	NotifyURL  string                 `json:"notify_url"`
	ExtraData  map[string]interface{} `json:"extra_data,omitempty"`
}

// CreateOrderResponse 创建订单响应
type CreateOrderResponse struct {
	OrderNo   string                 `json:"order_no"`
	PayURL    string                 `json:"pay_url,omitempty"`
	QRCode    string                 `json:"qr_code,omitempty"`
	PrepayID  string                 `json:"prepay_id,omitempty"`
	PayInfo   map[string]interface{} `json:"pay_info,omitempty"`
}

// Order 订单信息
type Order struct {
	OrderNo     string     `json:"order_no"`
	AppID       string     `json:"app_id"`
	OutTradeNo  string     `json:"out_trade_no"`
	Amount      int64      `json:"amount"`
	Currency    string     `json:"currency"`
	Subject     string     `json:"subject"`
	Body        string     `json:"body"`
	Channel     string     `json:"channel"`
	Status      string     `json:"status"`
	PaidAt      *time.Time `json:"paid_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// CallbackData 回调数据
type CallbackData struct {
	OrderNo    string `json:"order_no"`
	OutTradeNo string `json:"out_trade_no"`
	Amount     int64  `json:"amount"`
	Currency   string `json:"currency"`
	Channel    string `json:"channel"`
	Status     string `json:"status"`
	PaidAt     string `json:"paid_at,omitempty"`
}
