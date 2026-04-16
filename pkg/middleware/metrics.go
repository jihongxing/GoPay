package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gopay/internal/metrics"
)

// PrometheusMetrics 记录 HTTP 请求指标
func PrometheusMetrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		metrics.RecordHTTPRequest(
			c.Request.Method,
			path,
			strconv.Itoa(c.Writer.Status()),
			time.Since(start).Seconds(),
		)
	}
}
