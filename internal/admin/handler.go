package admin

import (
	"html/template"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// AdminHandler 管理后台处理器
type AdminHandler struct {
	orderService  OrderService
	notifyService NotifyService
}

// NewAdminHandler 创建管理后台处理器
func NewAdminHandler(orderService OrderService, notifyService NotifyService) *AdminHandler {
	return &AdminHandler{
		orderService:  orderService,
		notifyService: notifyService,
	}
}

// RegisterRoutes 注册路由
func (h *AdminHandler) RegisterRoutes(r *gin.Engine) {
	admin := r.Group("/admin")
	{
		// 首页
		admin.GET("/", h.Index)

		// 失败订单
		admin.GET("/failed-orders", h.FailedOrders)
		admin.POST("/retry-order/:order_no", h.RetryOrder)
		admin.POST("/batch-retry", h.BatchRetry)

		// 回调失败
		admin.GET("/failed-webhooks", h.FailedWebhooks)
		admin.POST("/retry-webhook/:order_no", h.RetryWebhook)

		// 对账报告
		admin.GET("/reconciliation", h.ReconciliationReports)
		admin.GET("/reconciliation/:id", h.ReconciliationDetail)

		// 操作日志
		admin.GET("/logs", h.OperationLogs)
	}
}

// Index 首页
func (h *AdminHandler) Index(c *gin.Context) {
	stats := h.getStatistics()
	c.HTML(http.StatusOK, "index.html", gin.H{
		"stats": stats,
	})
}

// FailedOrders 失败订单列表
func (h *AdminHandler) FailedOrders(c *gin.Context) {
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")
	channel := c.Query("channel")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	orders, total, err := h.orderService.GetFailedOrders(c.Request.Context(), FailedOrdersQuery{
		Page:      parseInt(page),
		PageSize:  parseInt(pageSize),
		Channel:   channel,
		StartDate: parseDate(startDate),
		EndDate:   parseDate(endDate),
	})

	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": err.Error(),
		})
		return
	}

	c.HTML(http.StatusOK, "failed_orders.html", gin.H{
		"orders": orders,
		"total":  total,
		"page":   page,
	})
}

// RetryOrder 重试订单
func (h *AdminHandler) RetryOrder(c *gin.Context) {
	orderNo := c.Param("order_no")

	err := h.orderService.RetryOrder(c.Request.Context(), orderNo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 记录操作日志
	h.logOperation(c, "retry_order", orderNo)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "订单重试成功",
	})
}

// BatchRetry 批量重试
func (h *AdminHandler) BatchRetry(c *gin.Context) {
	var req struct {
		OrderNos []string `json:"order_nos"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "参数错误",
		})
		return
	}

	successCount := 0
	failedCount := 0

	for _, orderNo := range req.OrderNos {
		err := h.orderService.RetryOrder(c.Request.Context(), orderNo)
		if err != nil {
			failedCount++
		} else {
			successCount++
		}
	}

	// 记录操作日志
	h.logOperation(c, "batch_retry", "")

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"success_count": successCount,
		"failed_count":  failedCount,
	})
}

// FailedWebhooks 回调失败列表
func (h *AdminHandler) FailedWebhooks(c *gin.Context) {
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "20")

	webhooks, total, err := h.notifyService.GetFailedWebhooks(c.Request.Context(), FailedWebhooksQuery{
		Page:     parseInt(page),
		PageSize: parseInt(pageSize),
	})

	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": err.Error(),
		})
		return
	}

	c.HTML(http.StatusOK, "failed_webhooks.html", gin.H{
		"webhooks": webhooks,
		"total":    total,
		"page":     page,
	})
}

// RetryWebhook 重试回调
func (h *AdminHandler) RetryWebhook(c *gin.Context) {
	orderNo := c.Param("order_no")

	err := h.notifyService.RetryWebhook(c.Request.Context(), orderNo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 记录操作日志
	h.logOperation(c, "retry_webhook", orderNo)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "回调重试成功",
	})
}

// ReconciliationReports 对账报告列表
func (h *AdminHandler) ReconciliationReports(c *gin.Context) {
	// TODO: 实现对账报告列表
	c.HTML(http.StatusOK, "reconciliation.html", gin.H{})
}

// ReconciliationDetail 对账报告详情
func (h *AdminHandler) ReconciliationDetail(c *gin.Context) {
	// TODO: 实现对账报告详情
	c.HTML(http.StatusOK, "reconciliation_detail.html", gin.H{})
}

// OperationLogs 操作日志
func (h *AdminHandler) OperationLogs(c *gin.Context) {
	// TODO: 实现操作日志
	c.HTML(http.StatusOK, "logs.html", gin.H{})
}

// getStatistics 获取统计数据
func (h *AdminHandler) getStatistics() Statistics {
	// TODO: 实现统计数据获取
	return Statistics{
		TotalOrders:    1000,
		FailedOrders:   10,
		FailedWebhooks: 5,
		TodayOrders:    100,
	}
}

// logOperation 记录操作日志
func (h *AdminHandler) logOperation(c *gin.Context, action, target string) {
	// TODO: 实现操作日志记录
}

// 辅助函数
func parseInt(s string) int {
	// TODO: 实现字符串转整数
	return 1
}

func parseDate(s string) time.Time {
	// TODO: 实现字符串转日期
	return time.Now()
}

// 数据结构
type FailedOrdersQuery struct {
	Page      int
	PageSize  int
	Channel   string
	StartDate time.Time
	EndDate   time.Time
}

type FailedWebhooksQuery struct {
	Page     int
	PageSize int
}

type Statistics struct {
	TotalOrders    int
	FailedOrders   int
	FailedWebhooks int
	TodayOrders    int
}

// 服务接口
type OrderService interface {
	GetFailedOrders(ctx interface{}, query FailedOrdersQuery) ([]interface{}, int, error)
	RetryOrder(ctx interface{}, orderNo string) error
}

type NotifyService interface {
	GetFailedWebhooks(ctx interface{}, query FailedWebhooksQuery) ([]interface{}, int, error)
	RetryWebhook(ctx interface{}, orderNo string) error
}
