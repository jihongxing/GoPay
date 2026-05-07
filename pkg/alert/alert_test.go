package alert

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gopay/internal/models"

	"github.com/stretchr/testify/assert"
)

func TestNewAlertManager(t *testing.T) {
	am := NewAlertManager("https://example.com/webhook")
	assert.NotNil(t, am)
	assert.Equal(t, "https://example.com/webhook", am.webhookURL)
}

func TestSendAlert_Success(t *testing.T) {
	var received AlertMessage
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	am := NewAlertManager(server.URL)
	err := am.SendAlert(&AlertMessage{
		Level:   AlertLevelError,
		Title:   "测试告警",
		Content: "测试内容",
	})

	assert.NoError(t, err)
	assert.Equal(t, AlertLevelError, received.Level)
	assert.Equal(t, "测试告警", received.Title)
}

func TestSendAlert_EmptyURL(t *testing.T) {
	am := NewAlertManager("")
	err := am.SendAlert(&AlertMessage{Level: AlertLevelInfo, Title: "test"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

func TestSendAlert_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	am := NewAlertManager(server.URL)
	err := am.SendAlert(&AlertMessage{Level: AlertLevelError, Title: "test"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status 500")
}

func TestAlertNotifyFailed(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		var msg AlertMessage
		json.NewDecoder(r.Body).Decode(&msg)
		assert.Equal(t, AlertLevelError, msg.Level)
		assert.Contains(t, msg.Content, "ORD_001")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	am := NewAlertManager(server.URL)
	am.AlertNotifyFailed(&models.Order{
		OrderNo:    "ORD_001",
		OutTradeNo: "BIZ_001",
		AppID:      "app1",
		Amount:     100,
		RetryCount: 3,
	})
	assert.True(t, called)
}

func TestAlertPaymentAbnormal(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	am := NewAlertManager(server.URL)
	am.AlertPaymentAbnormal("ORD_001", "金额不匹配", map[string]string{"expected": "100", "actual": "200"})
	assert.True(t, called)
}

func TestAlertSystemError(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	am := NewAlertManager(server.URL)
	am.AlertSystemError("database", assert.AnError, nil)
	assert.True(t, called)
}

func TestAlertHighRetryRate(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	am := NewAlertManager(server.URL)
	am.AlertHighRetryRate(0.15, 30)
	assert.True(t, called)
}

func TestDingTalkAlertManager_SendAlert_Success(t *testing.T) {
	var received DingTalkMessage
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	dm := NewDingTalkAlertManager(server.URL)
	err := dm.SendAlert(&AlertMessage{
		Level:   AlertLevelWarning,
		Title:   "钉钉测试",
		Content: "测试内容",
		Details: map[string]string{"key": "value"},
	})

	assert.NoError(t, err)
	assert.Equal(t, "text", received.MsgType)
	assert.Contains(t, received.Text.Content, "钉钉测试")
}

func TestDingTalkAlertManager_SendAlert_EmptyURL(t *testing.T) {
	dm := NewDingTalkAlertManager("")
	err := dm.SendAlert(&AlertMessage{Level: AlertLevelInfo, Title: "test"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

func TestDingTalkAlertManager_SendAlert_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	dm := NewDingTalkAlertManager(server.URL)
	err := dm.SendAlert(&AlertMessage{Level: AlertLevelError, Title: "test"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status 502")
}
