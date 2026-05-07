package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestTraceContext_GeneratesNewTrace(t *testing.T) {
	router := gin.New()
	router.Use(TraceContext())
	router.GET("/test", func(c *gin.Context) {
		traceID, _ := c.Get("trace_id")
		spanID, _ := c.Get("span_id")
		assert.NotEmpty(t, traceID)
		assert.NotEmpty(t, spanID)
		c.JSON(200, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	tp := w.Header().Get("traceparent")
	assert.NotEmpty(t, tp)
	assert.Regexp(t, `^00-[0-9a-f]{32}-[0-9a-f]{16}-01$`, tp)
}

func TestTraceContext_PropagatesExistingTrace(t *testing.T) {
	router := gin.New()
	router.Use(TraceContext())
	router.GET("/test", func(c *gin.Context) {
		traceID, _ := c.Get("trace_id")
		assert.Equal(t, "abcdef1234567890abcdef1234567890", traceID)
		c.JSON(200, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("traceparent", "00-abcdef1234567890abcdef1234567890-1234567890abcdef-01")
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	tp := w.Header().Get("traceparent")
	assert.Contains(t, tp, "abcdef1234567890abcdef1234567890")
}

func TestNonceChecker(t *testing.T) {
	nc := NewInMemoryNonceChecker()

	// First use should be valid
	assert.True(t, nc.Check("nonce-1"))

	// Second use of same nonce should be invalid
	assert.False(t, nc.Check("nonce-1"))

	// Different nonce should be valid
	assert.True(t, nc.Check("nonce-2"))
}

func TestRecovery_PanicRecovery(t *testing.T) {
	router := gin.New()
	router.Use(Recovery())
	router.GET("/panic", func(c *gin.Context) {
		panic("test panic")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/panic", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 500, w.Code)
}
