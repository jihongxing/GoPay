package admin

import (
	"fmt"
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
	page := c.DefaultQuery("page", "1")
	_ = c.DefaultQuery("page_size", "20") // pageSize for future use
	channel := c.Query("channel")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	// 查询对账报告列表
	// 实现步骤：
	// 1. 从数据库查询对账记录
	// 2. 支持按渠道、日期范围筛选
	// 3. 分页返回结果
	//
	// 示例代码：
	// reports, total, err := h.reconciliationService.GetReports(c.Request.Context(), ReconciliationQuery{
	//     Page:      parseInt(page),
	//     PageSize:  parseInt(pageSize),
	//     Channel:   channel,
	//     StartDate: parseDate(startDate),
	//     EndDate:   parseDate(endDate),
	// })

	c.HTML(http.StatusOK, "reconciliation.html", gin.H{
		"page":       page,
		"channel":    channel,
		"start_date": startDate,
		"end_date":   endDate,
	})
}

// ReconciliationDetail 对账报告详情
func (h *AdminHandler) ReconciliationDetail(c *gin.Context) {
	id := c.Param("id")

	// 查询对账报告详情
	// 实现步骤：
	// 1. 根据 ID 查询对账记录
	// 2. 获取长款、短款、金额不匹配明细
	// 3. 渲染详情页面
	//
	// 示例代码：
	// report, err := h.reconciliationService.GetReportByID(c.Request.Context(), id)
	// if err != nil {
	//     c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error()})
	//     return
	// }

	c.HTML(http.StatusOK, "reconciliation_detail.html", gin.H{
		"id": id,
	})
}

// OperationLogs 操作日志
func (h *AdminHandler) OperationLogs(c *gin.Context) {
	page := c.DefaultQuery("page", "1")
	_ = c.DefaultQuery("page_size", "20") // pageSize for future use
	action := c.Query("action")
	operator := c.Query("operator")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	// 查询操作日志
	// 实现步骤：
	// 1. 从数据库查询操作日志
	// 2. 支持按操作类型、操作人、日期范围筛选
	// 3. 分页返回结果
	//
	// 示例代码：
	// logs, total, err := h.auditService.GetLogs(c.Request.Context(), AuditLogQuery{
	//     Page:      parseInt(page),
	//     PageSize:  parseInt(pageSize),
	//     Action:    action,
	//     Operator:  operator,
	//     StartDate: parseDate(startDate),
	//     EndDate:   parseDate(endDate),
	// })

	c.HTML(http.StatusOK, "logs.html", gin.H{
		"page":       page,
		"action":     action,
		"operator":   operator,
		"start_date": startDate,
		"end_date":   endDate,
	})
}

// getStatistics 获取统计数据
func (h *AdminHandler) getStatistics() Statistics {
	// 实现统计数据获取
	// 实现步骤：
	// 1. 查询总订单数
	// 2. 查询失败订单数
	// 3. 查询失败回调数
	// 4. 查询今日订单数
	//
	// 示例代码：
	// ctx := context.Background()
	// totalOrders, _ := h.orderService.CountOrders(ctx, OrderCountQuery{})
	// failedOrders, _ := h.orderService.CountOrders(ctx, OrderCountQuery{Status: "failed"})
	// failedWebhooks, _ := h.notifyService.CountFailedWebhooks(ctx)
	// todayOrders, _ := h.orderService.CountOrders(ctx, OrderCountQuery{
	//     StartDate: time.Now().Truncate(24 * time.Hour),
	// })
	//
	// return Statistics{
	//     TotalOrders:    totalOrders,
	//     FailedOrders:   failedOrders,
	//     FailedWebhooks: failedWebhooks,
	//     TodayOrders:    todayOrders,
	// }

	return Statistics{
		TotalOrders:    0,
		FailedOrders:   0,
		FailedWebhooks: 0,
		TodayOrders:    0,
	}
}

// logOperation 记录操作日志
func (h *AdminHandler) logOperation(c *gin.Context, action, target string) {
	// 实现操作日志记录
	// 实现步骤：
	// 1. 获取操作人信息（从 session 或 JWT）
	// 2. 记录操作类型、目标、IP、时间等
	// 3. 写入数据库或日志文件
	//
	// 示例代码：
	// import "gopay/pkg/audit"
	//
	// operator := c.GetString("user_id") // 从认证中间件获取
	// ip := c.ClientIP()
	//
	// auditLog := audit.AuditLog{
	//     Operator:  operator,
	//     Action:    action,
	//     Target:    target,
	//     IP:        ip,
	//     UserAgent: c.Request.UserAgent(),
	//     CreatedAt: time.Now(),
	// }
	//
	// audit.GetAuditLogger().Log(c.Request.Context(), auditLog)
}

// 辅助函数
func parseInt(s string) int {
	// 实现字符串转整数
	var result int
	if _, err := fmt.Sscanf(s, "%d", &result); err != nil {
		return 1
	}
	return result
}

func parseDate(s string) time.Time {
	// 实现字符串转日期
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}
	}
	return t
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
	GetFailedOrders(ctx any, query FailedOrdersQuery) ([]any, int, error)
	RetryOrder(ctx any, orderNo string) error
}

type NotifyService interface {
	GetFailedWebhooks(ctx any, query FailedWebhooksQuery) ([]any, int, error)
	RetryWebhook(ctx any, orderNo string) error
}
