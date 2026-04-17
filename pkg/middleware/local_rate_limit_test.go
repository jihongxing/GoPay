package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestLocalRateLimit_AllowsNormalTraffic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(LocalRateLimit(LocalRateLimitConfig{Rate: 100, Burst: 100}))
	router.GET("/test", func(c *gin.Context) {
		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}

func TestLocalRateLimit_BlocksExcessiveTraffic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(LocalRateLimit(LocalRateLimitConfig{Rate: 1, Burst: 2}))
	router.GET("/test", func(c *gin.Context) {
		c.String(200, "ok")
	})

	// First 2 requests should pass (burst=2)
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, 200, w.Code, "request %d should pass", i+1)
	}

	// 3rd request should be rate limited
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, 429, w.Code)
}
