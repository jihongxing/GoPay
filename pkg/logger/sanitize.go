package logger

import (
	"regexp"
	"strings"
)

var sensitivePatterns = []*regexp.Regexp{
	// 手机号
	regexp.MustCompile(`(1[3-9]\d)\d{4}(\d{4})`),
	// 身份证号
	regexp.MustCompile(`(\d{6})\d{8}(\d{3}[\dXx])`),
	// 邮箱
	regexp.MustCompile(`([\w.]{1,3})[\w.]*@([\w.]+)`),
	// 银行卡号（16-19位数字）
	regexp.MustCompile(`(\d{4})\d{8,12}(\d{4})`),
}

var sensitiveReplacements = []string{
	"$1****$2",
	"$1********$2",
	"$1***@$2",
	"$1****$2",
}

// 敏感 key 名称（JSON key 或日志字段名）
var sensitiveKeys = []string{
	"app_secret", "api_key", "private_key", "api_v3_key",
	"password", "secret", "token", "authorization",
	"mch_id", "serial_no",
}

// Sanitize 对日志内容进行脱敏处理
func Sanitize(input string) string {
	result := input

	// 正则脱敏
	for i, pattern := range sensitivePatterns {
		result = pattern.ReplaceAllString(result, sensitiveReplacements[i])
	}

	// 敏感字段值脱敏（简单的 key=value 和 JSON 格式）
	for _, key := range sensitiveKeys {
		result = maskKeyValue(result, key)
	}

	return result
}

// maskKeyValue 对 key=value 或 "key":"value" 格式的敏感字段进行脱敏
func maskKeyValue(input, key string) string {
	// 处理 key=value 格式
	kvPattern := regexp.MustCompile(`(?i)(` + regexp.QuoteMeta(key) + `\s*[=:]\s*)([^\s,}"]{4})[^\s,}"]*`)
	input = kvPattern.ReplaceAllString(input, "${1}${2}****")

	// 处理 JSON "key":"value" 格式
	jsonPattern := regexp.MustCompile(`(?i)("` + regexp.QuoteMeta(key) + `"\s*:\s*")([^"]{4})[^"]*"`)
	input = jsonPattern.ReplaceAllString(input, `${1}${2}****"`)

	return input
}

// SanitizeMap 对 map 中的敏感字段进行脱敏
func SanitizeMap(data map[string]string) map[string]string {
	result := make(map[string]string, len(data))
	for k, v := range data {
		if isSensitiveKey(k) && len(v) > 4 {
			result[k] = v[:4] + "****"
		} else {
			result[k] = v
		}
	}
	return result
}

func isSensitiveKey(key string) bool {
	lower := strings.ToLower(key)
	for _, sk := range sensitiveKeys {
		if strings.Contains(lower, sk) {
			return true
		}
	}
	return false
}
