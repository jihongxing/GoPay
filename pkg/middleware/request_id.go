package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestID 请求 ID 中间件
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 尝试从请求头获取 request_id
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			// 生成新的 request_id
			requestID = uuid.New().String()
		}

		// 设置到上下文和响应头
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}

// GetRequestID 从上下文获取 request_id
func GetRequestID(c *gin.Context) string {
	if requestID, exists := c.Get("request_id"); exists {
		return requestID.(string)
	}
	return ""
}
