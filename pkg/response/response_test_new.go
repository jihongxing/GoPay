package response

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	pkgerrors "gopay/pkg/errors"
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

// TestFormatDetails 测试格式化详细信息
func TestFormatDetails(t *testing.T) {
	tests := []struct {
		name    string
		details map[string]string
		want    string
	}{
		{
			name:    "empty details",
			details: map[string]string{},
			want:    "",
		},
		{
			name: "single detail",
			details: map[string]string{
				"key": "value",
			},
			want: "key: value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDetails(tt.details)
			if tt.name == "empty details" && got != tt.want {
				t.Errorf("formatDetails() = %v, want %v", got, tt.want)
			}
			if tt.name == "single detail" && got != tt.want {
				t.Errorf("formatDetails() = %v, want %v", got, tt.want)
			}
		})
	}
}
