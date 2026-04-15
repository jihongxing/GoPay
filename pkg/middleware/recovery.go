package middleware

import (
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"gopay/pkg/logger"
	"gopay/pkg/response"
)

// Recovery panic 恢复中间件
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 记录 panic 信息和堆栈
				stack := string(debug.Stack())
				logger.Error("Panic recovered: %v\nStack trace:\n%s", err, stack)

				// 返回 500 错误
				response.InternalError(c, "服务器内部错误")
				c.Abort()
			}
		}()
		c.Next()
	}
}
