package middleware

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gopay/pkg/limiter"
	"gopay/pkg/logger"
	"gopay/pkg/response"
)

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	Rate  int // 每秒请求数
	Burst int // 桶容量
}

// RateLimit 限流中间件
func RateLimit(client *redis.Client, config RateLimitConfig) gin.HandlerFunc {
	ipLimiter := limiter.NewIPRateLimiter(client, config.Rate, config.Burst)

	return func(c *gin.Context) {
		ip := c.ClientIP()
		ctx, cancel := context.WithTimeout(c.Request.Context(), 100*time.Millisecond)
		defer cancel()

		allowed, err := ipLimiter.Allow(ctx, ip)
		if err != nil {
			logger.Error("Rate limit check failed: %v", err)
			// 限流检查失败，允许请求通过（fail open）
			c.Next()
			return
		}

		if !allowed {
			logger.Error("Rate limit exceeded for IP: %s", ip)
			response.TooManyRequests(c, "请求过于频繁，请稍后再试")
			c.Abort()
			return
		}

		c.Next()
	}
}
