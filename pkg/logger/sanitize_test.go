package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitize_PhoneNumber(t *testing.T) {
	input := "user phone is 13812345678"
	result := Sanitize(input)
	assert.NotContains(t, result, "12345678")
	assert.Contains(t, result, "138****5678")
}

func TestSanitize_IDCard(t *testing.T) {
	input := "id_card=110101199001011234"
	result := Sanitize(input)
	assert.NotContains(t, result, "19900101")
	assert.Contains(t, result, "****")
}

func TestSanitize_Email(t *testing.T) {
	input := "email is user@example.com"
	result := Sanitize(input)
	assert.NotContains(t, result, "user@")
	assert.Contains(t, result, "use***@example.com")
}

func TestSanitize_BankCard(t *testing.T) {
	input := "card=6222021234567890123"
	result := Sanitize(input)
	assert.NotEqual(t, input, result)
	assert.Contains(t, result, "****")
}

func TestSanitize_SensitiveKeyValue(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"api_key", `api_key=sk_live_abcdef123456`},
		{"password", `password=mysecretpassword`},
		{"json_secret", `{"app_secret":"very_secret_value"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Sanitize(tt.input)
			assert.NotEqual(t, tt.input, result, "should have sanitized something")
		})
	}
}

func TestSanitize_NoSensitiveData(t *testing.T) {
	input := "order_no=ORD_20260417_001 status=paid amount=100"
	result := Sanitize(input)
	assert.Equal(t, input, result)
}

func TestSanitizeMap(t *testing.T) {
	data := map[string]string{
		"order_no":   "ORD_001",
		"app_secret": "very_long_secret_key",
		"api_key":    "sk_live_12345678",
	}
	result := SanitizeMap(data)
	assert.Equal(t, "ORD_001", result["order_no"])
	assert.Equal(t, "very****", result["app_secret"])
	assert.Equal(t, "sk_l****", result["api_key"])
}
