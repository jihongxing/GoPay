package admin

import (
	"database/sql"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// WebHandler 管理后台 Web 处理器
type WebHandler struct {
	db *sql.DB
}

// NewWebHandler 创建 Web 处理器
func NewWebHandler(db *sql.DB) *WebHandler {
	return &WebHandler{
		db: db,
	}
}

// RegisterRoutesWithAuth 注册路由（带认证）
func (h *WebHandler) RegisterRoutesWithAuth(r *gin.Engine, authMiddleware ...gin.HandlerFunc) {
	// 加载模板
	r.LoadHTMLGlob("web/admin/templates/*.html")

	// 静态文件
	r.Static("/admin/static", "./web/admin/static")

	admin := r.Group("/admin")
	// 应用认证中间件
	admin.Use(authMiddleware...)
	{
		// 页面路由
		admin.GET("", h.Dashboard)
		admin.GET("/orders", h.OrdersPage)
		admin.GET("/reconciliation", h.ReconciliationPage)
		admin.GET("/logs", h.LogsPage)

		// API 路由
		api := admin.Group("/api/v1")
		{
			// 统计数据
			api.GET("/stats", h.GetStats)

			// 订单管理
			api.GET("/orders/failed", h.GetFailedOrders)
			api.GET("/orders/search", h.SearchOrder)
			api.GET("/orders/:order_no", h.GetOrderDetail)
			api.POST("/orders/:order_no/retry", h.RetryOrder)
			api.POST("/orders/batch-retry", h.BatchRetry)

			// 对账报告
			api.GET("/reconciliation/reports", h.GetReconciliationReports)
			api.GET("/reconciliation/:id", h.GetReconciliationDetail)
			api.GET("/reconciliation/:id/download", h.DownloadReport)

			// 统计图表
			api.GET("/stats/orders", h.GetOrderStats)
			api.GET("/stats/notifications", h.GetNotificationStats)
		}
	}
}

// RegisterRoutes 注册路由（无认证，向后兼容）
func (h *WebHandler) RegisterRoutes(r *gin.Engine) {
	h.RegisterRoutesWithAuth(r)
}

// Dashboard 数据概览页面
func (h *WebHandler) Dashboard(c *gin.Context) {
	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"Title": "数据概览",
	})
}

// OrdersPage 订单管理页面
func (h *WebHandler) OrdersPage(c *gin.Context) {
	c.HTML(http.StatusOK, "orders.html", gin.H{
		"Title": "订单管理",
	})
}

// ReconciliationPage 对账报告页面
func (h *WebHandler) ReconciliationPage(c *gin.Context) {
	c.HTML(http.StatusOK, "reconciliation.html", gin.H{
		"Title": "对账报告",
	})
}

// LogsPage 操作日志页面
func (h *WebHandler) LogsPage(c *gin.Context) {
	c.HTML(http.StatusOK, "logs.html", gin.H{
		"Title": "操作日志",
	})
}

// GetStats 获取统计数据
func (h *WebHandler) GetStats(c *gin.Context) {
	today := time.Now().Format("2006-01-02")

	// 今日订单数
	var todayOrders int
	h.db.QueryRow(`
		SELECT COUNT(*) FROM orders
		WHERE DATE(created_at) = $1
	`, today).Scan(&todayOrders)

	// 今日成功订单
	var successOrders int
	h.db.QueryRow(`
		SELECT COUNT(*) FROM orders
		WHERE DATE(created_at) = $1 AND status = 'paid'
	`, today).Scan(&successOrders)

	// 今日失败订单
	var failedOrders int
	h.db.QueryRow(`
		SELECT COUNT(*) FROM orders
		WHERE DATE(created_at) = $1 AND status = 'failed'
	`, today).Scan(&failedOrders)

	// 通知失败数
	var failedNotifications int
	h.db.QueryRow(`
		SELECT COUNT(*) FROM orders
		WHERE notify_status = 'failed'
	`).Scan(&failedNotifications)

	c.JSON(http.StatusOK, gin.H{
		"code":    "SUCCESS",
		"message": "查询成功",
		"data": gin.H{
			"today_orders":         todayOrders,
			"success_orders":       successOrders,
			"failed_orders":        failedOrders,
			"failed_notifications": failedNotifications,
		},
	})
}

