package response

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	pkgerrors "gopay/pkg/errors"
)

// ErrorCode 错误码
type ErrorCode string

const (
	// 通用错误码
	ErrInvalidRequest   ErrorCode = "INVALID_REQUEST"
	ErrInternalError    ErrorCode = "INTERNAL_ERROR"
	ErrUnauthorized     ErrorCode = "UNAUTHORIZED"
	ErrForbidden        ErrorCode = "FORBIDDEN"
	ErrNotFound         ErrorCode = "NOT_FOUND"
	ErrConflict         ErrorCode = "CONFLICT"
	ErrTooManyRequests  ErrorCode = "TOO_MANY_REQUESTS"

	// 业务错误码
	ErrAppNotFound      ErrorCode = "APP_NOT_FOUND"
	ErrAppInactive      ErrorCode = "APP_INACTIVE"
	ErrChannelNotFound  ErrorCode = "CHANNEL_NOT_FOUND"
	ErrChannelInactive  ErrorCode = "CHANNEL_INACTIVE"
	ErrOrderNotFound    ErrorCode = "ORDER_NOT_FOUND"
	ErrOrderExists      ErrorCode = "ORDER_EXISTS"
	ErrOrderPaid        ErrorCode = "ORDER_PAID"
	ErrOrderClosed      ErrorCode = "ORDER_CLOSED"
	ErrInvalidAmount    ErrorCode = "INVALID_AMOUNT"
	ErrInvalidChannel   ErrorCode = "INVALID_CHANNEL"
	ErrPaymentFailed    ErrorCode = "PAYMENT_FAILED"
	ErrNotifyFailed     ErrorCode = "NOTIFY_FAILED"
	ErrSignatureInvalid ErrorCode = "SIGNATURE_INVALID"
)

// ErrorResponse 统一错误响应
type ErrorResponse struct {
	Code    ErrorCode `json:"code"`              // 错误码
	Message string    `json:"message"`           // 错误消息
	Details string    `json:"details,omitempty"` // 详细信息（可选）
}

// SuccessResponse 统一成功响应
type SuccessResponse struct {
	Code    string `json:"code"`    // 成功码，固定为 "SUCCESS"
	Message string `json:"message"` // 成功消息
	Data    any    `json:"data"`    // 响应数据
}

// Error 返回错误响应
func Error(c *gin.Context, httpStatus int, code ErrorCode, message string, details ...string) {
	resp := ErrorResponse{
		Code:    code,
		Message: message,
	}
	if len(details) > 0 {
		resp.Details = details[0]
	}
	c.JSON(httpStatus, resp)
}

// Success 返回成功响应
func Success(c *gin.Context, message string, data any) {
	c.JSON(http.StatusOK, SuccessResponse{
		Code:    "SUCCESS",
		Message: message,
		Data:    data,
	})
}

