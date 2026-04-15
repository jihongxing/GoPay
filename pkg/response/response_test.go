package response

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gopay/pkg/errors"
)

func TestHandleError_BusinessError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedCode   ErrorCode
	}{
		{
			name:           "AppNotFound",
			err:            errors.NewAppNotFoundError("test_app"),
			expectedStatus: http.StatusNotFound,
			expectedCode:   ErrAppNotFound,
		},
		{
			name:           "AppInactive",
			err:            errors.NewAppInactiveError("test_app"),
			expectedStatus: http.StatusForbidden,
			expectedCode:   ErrAppInactive,
		},
		{
			name:           "ChannelNotFound",
			err:            errors.NewChannelNotFoundError("test_app", "wechat"),
			expectedStatus: http.StatusNotFound,
			expectedCode:   ErrChannelNotFound,
		},
		{
			name:           "ChannelInactive",
			err:            errors.NewChannelInactiveError("test_app", "wechat"),
			expectedStatus: http.StatusForbidden,
			expectedCode:   ErrChannelInactive,
		},
		{
			name:           "OrderNotFound",
			err:            errors.NewOrderNotFoundError("ORD123"),
			expectedStatus: http.StatusNotFound,
			expectedCode:   ErrOrderNotFound,
		},
		{
			name:           "OrderExists",
			err:            errors.NewOrderExistsError("OUT123"),
			expectedStatus: http.StatusConflict,
			expectedCode:   ErrOrderExists,
		},
		{
			name:           "OrderPaid",
			err:            errors.NewOrderPaidError("ORD123"),
			expectedStatus: http.StatusConflict,
			expectedCode:   ErrOrderPaid,
		},
		{
			name:           "OrderClosed",
			err:            errors.NewOrderClosedError("ORD123"),
			expectedStatus: http.StatusConflict,
			expectedCode:   ErrOrderClosed,
		},
		{
			name:           "InvalidAmount",
			err:            errors.NewInvalidAmountError(-100),
			expectedStatus: http.StatusBadRequest,
			expectedCode:   ErrInvalidAmount,
		},
		{
			name:           "InvalidChannel",
			err:            errors.NewInvalidChannelError("unknown"),
			expectedStatus: http.StatusBadRequest,
			expectedCode:   ErrInvalidChannel,
		},
		{
			name:           "PaymentFailed",
			err:            errors.NewPaymentFailedError("支付失败", nil),
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   ErrPaymentFailed,
		},
		{
			name:           "SignatureInvalid",
			err:            errors.NewSignatureInvalidError(),
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   ErrSignatureInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			HandleError(c, tt.err)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// 可以进一步解析 JSON 验证 code 字段
		})
	}
}

func TestHandleError_StandardError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedCode   ErrorCode
	}{
		{
			name:           "AppNotFound_StandardError",
			err:            errors.ErrAppNotFound,
			expectedStatus: http.StatusNotFound,
			expectedCode:   ErrAppNotFound,
		},
		{
			name:           "OrderExists_StandardError",
			err:            errors.ErrOrderExists,
			expectedStatus: http.StatusConflict,
			expectedCode:   ErrOrderExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			HandleError(c, tt.err)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}