// GetFailedOrders 获取失败订单列表
func (h *WebHandler) GetFailedOrders(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	channel := c.Query("channel")
	status := c.Query("status")
	notifyStatus := c.Query("notify_status")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	offset := (page - 1) * pageSize

	// 构建查询条件
	where := "WHERE (status = 'failed' OR notify_status = 'failed')"
	args := []interface{}{}
	argIndex := 1

	if channel != "" {
		where += " AND channel = $" + strconv.Itoa(argIndex)
		args = append(args, channel)
		argIndex++
	}

	if status != "" {
		where += " AND status = $" + strconv.Itoa(argIndex)
		args = append(args, status)
		argIndex++
	}

	if notifyStatus != "" {
		where += " AND notify_status = $" + strconv.Itoa(argIndex)
		args = append(args, notifyStatus)
		argIndex++
	}

	if startDate != "" {
		where += " AND DATE(created_at) >= $" + strconv.Itoa(argIndex)
		args = append(args, startDate)
		argIndex++
	}

	if endDate != "" {
		where += " AND DATE(created_at) <= $" + strconv.Itoa(argIndex)
		args = append(args, endDate)
		argIndex++
	}

	// 查询总数
	var total int
	h.db.QueryRow("SELECT COUNT(*) FROM orders "+where, args...).Scan(&total)

	// 查询数据
	args = append(args, pageSize, offset)
	rows, err := h.db.Query(`
		SELECT order_no, out_trade_no, app_id, channel, amount,
		       status, notify_status, retry_count, created_at, paid_at
		FROM orders `+where+`
		ORDER BY created_at DESC
		LIMIT $`+strconv.Itoa(argIndex)+` OFFSET $`+strconv.Itoa(argIndex+1),
		args...)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    "ERROR",
			"message": "查询失败: " + err.Error(),
		})
		return
	}
	defer rows.Close()

	orders := []map[string]any{}
	for rows.Next() {
		var orderNo, outTradeNo, appID, channel, status, notifyStatus string
		var amount, retryCount int
		var createdAt time.Time
		var paidAt sql.NullTime

		rows.Scan(&orderNo, &outTradeNo, &appID, &channel, &amount,
			&status, &notifyStatus, &retryCount, &createdAt, &paidAt)

		order := map[string]any{
			"order_no":      orderNo,
			"out_trade_no":  outTradeNo,
			"app_id":        appID,
			"channel":       channel,
			"amount":        amount,
			"status":        status,
			"notify_status": notifyStatus,
			"retry_count":   retryCount,
			"created_at":    createdAt,
		}

		if paidAt.Valid {
			order["paid_at"] = paidAt.Time
		}

		orders = append(orders, order)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    "SUCCESS",
		"message": "查询成功",
		"data": gin.H{
			"total":     total,
			"page":      page,
			"page_size": pageSize,
			"orders":    orders,
		},
	})
}

// SearchOrder 搜索订单
func (h *WebHandler) SearchOrder(c *gin.Context) {
	outTradeNo := c.Query("out_trade_no")
	if outTradeNo == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "ERROR",
			"message": "商户订单号不能为空",
		})
		return
	}

	var orderNo, appID, channel, status, notifyStatus, notifyURL string
	var amount, retryCount int
	var createdAt, updatedAt time.Time
	var paidAt, notifiedAt sql.NullTime
	var channelOrderNo, payURL sql.NullString

	err := h.db.QueryRow(`
		SELECT order_no, out_trade_no, app_id, channel, amount,
		       status, notify_status, notify_url, retry_count, channel_order_no,
		       pay_url, created_at, paid_at, notified_at, updated_at
		FROM orders
		WHERE out_trade_no = $1
	`, outTradeNo).Scan(&orderNo, &outTradeNo, &appID, &channel, &amount,
		&status, &notifyStatus, &notifyURL, &retryCount, &channelOrderNo,
		&payURL, &createdAt, &paidAt, &notifiedAt, &updatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    "ERROR",
			"message": "订单不存在",
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    "ERROR",
			"message": "查询失败",
		})
		return
	}

	order := gin.H{
		"order_no":      orderNo,
		"out_trade_no":  outTradeNo,
		"app_id":        appID,
		"channel":       channel,
		"amount":        amount,
		"status":        status,
		"notify_status": notifyStatus,
		"notify_url":    notifyURL,
		"retry_count":   retryCount,
		"created_at":    createdAt,
		"updated_at":    updatedAt,
	}

	if channelOrderNo.Valid {
		order["channel_order_no"] = channelOrderNo.String
	}
	if payURL.Valid {
		order["pay_url"] = payURL.String
	}
	if paidAt.Valid {
		order["paid_at"] = paidAt.Time
	}
	if notifiedAt.Valid {
		order["notified_at"] = notifiedAt.Time
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    "SUCCESS",
		"message": "查询成功",
		"data":    order,
	})
}

