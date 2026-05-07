package admin

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupAdminRouter(db *sql.DB) *gin.Engine {
	router := gin.New()
	h := NewWebHandler(db)

	api := router.Group("/admin/api/v1")
	api.GET("/stats", h.GetStats)
	api.GET("/orders/failed", h.GetFailedOrders)
	api.GET("/orders/search", h.SearchOrder)
	api.GET("/orders/:order_no", h.GetOrderDetail)
	api.POST("/orders/:order_no/retry", h.RetryOrder)
	api.POST("/orders/batch-retry", h.BatchRetry)
	api.GET("/stats/orders", h.GetOrderStats)
	api.GET("/stats/notifications", h.GetNotificationStats)

	return router
}

func TestGetStats(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	today := time.Now().Format("2006-01-02")

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM orders").WithArgs(today).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(50))
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM orders").WithArgs(today).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(45))
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM orders").WithArgs(today).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM orders").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	router := setupAdminRouter(db)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/admin/api/v1/stats", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "SUCCESS", resp["code"])
}

func TestSearchOrder_EmptyParam(t *testing.T) {
	db, _, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	router := setupAdminRouter(db)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/admin/api/v1/orders/search", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 400, w.Code)
}

func TestSearchOrder_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery("SELECT (.+) FROM orders").WithArgs("BIZ_NOTEXIST").
		WillReturnError(sql.ErrNoRows)

	router := setupAdminRouter(db)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/admin/api/v1/orders/search?out_trade_no=BIZ_NOTEXIST", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 404, w.Code)
}

func TestSearchOrder_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	now := time.Now()
	rows := sqlmock.NewRows([]string{
		"order_no", "out_trade_no", "app_id", "channel", "amount",
		"status", "notify_status", "notify_url", "retry_count", "channel_order_no",
		"pay_url", "created_at", "paid_at", "notified_at", "updated_at",
	}).AddRow("ORD_001", "BIZ_001", "app1", "wechat_native", 10000,
		"paid", "notified", "http://cb.com", 0, sql.NullString{String: "WX001", Valid: true},
		sql.NullString{}, now, sql.NullTime{Time: now, Valid: true}, sql.NullTime{}, now)

	mock.ExpectQuery("SELECT (.+) FROM orders").WithArgs("BIZ_001").WillReturnRows(rows)

	router := setupAdminRouter(db)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/admin/api/v1/orders/search?out_trade_no=BIZ_001", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "SUCCESS", resp["code"])
}

func TestGetOrderDetail_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery("SELECT (.+) FROM orders").WithArgs("ORD_NOTEXIST").
		WillReturnError(sql.ErrNoRows)

	router := setupAdminRouter(db)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/admin/api/v1/orders/ORD_NOTEXIST", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 404, w.Code)
}

func TestRetryOrder_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery("SELECT status, notify_status FROM orders").WithArgs("ORD_NOTEXIST").
		WillReturnError(sql.ErrNoRows)

	router := setupAdminRouter(db)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/admin/api/v1/orders/ORD_NOTEXIST/retry", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 404, w.Code)
}

func TestRetryOrder_NotPaid(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery("SELECT status, notify_status FROM orders").WithArgs("ORD_001").
		WillReturnRows(sqlmock.NewRows([]string{"status", "notify_status"}).AddRow("pending", "pending"))

	router := setupAdminRouter(db)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/admin/api/v1/orders/ORD_001/retry", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 400, w.Code)
}

func TestRetryOrder_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery("SELECT status, notify_status FROM orders").WithArgs("ORD_001").
		WillReturnRows(sqlmock.NewRows([]string{"status", "notify_status"}).AddRow("paid", "failed"))
	mock.ExpectExec("UPDATE orders").WithArgs("ORD_001").
		WillReturnResult(sqlmock.NewResult(1, 1))

	router := setupAdminRouter(db)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/admin/api/v1/orders/ORD_001/retry", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "SUCCESS", resp["code"])
}

func TestBatchRetry_InvalidJSON(t *testing.T) {
	db, _, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	router := setupAdminRouter(db)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/admin/api/v1/orders/batch-retry", bytes.NewReader([]byte("bad")))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, 400, w.Code)
}

func TestBatchRetry_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectExec("UPDATE orders").WithArgs("ORD_001").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("UPDATE orders").WithArgs("ORD_002").
		WillReturnResult(sqlmock.NewResult(1, 1))

	router := setupAdminRouter(db)
	body, _ := json.Marshal(map[string]interface{}{
		"order_nos": []string{"ORD_001", "ORD_002"},
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/admin/api/v1/orders/batch-retry", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, float64(2), data["success"])
}

func TestGetOrderStats_DBError(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery("SELECT DATE").WillReturnError(sql.ErrConnDone)

	router := setupAdminRouter(db)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/admin/api/v1/stats/orders?days=7", nil)
	router.ServeHTTP(w, req)

	// Should return 200 with empty data, not error
	assert.Equal(t, 200, w.Code)
}

func TestGetNotificationStats_DBError(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery("SELECT DATE").WillReturnError(sql.ErrConnDone)

	router := setupAdminRouter(db)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/admin/api/v1/stats/notifications?days=7", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
}
