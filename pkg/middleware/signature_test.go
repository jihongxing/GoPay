package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func signRequest(body, secret, timestamp, nonce string) string {
	message := body + "\n" + timestamp + "\n" + nonce
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}

func setupSignatureRouter(db *sql.DB, nonceChecker NonceChecker) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/checkout", SignatureAuth(db, nonceChecker), func(c *gin.Context) {
		appID, _ := c.Get("verified_app_id")
		c.JSON(200, gin.H{"app_id": appID})
	})
	return r
}

func TestSignatureAuth_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery("SELECT app_secret, status FROM apps WHERE app_id").
		WithArgs("test_app").
		WillReturnRows(sqlmock.NewRows([]string{"app_secret", "status"}).AddRow("my_secret", "active"))

	body := `{"app_id":"test_app","amount":100}`
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	nonce := "random123"
	sig := signRequest(body, "my_secret", ts, nonce)

	router := setupSignatureRouter(db, nil)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/checkout", bytes.NewBufferString(body))
	req.Header.Set("X-App-ID", "test_app")
	req.Header.Set("X-Timestamp", ts)
	req.Header.Set("X-Nonce", nonce)
	req.Header.Set("X-Signature", sig)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "test_app")
}

func TestSignatureAuth_MissingHeaders(t *testing.T) {
	db, _, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	router := setupSignatureRouter(db, nil)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/checkout", bytes.NewBufferString(`{}`))
	router.ServeHTTP(w, req)

	assert.Equal(t, 401, w.Code)
}

func TestSignatureAuth_ExpiredTimestamp(t *testing.T) {
	db, _, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	body := `{"app_id":"test_app"}`
	ts := strconv.FormatInt(time.Now().Unix()-600, 10) // 10 minutes ago
	nonce := "random123"

	router := setupSignatureRouter(db, nil)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/checkout", bytes.NewBufferString(body))
	req.Header.Set("X-App-ID", "test_app")
	req.Header.Set("X-Timestamp", ts)
	req.Header.Set("X-Nonce", nonce)
	req.Header.Set("X-Signature", "fake")
	router.ServeHTTP(w, req)

	assert.Equal(t, 401, w.Code)
	assert.Contains(t, w.Body.String(), "TIMESTAMP_EXPIRED")
}

func TestSignatureAuth_InvalidSignature(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery("SELECT app_secret, status FROM apps WHERE app_id").
		WithArgs("test_app").
		WillReturnRows(sqlmock.NewRows([]string{"app_secret", "status"}).AddRow("my_secret", "active"))

	body := `{"app_id":"test_app"}`
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	nonce := "random123"

	router := setupSignatureRouter(db, nil)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/checkout", bytes.NewBufferString(body))
	req.Header.Set("X-App-ID", "test_app")
	req.Header.Set("X-Timestamp", ts)
	req.Header.Set("X-Nonce", nonce)
	req.Header.Set("X-Signature", "wrong_signature")
	router.ServeHTTP(w, req)

	assert.Equal(t, 401, w.Code)
	assert.Contains(t, w.Body.String(), "SIGNATURE_INVALID")
}

func TestSignatureAuth_AppNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery("SELECT app_secret, status FROM apps WHERE app_id").
		WithArgs("nonexistent").
		WillReturnError(sql.ErrNoRows)

	body := `{"app_id":"nonexistent"}`
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	nonce := "random123"

	router := setupSignatureRouter(db, nil)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/checkout", bytes.NewBufferString(body))
	req.Header.Set("X-App-ID", "nonexistent")
	req.Header.Set("X-Timestamp", ts)
	req.Header.Set("X-Nonce", nonce)
	req.Header.Set("X-Signature", "fake")
	router.ServeHTTP(w, req)

	assert.Equal(t, 401, w.Code)
}

func TestSignatureAuth_DisabledApp(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery("SELECT app_secret, status FROM apps WHERE app_id").
		WithArgs("disabled_app").
		WillReturnRows(sqlmock.NewRows([]string{"app_secret", "status"}).AddRow("secret", "disabled"))

	body := `{}`
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	nonce := "random123"

	router := setupSignatureRouter(db, nil)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/checkout", bytes.NewBufferString(body))
	req.Header.Set("X-App-ID", "disabled_app")
	req.Header.Set("X-Timestamp", ts)
	req.Header.Set("X-Nonce", nonce)
	req.Header.Set("X-Signature", "fake")
	router.ServeHTTP(w, req)

	assert.Equal(t, 401, w.Code)
	assert.Contains(t, w.Body.String(), "APP_INACTIVE")
}

func TestSignatureAuth_NonceReplay(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	// First request succeeds
	mock.ExpectQuery("SELECT app_secret, status FROM apps WHERE app_id").
		WithArgs("test_app").
		WillReturnRows(sqlmock.NewRows([]string{"app_secret", "status"}).AddRow("my_secret", "active"))

	body := `{"app_id":"test_app"}`
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	nonce := "unique_nonce_123"
	sig := signRequest(body, "my_secret", ts, nonce)

	nonceChecker := NewInMemoryNonceChecker()
	router := setupSignatureRouter(db, nonceChecker)

	// First request
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/checkout", bytes.NewBufferString(body))
	req.Header.Set("X-App-ID", "test_app")
	req.Header.Set("X-Timestamp", ts)
	req.Header.Set("X-Nonce", nonce)
	req.Header.Set("X-Signature", sig)
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	// Replay with same nonce — should be rejected
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/api/v1/checkout", bytes.NewBufferString(body))
	req2.Header.Set("X-App-ID", "test_app")
	req2.Header.Set("X-Timestamp", ts)
	req2.Header.Set("X-Nonce", nonce)
	req2.Header.Set("X-Signature", sig)
	router.ServeHTTP(w2, req2)
	assert.Equal(t, 401, w2.Code)
	assert.Contains(t, w2.Body.String(), "NONCE_REPLAY")
}

// Suppress unused import warning
var _ = fmt.Sprintf
