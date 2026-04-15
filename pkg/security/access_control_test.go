package security

import (
	"testing"
	"time"
)

// TestNonceManager_CheckNonce 测试 nonce 检查
func TestNonceManager_CheckNonce(t *testing.T) {
	nm := NewNonceManager()

	// 第一次使用 nonce 应该成功
	nonce1 := "test_nonce_001"
	if !nm.CheckNonce(nonce1) {
		t.Errorf("First use of nonce should succeed")
	}

	// 重复使用相同 nonce 应该失败
	if nm.CheckNonce(nonce1) {
		t.Errorf("Duplicate nonce should fail")
	}

	// 使用不同 nonce 应该成功
	nonce2 := "test_nonce_002"
	if !nm.CheckNonce(nonce2) {
		t.Errorf("Different nonce should succeed")
	}
}

// TestNonceManager_Concurrent 测试并发安全
func TestNonceManager_Concurrent(t *testing.T) {
	nm := NewNonceManager()

	// 并发测试
	done := make(chan bool)
	for i := 0; i < 100; i++ {
		go func(n int) {
			nonce := string(rune(n))
			nm.CheckNonce(nonce)
			done <- true
		}(i)
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 100; i++ {
		<-done
	}

	// 如果没有 panic，说明并发安全
}

// TestSignatureValidator_Sign 测试签名生成
func TestSignatureValidator_Sign(t *testing.T) {
	sv := NewSignatureValidator("test_secret")

	data := "test_data"
	timestamp := time.Now().Unix()

	signature1 := sv.Sign(data, timestamp)
	signature2 := sv.Sign(data, timestamp)

	// 相同数据和时间戳应该生成相同签名
	if signature1 != signature2 {
		t.Errorf("Same data should generate same signature")
	}

	// 签名应该是 64 字符的十六进制字符串（SHA256）
	if len(signature1) != 64 {
		t.Errorf("Signature length = %d, want 64", len(signature1))
	}
}

// TestSignatureValidator_Verify 测试签名验证
func TestSignatureValidator_Verify(t *testing.T) {
	sv := NewSignatureValidator("test_secret")

	data := "test_data"
	timestamp := time.Now().Unix()
	signature := sv.Sign(data, timestamp)

	tests := []struct {
		name      string
		data      string
		signature string
		timestamp int64
		wantErr   bool
	}{
		{
			name:      "valid signature",
			data:      data,
			signature: signature,
			timestamp: timestamp,
			wantErr:   false,
		},
		{
			name:      "invalid signature",
			data:      data,
			signature: "invalid_signature",
			timestamp: timestamp,
			wantErr:   true,
		},
		{
			name:      "expired timestamp",
			data:      data,
			signature: signature,
			timestamp: timestamp - 400, // 超过 5 分钟
			wantErr:   true,
		},
		{
			name:      "wrong data",
			data:      "wrong_data",
			signature: signature,
			timestamp: timestamp,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := sv.Verify(tt.data, tt.signature, tt.timestamp)
			if (err != nil) != tt.wantErr {
				t.Errorf("Verify() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestAccessControl_CheckIP 测试 IP 白名单
func TestAccessControl_CheckIP(t *testing.T) {
	ac := NewAccessControl()

	// 未配置白名单，应该允许所有 IP
	if !ac.CheckIP("192.168.1.1") {
		t.Errorf("Should allow all IPs when whitelist is empty")
	}

	// 添加白名单
	ac.AddIPWhitelist("192.168.1.1", "10.0.0.0/8")

	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{
			name: "exact match",
			ip:   "192.168.1.1",
			want: true,
		},
		{
			name: "not in whitelist",
			ip:   "192.168.1.2",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ac.CheckIP(tt.ip)
			if got != tt.want {
				t.Errorf("CheckIP() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestAccessControl_CheckAPIKey 测试 API 密钥验证
func TestAccessControl_CheckAPIKey(t *testing.T) {
	ac := NewAccessControl()

	// 未配置 API 密钥，应该允许所有请求
	if !ac.CheckAPIKey("any_key") {
		t.Errorf("Should allow all keys when no keys configured")
	}

	// 添加 API 密钥
	ac.AddAPIKey("valid_key_001")
	ac.AddAPIKey("valid_key_002")

	tests := []struct {
		name   string
		apiKey string
		want   bool
	}{
		{
			name:   "valid key 1",
			apiKey: "valid_key_001",
			want:   true,
		},
		{
			name:   "valid key 2",
			apiKey: "valid_key_002",
			want:   true,
		},
		{
			name:   "invalid key",
			apiKey: "invalid_key",
			want:   false,
		},
		{
			name:   "empty key",
			apiKey: "",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ac.CheckAPIKey(tt.apiKey)
			if got != tt.want {
				t.Errorf("CheckAPIKey() = %v, want %v", got, tt.want)
			}
		})
	}
}
