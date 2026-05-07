package response

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	pkgerrors "gopay/pkg/errors"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// TestSuccess 测试成功响应
func TestSuccess(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	data := map[string]string{"key": "value"}
	Success(c, "操作成功", data)

	if w.Code != http.StatusOK {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusOK)
	}

	// 检查响应体包含预期内容
	body := w.Body.String()
	if body == "" {
		t.Error("Response body is empty")
	}
}

// TestBadRequest 测试 400 错误
func TestBadRequest(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	BadRequest(c, "请求参数错误")

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// TestUnauthorized 测试 401 错误
func TestUnauthorized(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Unauthorized(c, "未授权")

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

// TestForbidden 测试 403 错误
func TestForbidden(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Forbidden(c, "禁止访问")

	if w.Code != http.StatusForbidden {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusForbidden)
	}
}

// TestNotFound 测试 404 错误
func TestNotFound(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	NotFound(c, "资源不存在")

	if w.Code != http.StatusNotFound {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// TestInternalError 测试 500 错误
func TestInternalError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	InternalError(c, "服务器内部错误")

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// TestHandleError_BusinessError 测试处理业务错误
func TestHandleError_BusinessError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		wantStatusCode int
	}{
		{
			name:           "app not found",
			err:            pkgerrors.NewAppNotFoundError("test_app"),
			wantStatusCode: http.StatusNotFound,
		},
		{
			name:           "order exists",
			err:            pkgerrors.NewOrderExistsError("TEST_ORDER_001"),
			wantStatusCode: http.StatusConflict,
		},
		{
			name:           "invalid amount",
			err:            pkgerrors.NewInvalidAmountError(-100),
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "order paid",
			err:            pkgerrors.NewOrderPaidError("ORD_001"),
			wantStatusCode: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			HandleError(c, tt.err)

			if w.Code != tt.wantStatusCode {
				t.Errorf("Status code = %d, want %d", w.Code, tt.wantStatusCode)
			}
		})
	}
}

// TestHandleError_StandardError 测试处理标准错误
func TestHandleError_StandardError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		wantStatusCode int
	}{
		{
			name:           "unknown error",
			err:            errors.New("unknown error"),
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			HandleError(c, tt.err)

			if w.Code != tt.wantStatusCode {
				t.Errorf("Status code = %d, want %d", w.Code, tt.wantStatusCode)
			}
		})
	}
}

// TestErrorWithDetails 测试带结构化详情的错误响应
func TestErrorWithDetails(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	details := map[string]string{"app_id": "test_app", "channel": "wechat_native"}
	ErrorWithDetails(c, http.StatusNotFound, ErrChannelNotFound, "支付渠道不存在", details)

	if w.Code != http.StatusNotFound {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusNotFound)
	}

	body := w.Body.String()
	if body == "" {
		t.Error("Response body is empty")
	}
}

// TestConflict 测试 409 冲突
func TestConflict(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Conflict(c, "资源冲突")

	if w.Code != http.StatusConflict {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusConflict)
	}
}

// TestTooManyRequests 测试 429 限流
func TestTooManyRequests(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	TooManyRequests(c, "请求过多")

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusTooManyRequests)
	}
}

// TestBusinessErrorConvenience 测试业务错误便捷方法
func TestBusinessErrorConvenience(t *testing.T) {
	tests := []struct {
		name       string
		fn         func(*gin.Context)
		wantStatus int
	}{
		{"AppNotFound", func(c *gin.Context) { AppNotFound(c, "app1") }, http.StatusNotFound},
		{"AppInactive", func(c *gin.Context) { AppInactive(c, "app1") }, http.StatusForbidden},
		{"ChannelNotFound", func(c *gin.Context) { ChannelNotFound(c, "wechat") }, http.StatusNotFound},
		{"ChannelInactive", func(c *gin.Context) { ChannelInactive(c, "wechat") }, http.StatusForbidden},
		{"OrderNotFound", func(c *gin.Context) { OrderNotFound(c, "ORD_001") }, http.StatusNotFound},
		{"OrderExists", func(c *gin.Context) { OrderExists(c, "BIZ_001") }, http.StatusConflict},
		{"OrderPaid", func(c *gin.Context) { OrderPaid(c, "ORD_001") }, http.StatusConflict},
		{"OrderClosed", func(c *gin.Context) { OrderClosed(c, "ORD_001") }, http.StatusConflict},
		{"InvalidAmount", func(c *gin.Context) { InvalidAmount(c, -100) }, http.StatusBadRequest},
		{"InvalidChannel", func(c *gin.Context) { InvalidChannel(c, "unknown") }, http.StatusBadRequest},
		{"PaymentFailed", func(c *gin.Context) { PaymentFailed(c, "支付失败") }, http.StatusInternalServerError},
		{"NotifyFailed", func(c *gin.Context) { NotifyFailed(c, "通知失败") }, http.StatusInternalServerError},
		{"SignatureInvalid", func(c *gin.Context) { SignatureInvalid(c) }, http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			tt.fn(c)
			if w.Code != tt.wantStatus {
				t.Errorf("Status code = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

// TestHandleError_AllBusinessErrorTypes 测试所有 errors.Is 分支
func TestHandleError_AllBusinessErrorTypes(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"AppInactive", pkgerrors.NewAppInactiveError("app1"), http.StatusForbidden},
		{"ChannelNotFound", pkgerrors.NewChannelNotFoundError("app1", "ch"), http.StatusNotFound},
		{"ChannelInactive", pkgerrors.NewChannelInactiveError("app1", "ch"), http.StatusForbidden},
		{"OrderClosed", pkgerrors.NewOrderClosedError("ORD_001"), http.StatusConflict},
		{"InvalidChannel", pkgerrors.NewInvalidChannelError("bad"), http.StatusBadRequest},
		{"PaymentFailed", pkgerrors.NewPaymentFailedError("fail", nil), http.StatusInternalServerError},
		{"NotifyFailed", pkgerrors.NewNotifyFailedError("ORD_001", nil), http.StatusInternalServerError},
		{"SignatureInvalid", pkgerrors.NewSignatureInvalidError(), http.StatusUnauthorized},
		{"InvalidRequest", pkgerrors.NewInvalidRequestError("bad", nil), http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			HandleError(c, tt.err)
			if w.Code != tt.wantStatus {
				t.Errorf("Status code = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

// TestError_WithDetails 测试 Error 函数带 details
func TestError_WithDetails(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Error(c, http.StatusBadRequest, ErrInvalidRequest, "参数错误", "amount 必须大于 0")

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// TestError_WithoutDetails 测试 Error 函数不带 details
func TestError_WithoutDetails(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Error(c, http.StatusBadRequest, ErrInvalidRequest, "参数错误")

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusBadRequest)
	}
}
