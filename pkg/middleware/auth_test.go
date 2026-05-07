package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestAPIKeyAuth_ValidKey(t *testing.T) {
	config := NewAuthConfig()
	config.AddAPIKey("test-key-123")

	router := gin.New()
	router.Use(APIKeyAuth(config))
	router.GET("/test", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "test-key-123")
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}

func TestAPIKeyAuth_MissingKey(t *testing.T) {
	config := NewAuthConfig()
	config.AddAPIKey("test-key-123")

	router := gin.New()
	router.Use(APIKeyAuth(config))
	router.GET("/test", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 401, w.Code)
}

func TestAPIKeyAuth_InvalidKey(t *testing.T) {
	config := NewAuthConfig()
	config.AddAPIKey("test-key-123")

	router := gin.New()
	router.Use(APIKeyAuth(config))
	router.GET("/test", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "wrong-key")
	router.ServeHTTP(w, req)

	assert.Equal(t, 401, w.Code)
}

func TestIPWhitelist_AllowedIP(t *testing.T) {
	config := NewAuthConfig()
	config.AddIPWhitelist("127.0.0.1")

	router := gin.New()
	router.Use(IPWhitelist(config))
	router.GET("/test", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}

func TestIPWhitelist_BlockedIP(t *testing.T) {
	config := NewAuthConfig()
	config.AddIPWhitelist("10.0.0.1")

	router := gin.New()
	router.Use(IPWhitelist(config))
	router.GET("/test", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	router.ServeHTTP(w, req)

	assert.Equal(t, 403, w.Code)
}

func TestIPWhitelist_EmptyWhitelist(t *testing.T) {
	config := NewAuthConfig()

	router := gin.New()
	router.Use(IPWhitelist(config))
	router.GET("/test", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "1.2.3.4:12345"
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}

func TestMatchIP(t *testing.T) {
	assert.True(t, matchIP("127.0.0.1", "127.0.0.1"))
	assert.False(t, matchIP("127.0.0.1", "10.0.0.1"))
	assert.True(t, matchIP("192.168.1.1", "192.168.0.0/16"))
	assert.True(t, matchIP("10.1.2.3", "10.0.0.0/8"))
	assert.False(t, matchIP("172.16.0.1", "192.168.0.0/16"))
}

func TestRequestID(t *testing.T) {
	router := gin.New()
	router.Use(RequestID())
	router.GET("/test", func(c *gin.Context) {
		rid := GetRequestID(c)
		c.JSON(200, gin.H{"request_id": rid})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.NotEmpty(t, w.Header().Get("X-Request-ID"))
}

func TestRequestID_ExistingHeader(t *testing.T) {
	router := gin.New()
	router.Use(RequestID())
	router.GET("/test", func(c *gin.Context) {
		rid := GetRequestID(c)
		c.JSON(200, gin.H{"request_id": rid})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", "custom-id-123")
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "custom-id-123", w.Header().Get("X-Request-ID"))
}
