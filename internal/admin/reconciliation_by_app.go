package admin

import (
	"net/http"
	"strconv"
	"time"

	"gopay/internal/reconciliation"

	"github.com/gin-gonic/gin"
)

// GetReconciliationByApp 获取指定应用的对账报告
func (h *WebHandler) GetReconciliationByApp(c *gin.Context) {
	appID := c.Query("app_id")
	if appID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "INVALID_PARAMS",
			"message": "app_id is required",
		})
		return
	}

	// 解析日期范围
	startDateStr := c.DefaultQuery("start_date", time.Now().AddDate(0, 0, -30).Format("2006-01-02"))
	endDateStr := c.DefaultQuery("end_date", time.Now().Format("2006-01-02"))

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "INVALID_PARAMS",
			"message": "invalid start_date format",
		})
		return
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "INVALID_PARAMS",
			"message": "invalid end_date format",
		})
		return
	}

	// 分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	// 创建对账服务
	reconService := reconciliation.NewReconciliationService(h.db)

	// 查询报告列表
	reports, total, err := reconService.GetReportsByApp(c.Request.Context(), appID, startDate, endDate, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    "INTERNAL_ERROR",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    "SUCCESS",
		"message": "查询成功",
		"data": gin.H{
			"reports":   reports,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// GetAppReconciliationStats 获取应用对账统计
func (h *WebHandler) GetAppReconciliationStats(c *gin.Context) {
	appID := c.Query("app_id")
	if appID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "INVALID_PARAMS",
			"message": "app_id is required",
		})
		return
	}

	// 解析日期范围
	startDateStr := c.DefaultQuery("start_date", time.Now().AddDate(0, 0, -30).Format("2006-01-02"))
	endDateStr := c.DefaultQuery("end_date", time.Now().Format("2006-01-02"))

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "INVALID_PARAMS",
			"message": "invalid start_date format",
		})
		return
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "INVALID_PARAMS",
			"message": "invalid end_date format",
		})
		return
	}

	// 创建对账服务
	reconService := reconciliation.NewReconciliationService(h.db)

	// 查询统计数据
	stats, err := reconService.GetAppReconciliationStats(c.Request.Context(), appID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    "INTERNAL_ERROR",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    "SUCCESS",
		"message": "查询成功",
		"data":    stats,
	})
}

// GetAllAppsReconciliationSummary 获取所有应用的对账汇总
func (h *WebHandler) GetAllAppsReconciliationSummary(c *gin.Context) {
	// 解析日期
	dateStr := c.DefaultQuery("date", time.Now().AddDate(0, 0, -1).Format("2006-01-02"))
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "INVALID_PARAMS",
			"message": "invalid date format",
		})
		return
	}

	// 创建对账服务
	reconService := reconciliation.NewReconciliationService(h.db)

	// 查询汇总数据
	summaries, err := reconService.GetAllAppsReconciliationSummary(c.Request.Context(), date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    "INTERNAL_ERROR",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    "SUCCESS",
		"message": "查询成功",
		"data":    summaries,
	})
}

// GetAppList 获取应用列表（用于下拉选择）
func (h *WebHandler) GetAppList(c *gin.Context) {
	rows, err := h.db.Query(`
		SELECT app_id, app_name, status
		FROM apps
		WHERE status = 'active'
		ORDER BY app_name
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    "INTERNAL_ERROR",
			"message": err.Error(),
		})
		return
	}
	defer rows.Close()

	type AppInfo struct {
		AppID   string `json:"app_id"`
		AppName string `json:"app_name"`
		Status  string `json:"status"`
	}

	var apps []AppInfo
	for rows.Next() {
		var app AppInfo
		if err := rows.Scan(&app.AppID, &app.AppName, &app.Status); err != nil {
			continue
		}
		apps = append(apps, app)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    "SUCCESS",
		"message": "查询成功",
		"data":    apps,
	})
}
