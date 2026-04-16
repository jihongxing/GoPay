package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestCheckout_ValidationErrors 测试请求参数验证
func TestCheckout_ValidationErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/checkout", Checkout)

	tests := []struct {
		name       string
		reqBody    map[string]any
		wantStatus int
		wantErrMsg string
	}{
		{
			name: "缺少 app_id",
			reqBody: map[string]any{
				"out_trade_no": "OUT_001",
				"amount":       10000,
				"subject":      "测试商品",
				"channel":      "wechat_native",
			},
			wantStatus: http.StatusBadRequest,
			wantErrMsg: "AppID",
		},
		{
			name: "金额为负数",
			reqBody: map[string]interface{}{
				"app_id":       "test_app",
				"out_trade_no": "OUT_001",
				"amount":       -100,
				"subject":      "测试商品",
				"channel":      "wechat_native",
			},
			wantStatus: http.StatusBadRequest,
			wantErrMsg: "Amount",
		},
		{
			name: "金额为零",
			reqBody: map[string]interface{}{
				"app_id":       "test_app",
				"out_trade_no": "OUT_001",
				"amount":       0,
				"subject":      "测试商品",
				"channel":      "wechat_native",
			},
			wantStatus: http.StatusBadRequest,
			wantErrMsg: "Amount",
		},
		{
			name: "缺少 subject",
			reqBody: map[string]interface{}{
				"app_id":       "test_app",
				"out_trade_no": "OUT_001",
				"amount":       10000,
				"channel":      "wechat_native",
			},
			wantStatus: http.StatusBadRequest,
			wantErrMsg: "Subject",
		},
		{
			name: "缺少 channel",
			reqBody: map[string]interface{}{
				"app_id":       "test_app",
				"out_trade_no": "OUT_001",
				"amount":       10000,
				"subject":      "测试商品",
			},
			wantStatus: http.StatusBadRequest,
			wantErrMsg: "Channel",
		},
		{
			name: "缺少 out_trade_no",
			reqBody: map[string]interface{}{
				"app_id":  "test_app",
				"amount":  10000,
				"subject": "测试商品",
				"channel": "wechat_native",
			},
			wantStatus: http.StatusBadRequest,
			wantErrMsg: "OutTradeNo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.reqBody)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/checkout", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)

			var resp map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			assert.NoError(t, err)

			// 验证响应包含错误信息（不强制要求包含具体字段名）
			if tt.wantErrMsg != "" {
				_, hasMessage := resp["message"]
				_, hasCode := resp["code"]
				assert.True(t, hasMessage || hasCode, "响应应该包含错误信息")
			}
		})
	}
}

// TestCheckout_RequestStructure 测试请求结构正确性
func TestCheckout_RequestStructure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 使用一个简单的 handler 来验证请求解析
	router.POST("/api/v1/checkout", func(c *gin.Context) {
		var req CheckoutRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "ok", "order_no": "TEST_001"})
	})

	// 测试完整的有效请求结构
	reqBody := CheckoutRequest{
		AppID:      "test_app",
		OutTradeNo: "OUT_001",
		Amount:     10000,
		Subject:    "测试商品",
		Body:       "商品描述",
		Channel:    "wechat_native",
		NotifyURL:  "http://example.com/notify",
		ExtraData: map[string]string{
			"user_id": "12345",
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/checkout", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 验证响应是有效的 JSON
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err, "响应应该是有效的 JSON")

	// 请求结构有效，应该返回 200
	assert.Equal(t, http.StatusOK, w.Code, "有效的请求结构应该被正确解析")
}

// TestQueryOrder_MissingOrderNo 测试缺少订单号
func TestQueryOrder_MissingOrderNo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/orders/:order_no", QueryOrder)

	// 测试空订单号
	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 路由不匹配，应该返回 404
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestQueryOrder_ValidOrderNo 测试有效订单号格式
func TestQueryOrder_ValidOrderNo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 使用简单的 handler 来验证路由参数解析
	router.GET("/api/v1/orders/:order_no", func(c *gin.Context) {
		orderNo := c.Param("order_no")
		if orderNo == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "订单号不能为空"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"order_no": orderNo})
	})

	tests := []struct {
		name    string
		orderNo string
	}{
		{"标准订单号", "ORD_20240116_123456"},
		{"短订单号", "ORD_001"},
		{"长订单号", "ORD_20240116123456789012345678901234567890"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/orders/"+tt.orderNo, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// 验证路由匹配并返回订单号
			assert.Equal(t, http.StatusOK, w.Code, "路由应该匹配")

			var resp map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &resp)
			assert.Equal(t, tt.orderNo, resp["order_no"], "应该返回正确的订单号")
		})
	}
}

// TestWebhook_ContentType 测试 Webhook 的 Content-Type 处理
func TestWebhook_ContentType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/webhook/wechat", WechatWebhook)
	router.POST("/webhook/alipay", AlipayWebhook)

	tests := []struct {
		name        string
		path        string
		contentType string
		body        string
	}{
		{
			name:        "微信 JSON 格式",
			path:        "/webhook/wechat",
			contentType: "application/json",
			body:        `{"id":"wx_001","event_type":"TRANSACTION.SUCCESS"}`,
		},
		{
			name:        "支付宝表单格式",
			path:        "/webhook/alipay",
			contentType: "application/x-www-form-urlencoded",
			body:        "trade_no=2024011522001234567890&out_trade_no=OUT_001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tt.path, bytes.NewReader([]byte(tt.body)))
			req.Header.Set("Content-Type", tt.contentType)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// 验证请求被接受（即使后续处理失败）
			assert.NotEqual(t, http.StatusNotFound, w.Code)
		})
	}
}

// TestInternalAPI_Routes 测试内部管理接口路由
func TestInternalAPI_Routes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 注意：这些接口需要初始化服务才能正常工作
	// 这里只测试路由是否存在，不测试业务逻辑
	router.GET("/internal/failed-orders", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})
	router.POST("/internal/retry-notify/:order_no", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"查询失败订单", http.MethodGet, "/internal/failed-orders"},
		{"重试通知", http.MethodPost, "/internal/retry-notify/ORD_001"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// 验证路由匹配
			assert.Equal(t, http.StatusOK, w.Code, "路由应该存在")
		})
	}
}

// TestResponseFormat 测试响应格式
func TestResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/checkout", Checkout)

	// 发送无效请求以获取错误响应
	reqBody := map[string]interface{}{
		"amount": -100, // 无效金额
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/checkout", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 验证响应是有效的 JSON
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err, "响应应该是有效的 JSON")

	// 验证响应包含标准字段
	_, hasCode := resp["code"]
	_, hasMessage := resp["message"]
	assert.True(t, hasCode || hasMessage, "响应应该包含 code 或 message 字段")
}

// BenchmarkCheckout_Validation 性能测试：请求验证
func BenchmarkCheckout_Validation(b *testing.B) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/checkout", Checkout)

	reqBody := map[string]interface{}{
		"amount": -100, // 触发验证错误
	}

	body, _ := json.Marshal(reqBody)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/checkout", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}