// HandleError 精确处理业务错误（推荐使用）
func HandleError(c *gin.Context, err error) {
	// 尝试断言为 BusinessError
	var bizErr *pkgerrors.BusinessError
	if errors.As(err, &bizErr) {
		handleBusinessError(c, bizErr)
		return
	}

	// 使用 errors.Is 判断标准错误类型
	switch {
	case errors.Is(err, pkgerrors.ErrAppNotFound):
		Error(c, http.StatusNotFound, ErrAppNotFound, "应用不存在", err.Error())
	case errors.Is(err, pkgerrors.ErrAppInactive):
		Error(c, http.StatusForbidden, ErrAppInactive, "应用未激活", err.Error())
	case errors.Is(err, pkgerrors.ErrChannelNotFound):
		Error(c, http.StatusNotFound, ErrChannelNotFound, "支付渠道不存在", err.Error())
	case errors.Is(err, pkgerrors.ErrChannelInactive):
		Error(c, http.StatusForbidden, ErrChannelInactive, "支付渠道未激活", err.Error())
	case errors.Is(err, pkgerrors.ErrOrderNotFound):
		Error(c, http.StatusNotFound, ErrOrderNotFound, "订单不存在", err.Error())
	case errors.Is(err, pkgerrors.ErrOrderExists):
		Error(c, http.StatusConflict, ErrOrderExists, "订单已存在", err.Error())
	case errors.Is(err, pkgerrors.ErrOrderPaid):
		Error(c, http.StatusConflict, ErrOrderPaid, "订单已支付", err.Error())
	case errors.Is(err, pkgerrors.ErrOrderClosed):
		Error(c, http.StatusConflict, ErrOrderClosed, "订单已关闭", err.Error())
	case errors.Is(err, pkgerrors.ErrInvalidAmount):
		Error(c, http.StatusBadRequest, ErrInvalidAmount, "金额无效", err.Error())
	case errors.Is(err, pkgerrors.ErrInvalidChannel):
		Error(c, http.StatusBadRequest, ErrInvalidChannel, "支付渠道无效", err.Error())
	case errors.Is(err, pkgerrors.ErrPaymentFailed):
		Error(c, http.StatusInternalServerError, ErrPaymentFailed, "支付失败", err.Error())
	case errors.Is(err, pkgerrors.ErrNotifyFailed):
		Error(c, http.StatusInternalServerError, ErrNotifyFailed, "通知失败", err.Error())
	case errors.Is(err, pkgerrors.ErrSignatureInvalid):
		Error(c, http.StatusUnauthorized, ErrSignatureInvalid, "签名验证失败", err.Error())
	case errors.Is(err, pkgerrors.ErrInvalidRequest):
		Error(c, http.StatusBadRequest, ErrInvalidRequest, "请求参数错误", err.Error())
	default:
		// 未知错误，返回内部错误
		Error(c, http.StatusInternalServerError, ErrInternalError, "服务器内部错误", err.Error())
	}
}

// handleBusinessError 处理 BusinessError（包含详细信息）
func handleBusinessError(c *gin.Context, bizErr *pkgerrors.BusinessError) {
	var httpStatus int
	var code ErrorCode
	message := bizErr.Message

	// 根据错误类型精确映射
	switch bizErr.GetType() {
	case pkgerrors.TypeAppNotFound:
		httpStatus = http.StatusNotFound
		code = ErrAppNotFound
	case pkgerrors.TypeAppInactive:
		httpStatus = http.StatusForbidden
		code = ErrAppInactive
	case pkgerrors.TypeChannelNotFound:
		httpStatus = http.StatusNotFound
		code = ErrChannelNotFound
	case pkgerrors.TypeChannelInactive:
		httpStatus = http.StatusForbidden
		code = ErrChannelInactive
	case pkgerrors.TypeOrderNotFound:
		httpStatus = http.StatusNotFound
		code = ErrOrderNotFound
	case pkgerrors.TypeOrderExists:
		httpStatus = http.StatusConflict
		code = ErrOrderExists
	case pkgerrors.TypeOrderPaid:
		httpStatus = http.StatusConflict
		code = ErrOrderPaid
	case pkgerrors.TypeOrderClosed:
		httpStatus = http.StatusConflict
		code = ErrOrderClosed
	case pkgerrors.TypeInvalidAmount:
		httpStatus = http.StatusBadRequest
		code = ErrInvalidAmount
	case pkgerrors.TypeInvalidChannel:
		httpStatus = http.StatusBadRequest
		code = ErrInvalidChannel
	case pkgerrors.TypePaymentFailed:
		httpStatus = http.StatusInternalServerError
		code = ErrPaymentFailed
	case pkgerrors.TypeNotifyFailed:
		httpStatus = http.StatusInternalServerError
		code = ErrNotifyFailed
	case pkgerrors.TypeSignatureInvalid:
		httpStatus = http.StatusUnauthorized
		code = ErrSignatureInvalid
	case pkgerrors.TypeInvalidRequest:
		httpStatus = http.StatusBadRequest
		code = ErrInvalidRequest
	default:
		httpStatus = http.StatusInternalServerError
		code = ErrInternalError
	}

	// 格式化详细信息
	details := formatDetails(bizErr.GetDetails())
	Error(c, httpStatus, code, message, details)
}

// formatDetails 格式化详细信息
func formatDetails(details map[string]string) string {
	if len(details) == 0 {
		return ""
	}

	result := ""
	for k, v := range details {
		if result != "" {
			result += ", "
		}
		result += fmt.Sprintf("%s: %s", k, v)
	}
	return result
}

