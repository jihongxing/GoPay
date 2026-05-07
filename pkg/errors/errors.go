package errors

import (
	"errors"
	"fmt"
)

// ErrorType 错误类型枚举
type ErrorType int

const (
	// 通用查询错误
	TypeNotFound ErrorType = iota + 1

	// 应用相关错误
	TypeAppNotFound ErrorType = iota + 1
	TypeAppInactive

	// 渠道相关错误
	TypeChannelNotFound
	TypeChannelInactive

	// 订单相关错误
	TypeOrderNotFound
	TypeOrderExists
	TypeOrderPaid
	TypeOrderClosed

	// 金额相关错误
	TypeInvalidAmount

	// 支付相关错误
	TypePaymentFailed
	TypeNotifyFailed
	TypeSignatureInvalid

	// 参数相关错误
	TypeInvalidRequest
	TypeInvalidChannel

	// 认证相关错误
	TypeTimestampExpired
	TypeNonceReplay
)

// 定义业务错误类型（用于 errors.Is 判断）
var (
	// 通用查询错误
	ErrNotFound = errors.New("not found")

	// 应用相关错误
	ErrAppNotFound = errors.New("app not found")
	ErrAppInactive = errors.New("app inactive")

	// 渠道相关错误
	ErrChannelNotFound = errors.New("channel not found")
	ErrChannelInactive = errors.New("channel inactive")

	// 订单相关错误
	ErrOrderNotFound = errors.New("order not found")
	ErrOrderExists   = errors.New("order exists")
	ErrOrderPaid     = errors.New("order paid")
	ErrOrderClosed   = errors.New("order closed")

	// 金额相关错误
	ErrInvalidAmount = errors.New("invalid amount")

	// 支付相关错误
	ErrPaymentFailed    = errors.New("payment failed")
	ErrNotifyFailed     = errors.New("notify failed")
	ErrSignatureInvalid = errors.New("signature invalid")

	// 参数相关错误
	ErrInvalidRequest = errors.New("invalid request")
	ErrInvalidChannel = errors.New("invalid channel")
)

// BusinessError 业务错误
type BusinessError struct {
	Type    ErrorType         // 错误类型（用于精确匹配）
	Code    string            // 错误码
	Message string            // 错误消息
	Details map[string]string // 详细信息
	Err     error             // 原始错误
}

func (e *BusinessError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *BusinessError) Unwrap() error {
	return e.Err
}

// GetType 获取错误类型
func (e *BusinessError) GetType() ErrorType {
	return e.Type
}

// GetDetails 获取详细信息
func (e *BusinessError) GetDetails() map[string]string {
	return e.Details
}

// NewBusinessError 创建业务错误
func NewBusinessError(errType ErrorType, code, message string, err error, details map[string]string) *BusinessError {
	return &BusinessError{
		Type:    errType,
		Code:    code,
		Message: message,
		Details: details,
		Err:     err,
	}
}

// 便捷方法 - 应用相关

func NewAppNotFoundError(appID string) error {
	return &BusinessError{
		Type:    TypeAppNotFound,
		Code:    "APP_NOT_FOUND",
		Message: "应用不存在",
		Details: map[string]string{"app_id": appID},
		Err:     ErrAppNotFound,
	}
}

func NewAppInactiveError(appID string) error {
	return &BusinessError{
		Type:    TypeAppInactive,
		Code:    "APP_INACTIVE",
		Message: "应用未激活",
		Details: map[string]string{"app_id": appID},
		Err:     ErrAppInactive,
	}
}

// 便捷方法 - 渠道相关

func NewChannelNotFoundError(appID, channel string) error {
	return &BusinessError{
		Type:    TypeChannelNotFound,
		Code:    "CHANNEL_NOT_FOUND",
		Message: "支付渠道不存在",
		Details: map[string]string{"app_id": appID, "channel": channel},
		Err:     ErrChannelNotFound,
	}
}

func NewChannelInactiveError(appID, channel string) error {
	return &BusinessError{
		Type:    TypeChannelInactive,
		Code:    "CHANNEL_INACTIVE",
		Message: "支付渠道未激活",
		Details: map[string]string{"app_id": appID, "channel": channel},
		Err:     ErrChannelInactive,
	}
}

// 便捷方法 - 订单相关

func NewOrderNotFoundError(orderNo string) error {
	return &BusinessError{
		Type:    TypeOrderNotFound,
		Code:    "ORDER_NOT_FOUND",
		Message: "订单不存在",
		Details: map[string]string{"order_no": orderNo},
		Err:     ErrOrderNotFound,
	}
}

func NewOrderExistsError(outTradeNo string) error {
	return &BusinessError{
		Type:    TypeOrderExists,
		Code:    "ORDER_EXISTS",
		Message: "订单已存在",
		Details: map[string]string{"out_trade_no": outTradeNo},
		Err:     ErrOrderExists,
	}
}

func NewOrderPaidError(orderNo string) error {
	return &BusinessError{
		Type:    TypeOrderPaid,
		Code:    "ORDER_PAID",
		Message: "订单已支付",
		Details: map[string]string{"order_no": orderNo},
		Err:     ErrOrderPaid,
	}
}

func NewOrderClosedError(orderNo string) error {
	return &BusinessError{
		Type:    TypeOrderClosed,
		Code:    "ORDER_CLOSED",
		Message: "订单已关闭",
		Details: map[string]string{"order_no": orderNo},
		Err:     ErrOrderClosed,
	}
}

// 便捷方法 - 金额相关

func NewInvalidAmountError(amount int64) error {
	return &BusinessError{
		Type:    TypeInvalidAmount,
		Code:    "INVALID_AMOUNT",
		Message: "金额无效",
		Details: map[string]string{"amount": fmt.Sprintf("%d", amount), "requirement": "必须大于 0"},
		Err:     ErrInvalidAmount,
	}
}

// 便捷方法 - 支付相关

func NewPaymentFailedError(message string, err error) error {
	details := make(map[string]string)
	if err != nil {
		details["error"] = err.Error()
	}
	return &BusinessError{
		Type:    TypePaymentFailed,
		Code:    "PAYMENT_FAILED",
		Message: message,
		Details: details,
		Err:     err,
	}
}

func NewNotifyFailedError(orderNo string, err error) error {
	details := map[string]string{"order_no": orderNo}
	if err != nil {
		details["error"] = err.Error()
	}
	return &BusinessError{
		Type:    TypeNotifyFailed,
		Code:    "NOTIFY_FAILED",
		Message: "通知失败",
		Details: details,
		Err:     err,
	}
}

func NewSignatureInvalidError() error {
	return &BusinessError{
		Type:    TypeSignatureInvalid,
		Code:    "SIGNATURE_INVALID",
		Message: "签名验证失败",
		Details: map[string]string{"hint": "请检查签名算法和密钥"},
		Err:     ErrSignatureInvalid,
	}
}

// 便捷方法 - 参数相关

func NewInvalidChannelError(channel string) error {
	return &BusinessError{
		Type:    TypeInvalidChannel,
		Code:    "INVALID_CHANNEL",
		Message: "支付渠道无效",
		Details: map[string]string{"channel": channel},
		Err:     ErrInvalidChannel,
	}
}

func NewInvalidRequestError(message string, details map[string]string) error {
	return &BusinessError{
		Type:    TypeInvalidRequest,
		Code:    "INVALID_REQUEST",
		Message: message,
		Details: details,
		Err:     ErrInvalidRequest,
	}
}
