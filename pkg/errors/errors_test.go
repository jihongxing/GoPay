package errors

import (
	"errors"
	"testing"
)

// TestBusinessError_Error 测试错误消息
func TestBusinessError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *BusinessError
		want string
	}{
		{
			name: "without wrapped error",
			err: &BusinessError{
				Type:    TypeAppNotFound,
				Code:    "APP_NOT_FOUND",
				Message: "应用不存在",
			},
			want: "应用不存在",
		},
		{
			name: "with wrapped error",
			err: &BusinessError{
				Type:    TypeAppNotFound,
				Code:    "APP_NOT_FOUND",
				Message: "应用不存在",
				Err:     errors.New("database error"),
			},
			want: "应用不存在: database error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.want {
				t.Errorf("Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestBusinessError_Unwrap 测试错误解包
func TestBusinessError_Unwrap(t *testing.T) {
	innerErr := errors.New("inner error")
	bizErr := &BusinessError{
		Type:    TypeAppNotFound,
		Code:    "APP_NOT_FOUND",
		Message: "应用不存在",
		Err:     innerErr,
	}

	unwrapped := bizErr.Unwrap()
	if unwrapped != innerErr {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, innerErr)
	}
}

// TestBusinessError_Is 测试错误判断
func TestBusinessError_Is(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		target error
		want   bool
	}{
		{
			name:   "app not found",
			err:    NewAppNotFoundError("test_app"),
			target: ErrAppNotFound,
			want:   true,
		},
		{
			name:   "order not found",
			err:    NewOrderNotFoundError("ORD_001"),
			target: ErrOrderNotFound,
			want:   true,
		},
		{
			name:   "different error type",
			err:    NewAppNotFoundError("test_app"),
			target: ErrOrderNotFound,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := errors.Is(tt.err, tt.target)
			if got != tt.want {
				t.Errorf("errors.Is() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestNewAppNotFoundError 测试创建应用不存在错误
func TestNewAppNotFoundError(t *testing.T) {
	appID := "test_app_001"
	err := NewAppNotFoundError(appID)

	bizErr, ok := err.(*BusinessError)
	if !ok {
		t.Fatal("Expected *BusinessError")
	}

	if bizErr.Type != TypeAppNotFound {
		t.Errorf("Type = %v, want %v", bizErr.Type, TypeAppNotFound)
	}
	if bizErr.Code != "APP_NOT_FOUND" {
		t.Errorf("Code = %v, want APP_NOT_FOUND", bizErr.Code)
	}
	if bizErr.Details["app_id"] != appID {
		t.Errorf("Details[app_id] = %v, want %v", bizErr.Details["app_id"], appID)
	}
}

// TestNewOrderExistsError 测试创建订单已存在错误
func TestNewOrderExistsError(t *testing.T) {
	outTradeNo := "TEST_ORDER_001"
	err := NewOrderExistsError(outTradeNo)

	bizErr, ok := err.(*BusinessError)
	if !ok {
		t.Fatal("Expected *BusinessError")
	}

	if bizErr.Type != TypeOrderExists {
		t.Errorf("Type = %v, want %v", bizErr.Type, TypeOrderExists)
	}
	if bizErr.Code != "ORDER_EXISTS" {
		t.Errorf("Code = %v, want ORDER_EXISTS", bizErr.Code)
	}
	if bizErr.Details["out_trade_no"] != outTradeNo {
		t.Errorf("Details[out_trade_no] = %v, want %v", bizErr.Details["out_trade_no"], outTradeNo)
	}
}

// TestNewInvalidAmountError 测试创建金额无效错误
func TestNewInvalidAmountError(t *testing.T) {
	amount := int64(-100)
	err := NewInvalidAmountError(amount)

	bizErr, ok := err.(*BusinessError)
	if !ok {
		t.Fatal("Expected *BusinessError")
	}

	if bizErr.Type != TypeInvalidAmount {
		t.Errorf("Type = %v, want %v", bizErr.Type, TypeInvalidAmount)
	}
	if bizErr.Code != "INVALID_AMOUNT" {
		t.Errorf("Code = %v, want INVALID_AMOUNT", bizErr.Code)
	}
	if bizErr.Details["amount"] != "-100" {
		t.Errorf("Details[amount] = %v, want -100", bizErr.Details["amount"])
	}
}

// TestBusinessError_GetType 测试获取错误类型
func TestBusinessError_GetType(t *testing.T) {
	err := &BusinessError{
		Type:    TypeOrderPaid,
		Code:    "ORDER_PAID",
		Message: "订单已支付",
	}

	if err.GetType() != TypeOrderPaid {
		t.Errorf("GetType() = %v, want %v", err.GetType(), TypeOrderPaid)
	}
}

// TestBusinessError_GetDetails 测试获取详细信息
func TestBusinessError_GetDetails(t *testing.T) {
	details := map[string]string{
		"order_no": "ORD_001",
		"status":   "paid",
	}

	err := &BusinessError{
		Type:    TypeOrderPaid,
		Code:    "ORDER_PAID",
		Message: "订单已支付",
		Details: details,
	}

	got := err.GetDetails()
	if len(got) != len(details) {
		t.Errorf("GetDetails() length = %v, want %v", len(got), len(details))
	}
	if got["order_no"] != details["order_no"] {
		t.Errorf("GetDetails()[order_no] = %v, want %v", got["order_no"], details["order_no"])
	}
}
