package middleware

import (
	"crypto/subtle"
	"net"
	"strings"

	"github.com/gin-gonic/gin"
	"gopay/pkg/logger"
	"gopay/pkg/response"
)

// AuthConfig 认证配置
type AuthConfig struct {
	APIKeys     map[string]bool
	IPWhitelist []string
}

// NewAuthConfig 创建认证配置
func NewAuthConfig() *AuthConfig {
	return &AuthConfig{
		APIKeys:     make(map[string]bool),
		IPWhitelist: make([]string, 0),
	}
}

// AddAPIKey 添加 API 密钥
func (ac *AuthConfig) AddAPIKey(apiKey string) {
	ac.APIKeys[apiKey] = true
}

// AddIPWhitelist 添加 IP 白名单
func (ac *AuthConfig) AddIPWhitelist(ips ...string) {
	ac.IPWhitelist = append(ac.IPWhitelist, ips...)
}

// APIKeyAuth API 密钥认证中间件
func APIKeyAuth(config *AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			logger.Error("Missing API key from IP: %s", c.ClientIP())
			response.Unauthorized(c, "缺少 API 密钥")
			c.Abort()
			return
		}

		// 使用常量时间比较防止时序攻击
		valid := false
		for key := range config.APIKeys {
			if subtle.ConstantTimeCompare([]byte(apiKey), []byte(key)) == 1 {
				valid = true
				break
			}
		}

		if !valid {
			logger.Error("Invalid API key from IP: %s", c.ClientIP())
			response.Unauthorized(c, "无效的 API 密钥")
			c.Abort()
			return
		}

		c.Next()
	}
}

// IPWhitelist IP 白名单中间件
func IPWhitelist(config *AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if len(config.IPWhitelist) == 0 {
			// 未配置白名单，允许所有 IP
			c.Next()
			return
		}

		clientIP := c.ClientIP()
		allowed := false

		for _, whiteIP := range config.IPWhitelist {
			if matchIP(clientIP, whiteIP) {
				allowed = true
				break
			}
		}

		if !allowed {
			logger.Error("IP not in whitelist: %s", clientIP)
			response.Forbidden(c, "IP 地址未授权")
			c.Abort()
			return
		}

		c.Next()
	}
}

// matchIP 匹配 IP（支持 CIDR）
func matchIP(ip, pattern string) bool {
	parsedIP := net.ParseIP(strings.TrimSpace(ip))
	if parsedIP == nil {
		return false
	}

	pattern = strings.TrimSpace(pattern)

	// 精确匹配
	if parsedPattern := net.ParseIP(pattern); parsedPattern != nil {
		return parsedIP.Equal(parsedPattern)
	}

	// CIDR 匹配
	if _, ipNet, err := net.ParseCIDR(pattern); err == nil {
		return ipNet.Contains(parsedIP)
	}

	if ip == pattern {
		return true
	}

	return false
}
