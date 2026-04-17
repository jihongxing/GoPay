package handler

import (
	"gopay/internal/service"
	"gopay/pkg/logger"
	"gopay/pkg/response"

	"github.com/gin-gonic/gin"
)

var (
	orderService  *service.OrderService
	refundService *service.RefundService
)

// InitServices 初始化服务
func InitServices(os *service.OrderService) {
	orderService = os
}

// InitRefundService 初始化退款服务
func InitRefundService(rs *service.RefundService) {
	refundService = rs
}

// CheckoutRequest 支付请求
type CheckoutRequest struct {
	AppID      string            `json:"app_id" binding:"required"`
	OutTradeNo string            `json:"out_trade_no" binding:"required"`
	Amount     int64             `json:"amount" binding:"required,gt=0"`
	Subject    string            `json:"subject" binding:"required"`
	Body       string            `json:"body"`
	Channel    string            `json:"channel" binding:"required"`
	NotifyURL  string            `json:"notify_url"`
	ExtraData  map[string]string `json:"extra_data"`
}

// CheckoutResponse 支付响应
type CheckoutResponse struct {
	OrderNo  string            `json:"order_no"`
	PayURL   string            `json:"pay_url,omitempty"`   // Native、H5 使用
	QRCode   string            `json:"qr_code,omitempty"`   // Native 使用
	PrepayID string            `json:"prepay_id,omitempty"` // JSAPI、APP 使用
	PayInfo  map[string]string `json:"pay_info,omitempty"`  // JSAPI、APP 调起支付参数
}

// Checkout 创建支付订单
func Checkout(c *gin.Context) {
	var req CheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("Invalid checkout request: %v", err)
		response.BadRequest(c, "请求参数错误", err.Error())
		return
	}

	logger.Info("Checkout request: app_id=%s, out_trade_no=%s, amount=%d, channel=%s",
		req.AppID, req.OutTradeNo, req.Amount, req.Channel)

	// 转换为服务层请求
	serviceReq := &service.CreateOrderRequest{
		AppID:      req.AppID,
		OutTradeNo: req.OutTradeNo,
		Amount:     req.Amount,
		Subject:    req.Subject,
		Body:       req.Body,
		Channel:    req.Channel,
		NotifyURL:  req.NotifyURL,
		ExtraData:  req.ExtraData,
	}

	// 如果经过签名验证，确保请求体中的 app_id 与签名中的一致
	if verifiedAppID, exists := c.Get("verified_app_id"); exists {
		if req.AppID != verifiedAppID.(string) {
			response.BadRequest(c, "app_id 与签名不匹配")
			return
		}
	}

	// 调用服务层创建订单
	resp, err := orderService.CreateOrder(c.Request.Context(), serviceReq)
	if err != nil {
		logger.Error("Failed to create order: %v", err)
		handleServiceError(c, err)
		return
	}

	logger.Info("Order created: order_no=%s", resp.OrderNo)

	response.Success(c, "订单创建成功", CheckoutResponse{
		OrderNo:  resp.OrderNo,
		PayURL:   resp.PayURL,
		QRCode:   resp.QRCode,
		PrepayID: resp.PrepayID,
		PayInfo:  resp.ExtraData,
	})
}

// QueryOrder 查询订单
func QueryOrder(c *gin.Context) {
	orderNo := c.Param("order_no")
	if orderNo == "" {
		response.BadRequest(c, "订单号不能为空")
		return
	}

	logger.Info("Query order request: order_no=%s", orderNo)

	order, err := orderService.QueryOrder(c.Request.Context(), orderNo)
	if err != nil {
		logger.Error("Failed to query order: %v", err)
		handleServiceError(c, err)
		return
	}

	response.Success(c, "查询成功", order)
}

// handleServiceError 处理服务层错误（使用精确的错误类型映射）
func handleServiceError(c *gin.Context, err error) {
	// 使用新的精确错误处理函数
	response.HandleError(c, err)
}
