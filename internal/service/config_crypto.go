package service

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"gopay/pkg/security"
)

var sensitiveConfigTokens = []string{
	"secret",
	"private_key",
	"public_key",
	"api_key",
	"token",
	"password",
}

func newConfigKeyManager() *security.KeyManager {
	masterKey := strings.TrimSpace(os.Getenv("MASTER_KEY"))
	if masterKey == "" {
		return nil
	}
	return security.NewKeyManager(masterKey)
}

func isSensitiveConfigKey(key string) bool {
	lowerKey := strings.ToLower(strings.TrimSpace(key))
	for _, token := range sensitiveConfigTokens {
		if strings.Contains(lowerKey, token) {
			return true
		}
	}
	return false
}

func encryptConfigJSON(raw string) (string, error) {
	km := newConfigKeyManager()

	var payload any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return "", err
	}

	encrypted, err := transformConfigValue(payload, km, transformEncrypt, "")
	if err != nil {
		return "", err
	}

	data, err := json.Marshal(encrypted)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func decryptConfigJSON(raw string) (string, error) {
	var payload any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return "", err
	}

	decrypted, err := transformConfigValue(payload, newConfigKeyManager(), transformDecrypt, "")
	if err != nil {
		return "", err
	}

	data, err := json.Marshal(decrypted)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// MaskSensitiveConfigJSON 返回适合 API / 审计日志展示的脱敏配置。
func MaskSensitiveConfigJSON(raw string) string {
	var payload any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return raw
	}

	masked, err := transformConfigValue(payload, nil, transformMask, "")
	if err != nil {
		return raw
	}

	data, err := json.Marshal(masked)
	if err != nil {
		return raw
	}
	return string(data)
}

type transformMode int

const (
	transformEncrypt transformMode = iota + 1
	transformDecrypt
	transformMask
)

func transformConfigValue(value any, km *security.KeyManager, mode transformMode, parentKey string) (any, error) {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, child := range typed {
			next, err := transformConfigValue(child, km, mode, key)
			if err != nil {
				return nil, err
			}
			out[key] = next
		}
		return out, nil
	case []any:
		out := make([]any, len(typed))
		for i, child := range typed {
			next, err := transformConfigValue(child, km, mode, parentKey)
			if err != nil {
				return nil, err
			}
			out[i] = next
		}
		return out, nil
	case string:
		sensitive := isSensitiveConfigKey(parentKey) || security.IsEncryptedValue(typed)
		if !sensitive || typed == "" {
			return typed, nil
		}

		switch mode {
		case transformEncrypt:
			if security.IsEncryptedValue(typed) {
				return typed, nil
			}
			if km == nil {
				return nil, fmt.Errorf("key manager is required for encryption")
			}
			return km.Seal(typed)
		case transformDecrypt:
			if !security.IsEncryptedValue(typed) {
				return typed, nil
			}
			if km == nil {
				return nil, fmt.Errorf("MASTER_KEY is required to decrypt sensitive channel config")
			}
			return km.Open(typed)
		case transformMask:
			return "******", nil
		default:
			return typed, nil
		}
	default:
		return value, nil
	}
}