// GetOrderDetail 获取订单详情
func (h *WebHandler) GetOrderDetail(c *gin.Context) {
	orderNo := c.Param("order_no")

	var outTradeNo, appID, channel, status, notifyStatus, notifyURL string
	var amount, retryCount int
	var createdAt, updatedAt time.Time
	var paidAt, notifiedAt sql.NullTime
	var channelOrderNo, payURL sql.NullString

	err := h.db.QueryRow(`
		SELECT order_no, out_trade_no, app_id, channel, amount,
		       status, notify_status, notify_url, retry_count, channel_order_no,
		       pay_url, created_at, paid_at, notified_at, updated_at
		FROM orders
		WHERE order_no = $1
	`, orderNo).Scan(&orderNo, &outTradeNo, &appID, &channel, &amount,
		&status, &notifyStatus, &notifyURL, &retryCount, &channelOrderNo,
		&payURL, &createdAt, &paidAt, &notifiedAt, &updatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    "ERROR",
			"message": "订单不存在",
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    "ERROR",
			"message": "查询失败",
		})
		return
	}

	order := gin.H{
		"order_no":      orderNo,
		"out_trade_no":  outTradeNo,
		"app_id":        appID,
		"channel":       channel,
		"amount":        amount,
		"status":        status,
		"notify_status": notifyStatus,
		"notify_url":    notifyURL,
		"retry_count":   retryCount,
		"created_at":    createdAt,
		"updated_at":    updatedAt,
	}

	if channelOrderNo.Valid {
		order["channel_order_no"] = channelOrderNo.String
	}
	if payURL.Valid {
		order["pay_url"] = payURL.String
	}
	if paidAt.Valid {
		order["paid_at"] = paidAt.Time
	}
	if notifiedAt.Valid {
		order["notified_at"] = notifiedAt.Time
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    "SUCCESS",
		"message": "查询成功",
		"data":    order,
	})
}

// RetryOrder 重试订单通知
func (h *WebHandler) RetryOrder(c *gin.Context) {
	orderNo := c.Param("order_no")

	// 检查订单是否存在且已支付
	var status, notifyStatus string
	err := h.db.QueryRow(`
		SELECT status, notify_status FROM orders WHERE order_no = $1
	`, orderNo).Scan(&status, &notifyStatus)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    "ERROR",
			"message": "订单不存在",
		})
		return
	}

	if status != "paid" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "ERROR",
			"message": "只能重试已支付的订单",
		})
		return
	}

	// 重置通知状态
	_, err = h.db.Exec(`
		UPDATE orders
		SET retry_count = 0, notify_status = 'pending', updated_at = NOW()
		WHERE order_no = $1
	`, orderNo)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    "ERROR",
			"message": "重试失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    "SUCCESS",
		"message": "重试任务已提交",
	})
}

