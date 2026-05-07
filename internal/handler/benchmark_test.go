package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func BenchmarkHealthCheck(b *testing.B) {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"code":    "SUCCESS",
			"message": "服务正常",
			"data": gin.H{
				"status":  "healthy",
				"service": "gopay",
			},
		})
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/health", nil)
			router.ServeHTTP(w, req)
		}
	})
}

func BenchmarkCheckoutValidation(b *testing.B) {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.POST("/api/v1/checkout", func(c *gin.Context) {
		var req struct {
			AppID      string `json:"app_id" binding:"required"`
			OutTradeNo string `json:"out_trade_no" binding:"required"`
			Amount     int64  `json:"amount" binding:"required,gt=0"`
			Subject    string `json:"subject" binding:"required"`
			Channel    string `json:"channel" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"code": "SUCCESS"})
	})

	body, _ := json.Marshal(map[string]interface{}{
		"app_id":       "test_app",
		"out_trade_no": "ORDER_001",
		"amount":       100,
		"subject":      "测试商品",
		"channel":      "wechat_native",
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/checkout", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)
		}
	})
}
