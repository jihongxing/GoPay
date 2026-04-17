package middleware

import (
	"gopay/pkg/security"
)

// InMemoryNonceChecker 基于内存的 nonce 检查器
// 包装 security.NonceManager 实现 NonceChecker 接口
type InMemoryNonceChecker struct {
	manager *security.NonceManager
}

// NewInMemoryNonceChecker 创建内存 nonce 检查器
func NewInMemoryNonceChecker() *InMemoryNonceChecker {
	return &InMemoryNonceChecker{
		manager: security.NewNonceManager(),
	}
}

// Check 检查 nonce 是否已使用，返回 true 表示未使用（合法）
func (c *InMemoryNonceChecker) Check(nonce string) bool {
	return c.manager.CheckNonce(nonce)
}