// BatchRetry 批量重试
func (h *WebHandler) BatchRetry(c *gin.Context) {
	var req struct {
		OrderNos []string `json:"order_nos"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "ERROR",
			"message": "参数错误",
		})
		return
	}

	success := 0
	failed := 0

	for _, orderNo := range req.OrderNos {
		_, err := h.db.Exec(`
			UPDATE orders
			SET retry_count = 0, notify_status = 'pending', updated_at = NOW()
			WHERE order_no = $1 AND status = 'paid'
		`, orderNo)

		if err != nil {
			failed++
		} else {
			success++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    "SUCCESS",
		"message": "批量重试完成",
		"data": gin.H{
			"total":   len(req.OrderNos),
			"success": success,
			"failed":  failed,
		},
	})
}

// GetReconciliationReports 获取对账报告列表
func (h *WebHandler) GetReconciliationReports(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	channel := c.Query("channel")
	status := c.Query("status")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	offset := (page - 1) * pageSize

	// 构建查询条件
	where := "WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

	if channel != "" {
		where += " AND channel = $" + strconv.Itoa(argIndex)
		args = append(args, channel)
		argIndex++
	}

	if status != "" {
		where += " AND status = $" + strconv.Itoa(argIndex)
		args = append(args, status)
		argIndex++
	}

	if startDate != "" {
		where += " AND date >= $" + strconv.Itoa(argIndex)
		args = append(args, startDate)
		argIndex++
	}

	if endDate != "" {
		where += " AND date <= $" + strconv.Itoa(argIndex)
		args = append(args, endDate)
		argIndex++
	}

	// 查询总数
	var total int
	countQuery := "SELECT COUNT(*) FROM reconciliation_reports " + where
	h.db.QueryRow(countQuery, args...).Scan(&total)

	// 查询数据
	args = append(args, pageSize, offset)
	query := `
		SELECT id, date, channel, total_orders, matched_orders,
		       long_orders, short_orders, amount_mismatch, status, created_at
		FROM reconciliation_reports ` + where + `
		ORDER BY date DESC
		LIMIT $` + strconv.Itoa(argIndex) + ` OFFSET $` + strconv.Itoa(argIndex+1)

	rows, err := h.db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    "ERROR",
			"message": "查询失败",
		})
		return
	}
	defer rows.Close()

	reports := []map[string]any{}
	for rows.Next() {
		var id int
		var date, channel, status string
		var totalOrders, matchedOrders, longOrders, shortOrders, amountMismatch int
		var createdAt time.Time

		rows.Scan(&id, &date, &channel, &totalOrders, &matchedOrders,
			&longOrders, &shortOrders, &amountMismatch, &status, &createdAt)

		report := map[string]any{
			"id":              id,
			"date":            date,
			"channel":         channel,
			"total_orders":    totalOrders,
			"matched_orders":  matchedOrders,
			"long_orders":     longOrders,
			"short_orders":    shortOrders,
			"amount_mismatch": amountMismatch,
			"status":          status,
			"created_at":      createdAt,
		}

		reports = append(reports, report)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    "SUCCESS",
		"message": "查询成功",
		"data": gin.H{
			"total": total,
			"page":  page,
			"data":  reports,
		},
	})
}

// GetReconciliationDetail 获取对账报告详情
func (h *WebHandler) GetReconciliationDetail(c *gin.Context) {
	id := c.Param("id")

	// 查询报告基本信息
	var date, channel, status string
	var totalOrders, matchedOrders, longOrders, shortOrders, amountMismatch int
	var createdAt time.Time

	err := h.db.QueryRow(`
		SELECT date, channel, total_orders, matched_orders,
		       long_orders, short_orders, amount_mismatch, status, created_at
		FROM reconciliation_reports
		WHERE id = $1
	`, id).Scan(&date, &channel, &totalOrders, &matchedOrders,
		&longOrders, &shortOrders, &amountMismatch, &status, &createdAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    "ERROR",
			"message": "报告不存在",
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    "ERROR",
			"message": "查询失败",
		})
		return
	}

	report := gin.H{
		"date":            date,
		"channel":         channel,
		"total_orders":    totalOrders,
		"matched_orders":  matchedOrders,
		"long_orders":     longOrders,
		"short_orders":    shortOrders,
		"amount_mismatch": amountMismatch,
		"status":          status,
		"created_at":      createdAt,
	}

	// 查询长款明细
	if longOrders > 0 {
		longRows, _ := h.db.Query(`
			SELECT order_no, external_amount
			FROM reconciliation_details
			WHERE report_id = $1 AND type = 'long'
			LIMIT 100
		`, id)
		defer longRows.Close()

		longDetails := []map[string]any{}
		for longRows.Next() {
			var orderNo string
			var amount int
			longRows.Scan(&orderNo, &amount)
			longDetails = append(longDetails, map[string]any{
				"order_no": orderNo,
				"amount":   amount,
			})
		}
		report["long_order_details"] = longDetails
	}

	// 查询短款明细
	if shortOrders > 0 {
		shortRows, _ := h.db.Query(`
			SELECT order_no, internal_amount
			FROM reconciliation_details
			WHERE report_id = $1 AND type = 'short'
			LIMIT 100
		`, id)
		defer shortRows.Close()

		shortDetails := []map[string]any{}
		for shortRows.Next() {
			var orderNo string
			var amount int
			shortRows.Scan(&orderNo, &amount)
			shortDetails = append(shortDetails, map[string]any{
				"order_no": orderNo,
				"amount":   amount,
			})
		}
		report["short_order_details"] = shortDetails
	}

	// 查询金额不匹配明细
	if amountMismatch > 0 {
		mismatchRows, _ := h.db.Query(`
			SELECT order_no, internal_amount, external_amount
			FROM reconciliation_details
			WHERE report_id = $1 AND type = 'mismatch'
			LIMIT 100
		`, id)
		defer mismatchRows.Close()

		mismatchDetails := []map[string]any{}
		for mismatchRows.Next() {
			var orderNo string
			var internalAmount, externalAmount int
			mismatchRows.Scan(&orderNo, &internalAmount, &externalAmount)
			mismatchDetails = append(mismatchDetails, map[string]any{
				"order_no":        orderNo,
				"internal_amount": internalAmount,
				"external_amount": externalAmount,
				"diff":            internalAmount - externalAmount,
			})
		}
		report["amount_mismatch_details"] = mismatchDetails
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    "SUCCESS",
		"message": "查询成功",
		"data":    report,
	})
}

// DownloadReport 下载对账报告
func (h *WebHandler) DownloadReport(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "ERROR",
			"message": "参数错误",
		})
		return
	}

	var filePath, channel string
	var date string
	err = h.db.QueryRow(`
		SELECT file_path, channel, date
		FROM reconciliation_reports
		WHERE id = $1
	`, id).Scan(&filePath, &channel, &date)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    "ERROR",
			"message": "报告不存在",
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    "ERROR",
			"message": "查询失败",
		})
		return
	}

	if filePath == "" {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    "ERROR",
			"message": "报告文件不存在",
		})
		return
	}

	if _, err := os.Stat(filePath); err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    "ERROR",
			"message": "报告文件已丢失",
		})
		return
	}

	filename := filepath.Base(filePath)
	if filename == "" {
		filename = "reconciliation_report.csv"
	}
	if date != "" {
		filename = channel + "_" + date + ".csv"
	}

	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.File(filePath)
}

// GetOrderStats 获取订单统计
func (h *WebHandler) GetOrderStats(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))

	// 查询最近N天的订单统计
	rows, err := h.db.Query(`
		SELECT DATE(created_at) as date,
		       COUNT(*) as total,
		       COUNT(CASE WHEN status = 'paid' THEN 1 END) as paid,
		       COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed
		FROM orders
		WHERE created_at >= NOW() - INTERVAL '1 day' * $1
		GROUP BY DATE(created_at)
		ORDER BY date DESC
	`, days)

	if err != nil {
		// 如果查询失败，返回空数据而不是错误
		c.JSON(http.StatusOK, gin.H{
			"code":    "SUCCESS",
			"message": "查询成功",
			"data":    []map[string]any{},
		})
		return
	}
	defer rows.Close()

	stats := []map[string]any{}
	for rows.Next() {
		var date string
		var total, paid, failed int
		rows.Scan(&date, &total, &paid, &failed)

		stats = append(stats, map[string]any{
			"date":   date,
			"total":  total,
			"paid":   paid,
			"failed": failed,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    "SUCCESS",
		"message": "查询成功",
		"data":    stats,
	})
}

// GetNotificationStats 获取通知统计
func (h *WebHandler) GetNotificationStats(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))

	// 查询最近N天的通知统计
	rows, err := h.db.Query(`
		SELECT DATE(created_at) as date,
		       COUNT(CASE WHEN notify_status = 'notified' THEN 1 END) as success,
		       COUNT(CASE WHEN notify_status = 'failed' THEN 1 END) as failed,
		       COUNT(CASE WHEN notify_status = 'pending' THEN 1 END) as pending
		FROM orders
		WHERE created_at >= NOW() - INTERVAL '1 day' * $1
		  AND status = 'paid'
		GROUP BY DATE(created_at)
		ORDER BY date DESC
	`, days)

	if err != nil {
		// 如果查询失败，返回空数据而不是错误
		c.JSON(http.StatusOK, gin.H{
			"code":    "SUCCESS",
			"message": "查询成功",
			"data":    []map[string]any{},
		})
		return
	}
	defer rows.Close()

	stats := []map[string]any{}
	for rows.Next() {
		var date string
		var success, failed, pending int
		rows.Scan(&date, &success, &failed, &pending)

		stats = append(stats, map[string]any{
			"date":    date,
			"success": success,
			"failed":  failed,
			"pending": pending,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    "SUCCESS",
		"message": "查询成功",
		"data":    stats,
	})
}
