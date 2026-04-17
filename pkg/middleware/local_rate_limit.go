package middleware

import (
	"sync"
	"time"

	"gopay/pkg/logger"
	"gopay/pkg/response"

	"github.com/gin-gonic/gin"
)

// LocalRateLimitConfig 本地限流配置
type LocalRateLimitConfig struct {
	Rate  int // 每个时间窗口允许的请求数
	Burst int // 突发容量
}

type ipEntry struct {
	count   int
	resetAt time.Time
}

// LocalRateLimit 基于内存的 IP 限流中间件（无 Redis 依赖）
func LocalRateLimit(config LocalRateLimitConfig) gin.HandlerFunc {
	var mu sync.Mutex
	entries := make(map[string]*ipEntry)

	// 后台清理过期条目
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			mu.Lock()
			now := time.Now()
			for ip, e := range entries {
				if now.After(e.resetAt) {
					delete(entries, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now()

		mu.Lock()
		e, exists := entries[ip]
		if !exists || now.After(e.resetAt) {
			entries[ip] = &ipEntry{count: 1, resetAt: now.Add(time.Second)}
			mu.Unlock()
			c.Next()
			return
		}

		e.count++
		if e.count > config.Burst {
			mu.Unlock()
			logger.Error("Rate limit exceeded for IP: %s", ip)
			response.TooManyRequests(c, "请求过于频繁，请稍后再试")
			c.Abort()
			return
		}
		mu.Unlock()
		c.Next()
	}
}
