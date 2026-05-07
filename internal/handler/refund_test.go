package handler

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gopay/internal/service"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupRefundRouter(db *sql.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	mockCM := NewMockChannelManager()
	os := service.NewOrderService(db, mockCM)
	rs := service.NewRefundService(db, os, mockCM)
	InitServices(os)
	InitRefundService(rs)

	router.POST("/orders/:order_no/refund", RefundOrder)
	router.GET("/orders/:order_no/refunds/:refund_no", QueryRefund)
	return router
}

func TestRefundOrder_EmptyOrderNo(t *testing.T) {
	db, _, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	router := setupRefundRouter(db)

	body, _ := json.Marshal(map[string]interface{}{"amount": 100})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/orders//refund", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	// Gin redirects or returns 400 for empty param
	assert.True(t, w.Code == http.StatusBadRequest || w.Code == http.StatusNotFound || w.Code == http.StatusMovedPermanently)
}

func TestRefundOrder_ServiceNotInitialized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Don't init refund service
	oldRS := refundService
	refundService = nil
	defer func() { refundService = oldRS }()

	router.POST("/orders/:order_no/refund", RefundOrder)

	body, _ := json.Marshal(map[string]interface{}{"amount": 100})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/orders/ORD_001/refund", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRefundOrder_InvalidJSON(t *testing.T) {
	db, _, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	router := setupRefundRouter(db)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/orders/ORD_001/refund", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRefundOrder_OrderNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	router := setupRefundRouter(db)

	mock.ExpectQuery("SELECT (.+) FROM orders WHERE order_no").
		WithArgs("ORD_NOTEXIST").
		WillReturnError(sql.ErrNoRows)

	body, _ := json.Marshal(map[string]interface{}{"amount": 100})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/orders/ORD_NOTEXIST/refund", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestQueryRefund_ServiceNotInitialized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	oldRS := refundService
	refundService = nil
	defer func() { refundService = oldRS }()

	router.GET("/orders/:order_no/refunds/:refund_no", QueryRefund)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/orders/ORD_001/refunds/RFD_001", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestQueryRefund_MissingParams(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	db, _, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	_ = setupRefundRouter(db) // init services
	router.GET("/orders/:order_no/refunds/:refund_no", QueryRefund)

	// Both params present but order not found
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/orders//refunds/", nil)
	router.ServeHTTP(w, req)

	// Gin won't match empty params
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRefundOrder_Success(t *testing.T) {
	// This test requires a refundable channel provider which needs
	// deeper integration setup. The refund success path is covered
	// by internal/service/refund_service_test.go.
	t.Skip("Covered by service layer tests")
}
