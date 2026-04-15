package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// AccessControl 访问控制
type AccessControl struct {
	ipWhitelist []string
	apiKeys     map[string]bool
}

// NewAccessControl 创建访问控制
func NewAccessControl() *AccessControl {
	return &AccessControl{
		ipWhitelist: make([]string, 0),
		apiKeys:     make(map[string]bool),
	}
}

// AddIPWhitelist 添加 IP 白名单
func (ac *AccessControl) AddIPWhitelist(ips ...string) {
	ac.ipWhitelist = append(ac.ipWhitelist, ips...)
}

// CheckIP 检查 IP 是否在白名单中
func (ac *AccessControl) CheckIP(ip string) bool {
	if len(ac.ipWhitelist) == 0 {
		return true // 未配置白名单，允许所有 IP
	}

	for _, whiteIP := range ac.ipWhitelist {
		if ac.matchIP(ip, whiteIP) {
			return true
		}
	}

	return false
}

// matchIP 匹配 IP（支持 CIDR）
func (ac *AccessControl) matchIP(ip, pattern string) bool {
	// 精确匹配
	if ip == pattern {
		return true
	}

	// CIDR 匹配
	if strings.Contains(pattern, "/") {
		_, ipNet, err := net.ParseCIDR(pattern)
		if err != nil {
			return false
		}
		return ipNet.Contains(net.ParseIP(ip))
	}

	return false
}

// AddAPIKey 添加 API 密钥
func (ac *AccessControl) AddAPIKey(apiKey string) {
	ac.apiKeys[apiKey] = true
}

// CheckAPIKey 检查 API 密钥
func (ac *AccessControl) CheckAPIKey(apiKey string) bool {
	if len(ac.apiKeys) == 0 {
		return true // 未配置 API 密钥，允许所有请求
	}

	return ac.apiKeys[apiKey]
}

// SignatureValidator 签名验证器
type SignatureValidator struct {
	secret string
}

// NewSignatureValidator 创建签名验证器
func NewSignatureValidator(secret string) *SignatureValidator {
	return &SignatureValidator{
		secret: secret,
	}
}

// Sign 生成签名
func (sv *SignatureValidator) Sign(data string, timestamp int64) string {
	message := data + ":" + strconv.FormatInt(timestamp, 10)
	h := hmac.New(sha256.New, []byte(sv.secret))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}

// Verify 验证签名
func (sv *SignatureValidator) Verify(data, signature string, timestamp int64) error {
	// 检查时间戳（防重放攻击）
	now := time.Now().Unix()
	if now-timestamp > 300 { // 5 分钟内有效
		return errors.New("signature expired")
	}

	expectedSignature := sv.Sign(data, timestamp)
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return errors.New("invalid signature")
	}

	return nil
}

// NonceManager 随机数管理器（防重放攻击）
type NonceManager struct {
	mu    sync.RWMutex
	cache map[string]int64 // nonce -> timestamp
}

// NewNonceManager 创建随机数管理器
func NewNonceManager() *NonceManager {
	nm := &NonceManager{
		cache: make(map[string]int64),
	}
	// 启动定期清理过期 nonce 的 goroutine
	go nm.cleanupExpired()
	return nm
}

// CheckNonce 检查随机数
func (nm *NonceManager) CheckNonce(nonce string) bool {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	now := time.Now().Unix()

	// 检查 nonce 是否已使用
	if _, exists := nm.cache[nonce]; exists {
		return false
	}

	// 记录 nonce
	nm.cache[nonce] = now
	return true
}

// cleanupExpired 定期清理过期的 nonce
func (nm *NonceManager) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		nm.mu.Lock()
		now := time.Now().Unix()
		for n, ts := range nm.cache {
			if now-ts > 300 { // 5 分钟
				delete(nm.cache, n)
			}
		}
		nm.mu.Unlock()
	}
}

// AuditLogger 审计日志
type AuditLogger struct {
	// 实际实现在 pkg/audit 包中
}

// NewAuditLogger 创建审计日志
func NewAuditLogger() *AuditLogger {
	return &AuditLogger{}
}

// LogOperation 记录操作
func (al *AuditLogger) LogOperation(userID, action, target, ip string) {
	// 实际实现在 pkg/audit 包中
	// 这里保留接口兼容性
}

// LogSensitiveOperation 记录敏感操作
func (al *AuditLogger) LogSensitiveOperation(userID, action, target, ip string) {
	// 实际实现在 pkg/audit 包中
	// 这里保留接口兼容性
}
