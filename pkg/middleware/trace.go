package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"gopay/pkg/logger"

	"github.com/gin-gonic/gin"
)

// TraceContext W3C Trace Context 兼容的追踪中间件
// 支持 OpenTelemetry / Jaeger 等追踪系统的 traceparent header 传播
func TraceContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := ""
		spanID := generateSpanID()

		// 尝试从 W3C traceparent header 解析 trace ID
		// 格式: 00-<trace-id>-<parent-span-id>-<trace-flags>
		if tp := c.GetHeader("traceparent"); len(tp) >= 55 {
			traceID = tp[3:35]
		}

		if traceID == "" {
			traceID = generateTraceID()
		}

		// 设置到上下文
		c.Set("trace_id", traceID)
		c.Set("span_id", spanID)

		// 设置响应头
		traceparent := fmt.Sprintf("00-%s-%s-01", traceID, spanID)
		c.Header("traceparent", traceparent)

		// 记录追踪日志
		logger.Debug("trace: %s %s trace_id=%s span_id=%s",
			c.Request.Method, c.Request.URL.Path, traceID, spanID)

		c.Next()
	}
}

func generateTraceID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func generateSpanID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}
