package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAllErrorConstructors(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		errType ErrorType
		target  error
	}{
		{"AppNotFound", NewAppNotFoundError("app1"), TypeAppNotFound, ErrAppNotFound},
		{"AppInactive", NewAppInactiveError("app1"), TypeAppInactive, ErrAppInactive},
		{"ChannelNotFound", NewChannelNotFoundError("app1", "wechat"), TypeChannelNotFound, ErrChannelNotFound},
		{"ChannelInactive", NewChannelInactiveError("app1", "wechat"), TypeChannelInactive, ErrChannelInactive},
		{"OrderNotFound", NewOrderNotFoundError("ORD_001"), TypeOrderNotFound, ErrOrderNotFound},
		{"OrderExists", NewOrderExistsError("BIZ_001"), TypeOrderExists, ErrOrderExists},
		{"OrderPaid", NewOrderPaidError("ORD_001"), TypeOrderPaid, ErrOrderPaid},
		{"OrderClosed", NewOrderClosedError("ORD_001"), TypeOrderClosed, ErrOrderClosed},
		{"InvalidAmount", NewInvalidAmountError(-100), TypeInvalidAmount, ErrInvalidAmount},
		{"InvalidChannel", NewInvalidChannelError("unknown"), TypeInvalidChannel, ErrInvalidChannel},
		{"SignatureInvalid", NewSignatureInvalidError(), TypeSignatureInvalid, ErrSignatureInvalid},
		{"InvalidRequest", NewInvalidRequestError("bad param", nil), TypeInvalidRequest, ErrInvalidRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Error(t, tt.err)
			assert.True(t, errors.Is(tt.err, tt.target))

			var bizErr *BusinessError
			assert.True(t, errors.As(tt.err, &bizErr))
			assert.Equal(t, tt.errType, bizErr.GetType())
			assert.NotEmpty(t, bizErr.Error())
			assert.NotEmpty(t, bizErr.Code)
		})
	}
}

func TestPaymentFailedError(t *testing.T) {
	inner := errors.New("timeout")
	err := NewPaymentFailedError("支付超时", inner)

	assert.True(t, errors.Is(err, inner))

	var bizErr *BusinessError
	assert.True(t, errors.As(err, &bizErr))
	assert.Equal(t, TypePaymentFailed, bizErr.GetType())
	assert.Contains(t, bizErr.Error(), "支付超时")
	assert.Contains(t, bizErr.Error(), "timeout")
	assert.Equal(t, "timeout", bizErr.GetDetails()["error"])
}

func TestPaymentFailedError_NilInner(t *testing.T) {
	err := NewPaymentFailedError("支付失败", nil)
	var bizErr *BusinessError
	assert.True(t, errors.As(err, &bizErr))
	assert.Equal(t, "支付失败", bizErr.Error())
}

func TestNotifyFailedError(t *testing.T) {
	inner := errors.New("connection refused")
	err := NewNotifyFailedError("ORD_001", inner)

	var bizErr *BusinessError
	assert.True(t, errors.As(err, &bizErr))
	assert.Equal(t, TypeNotifyFailed, bizErr.GetType())
	assert.Equal(t, "ORD_001", bizErr.GetDetails()["order_no"])
}

func TestNewBusinessError(t *testing.T) {
	details := map[string]string{"key": "value"}
	err := NewBusinessError(TypeInvalidRequest, "TEST_CODE", "test message", nil, details)

	assert.Equal(t, TypeInvalidRequest, err.GetType())
	assert.Equal(t, "TEST_CODE", err.Code)
	assert.Equal(t, "test message", err.Message)
	assert.Equal(t, "test message", err.Error())
	assert.Nil(t, err.Unwrap())
	assert.Equal(t, details, err.GetDetails())
}

func TestBusinessError_Unwrap_WithInner(t *testing.T) {
	inner := errors.New("root cause")
	err := NewBusinessError(TypeInvalidRequest, "CODE", "msg", inner, nil)

	assert.Equal(t, inner, err.Unwrap())
	assert.True(t, errors.Is(err, inner))
}

func TestInvalidRequestError_WithDetails(t *testing.T) {
	details := map[string]string{"field": "amount", "reason": "must be positive"}
	err := NewInvalidRequestError("参数错误", details)

	var bizErr *BusinessError
	assert.True(t, errors.As(err, &bizErr))
	assert.Equal(t, "amount", bizErr.GetDetails()["field"])
	assert.Equal(t, "must be positive", bizErr.GetDetails()["reason"])
}