// 便捷方法

// BadRequest 400 错误请求
func BadRequest(c *gin.Context, message string, details ...string) {
	Error(c, http.StatusBadRequest, ErrInvalidRequest, message, details...)
}

// Unauthorized 401 未授权
func Unauthorized(c *gin.Context, message string, details ...string) {
	Error(c, http.StatusUnauthorized, ErrUnauthorized, message, details...)
}

// Forbidden 403 禁止访问
func Forbidden(c *gin.Context, message string, details ...string) {
	Error(c, http.StatusForbidden, ErrForbidden, message, details...)
}

// NotFound 404 未找到
func NotFound(c *gin.Context, message string, details ...string) {
	Error(c, http.StatusNotFound, ErrNotFound, message, details...)
}

// Conflict 409 冲突
func Conflict(c *gin.Context, message string, details ...string) {
	Error(c, http.StatusConflict, ErrConflict, message, details...)
}

// InternalError 500 内部错误
func InternalError(c *gin.Context, message string, details ...string) {
	Error(c, http.StatusInternalServerError, ErrInternalError, message, details...)
}

// TooManyRequests 429 请求过多
func TooManyRequests(c *gin.Context, message string, details ...string) {
	Error(c, http.StatusTooManyRequests, ErrTooManyRequests, message, details...)
}

// 业务错误便捷方法

// AppNotFound 应用不存在
func AppNotFound(c *gin.Context, appID string) {
	Error(c, http.StatusNotFound, ErrAppNotFound, "应用不存在", "app_id: "+appID)
}

// AppInactive 应用未激活
func AppInactive(c *gin.Context, appID string) {
	Error(c, http.StatusForbidden, ErrAppInactive, "应用未激活", "app_id: "+appID)
}

// ChannelNotFound 渠道不存在
func ChannelNotFound(c *gin.Context, channel string) {
	Error(c, http.StatusNotFound, ErrChannelNotFound, "支付渠道不存在", "channel: "+channel)
}

// ChannelInactive 渠道未激活
func ChannelInactive(c *gin.Context, channel string) {
	Error(c, http.StatusForbidden, ErrChannelInactive, "支付渠道未激活", "channel: "+channel)
}

// OrderNotFound 订单不存在
func OrderNotFound(c *gin.Context, orderNo string) {
	Error(c, http.StatusNotFound, ErrOrderNotFound, "订单不存在", "order_no: "+orderNo)
}

// OrderExists 订单已存在
func OrderExists(c *gin.Context, outTradeNo string) {
	Error(c, http.StatusConflict, ErrOrderExists, "订单已存在", "out_trade_no: "+outTradeNo)
}

// OrderPaid 订单已支付
func OrderPaid(c *gin.Context, orderNo string) {
	Error(c, http.StatusConflict, ErrOrderPaid, "订单已支付", "order_no: "+orderNo)
}

// OrderClosed 订单已关闭
func OrderClosed(c *gin.Context, orderNo string) {
	Error(c, http.StatusConflict, ErrOrderClosed, "订单已关闭", "order_no: "+orderNo)
}

// InvalidAmount 金额无效
func InvalidAmount(c *gin.Context, amount int64) {
	Error(c, http.StatusBadRequest, ErrInvalidAmount, "金额无效", "amount 必须大于 0")
}

// InvalidChannel 渠道无效
func InvalidChannel(c *gin.Context, channel string) {
	Error(c, http.StatusBadRequest, ErrInvalidChannel, "支付渠道无效", "channel: "+channel)
}

// PaymentFailed 支付失败
func PaymentFailed(c *gin.Context, message string, details ...string) {
	Error(c, http.StatusInternalServerError, ErrPaymentFailed, message, details...)
}

// NotifyFailed 通知失败
func NotifyFailed(c *gin.Context, message string, details ...string) {
	Error(c, http.StatusInternalServerError, ErrNotifyFailed, message, details...)
}

// SignatureInvalid 签名无效
func SignatureInvalid(c *gin.Context) {
	Error(c, http.StatusUnauthorized, ErrSignatureInvalid, "签名验证失败", "请检查签名算法和密钥")
}
