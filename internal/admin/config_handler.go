package admin

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gopay/internal/models"
	"gopay/internal/service"
)

// ConfigHandler 配置管理处理器
type ConfigHandler struct {
	configService *service.ConfigService
}

// NewConfigHandler 创建配置管理处理器
func NewConfigHandler(db *sql.DB) *ConfigHandler {
	return &ConfigHandler{
		configService: service.NewConfigService(db),
	}
}

// RegisterRoutesWithAuth 注册路由（带认证）
func (h *ConfigHandler) RegisterRoutesWithAuth(r *gin.Engine, authMiddleware ...gin.HandlerFunc) {
	admin := r.Group("/admin")
	// 应用认证中间件
	admin.Use(authMiddleware...)
	{
		// 页面路由
		admin.GET("/apps", h.AppsPage)
		admin.GET("/apps/:app_id/channels", h.ChannelsPage)
		admin.GET("/config-logs", h.ConfigLogsPage)

		// API 路由
		api := admin.Group("/api")
		{
			// App 管理
			api.GET("/apps", h.ListApps)
			api.GET("/apps/:app_id", h.GetApp)
			api.POST("/apps", h.CreateApp)
			api.PUT("/apps/:app_id", h.UpdateApp)
			api.DELETE("/apps/:app_id", h.DeleteApp)

			// 渠道配置管理
			api.GET("/apps/:app_id/channels", h.ListChannelConfigs)
			api.GET("/channels/:id", h.GetChannelConfig)
			api.POST("/apps/:app_id/channels", h.CreateChannelConfig)
			api.PUT("/channels/:id", h.UpdateChannelConfig)
			api.DELETE("/channels/:id", h.DeleteChannelConfig)

			// 审计日志
			api.GET("/config-logs", h.ListAuditLogs)
		}
	}
}

// RegisterRoutes 注册路由（无认证，向后兼容）
func (h *ConfigHandler) RegisterRoutes(r *gin.Engine) {
	h.RegisterRoutesWithAuth(r)
}

// ========== 页面路由 ==========

// AppsPage 应用管理页面
func (h *ConfigHandler) AppsPage(c *gin.Context) {
	c.HTML(http.StatusOK, "apps.html", gin.H{
		"Title": "应用管理",
	})
}

// ChannelsPage 渠道配置页面
func (h *ConfigHandler) ChannelsPage(c *gin.Context) {
	appID := c.Param("app_id")
	c.HTML(http.StatusOK, "channels.html", gin.H{
		"Title": "渠道配置",
		"AppID": appID,
	})
}

// ConfigLogsPage 配置日志页面
func (h *ConfigHandler) ConfigLogsPage(c *gin.Context) {
	c.HTML(http.StatusOK, "config_logs.html", gin.H{
		"Title": "配置变更日志",
	})
}

// ========== App 管理 API ==========

// ListApps 获取应用列表
func (h *ConfigHandler) ListApps(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.Query("status")

	apps, total, err := h.configService.ListApps(c.Request.Context(), page, pageSize, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    "ERROR",
			"message": "查询失败: " + err.Error(),
		})
		return
	}

	// 隐藏 app_secret（只显示前4位和后4位）
	for i := range apps {
		if len(apps[i].AppSecret) > 8 {
			apps[i].AppSecret = apps[i].AppSecret[:4] + "****" + apps[i].AppSecret[len(apps[i].AppSecret)-4:]
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    "SUCCESS",
		"message": "查询成功",
		"data": gin.H{
			"total":     total,
			"page":      page,
			"page_size": pageSize,
			"apps":      apps,
		},
	})
}

// GetApp 获取应用详情
func (h *ConfigHandler) GetApp(c *gin.Context) {
	appID := c.Param("app_id")

	app, err := h.configService.GetApp(c.Request.Context(), appID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    "ERROR",
			"message": err.Error(),
		})
		return
	}

	// 隐藏 app_secret
	if len(app.AppSecret) > 8 {
		app.AppSecret = app.AppSecret[:4] + "****" + app.AppSecret[len(app.AppSecret)-4:]
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    "SUCCESS",
		"message": "查询成功",
		"data":    app,
	})
}

// CreateApp 创建应用
func (h *ConfigHandler) CreateApp(c *gin.Context) {
	var req struct {
		AppID       string `json:"app_id" binding:"required"`
		AppName     string `json:"app_name" binding:"required"`
		AppSecret   string `json:"app_secret" binding:"required"`
		CallbackURL string `json:"callback_url" binding:"required"`
		Status      string `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "ERROR",
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	// 默认状态为 active
	if req.Status == "" {
		req.Status = "active"
	}

	app := &models.App{
		AppID:       req.AppID,
		AppName:     req.AppName,
		AppSecret:   req.AppSecret,
		CallbackURL: req.CallbackURL,
		Status:      req.Status,
	}

	// 获取操作人信息
	operator := c.GetString("operator")
	if operator == "" {
		operator = "admin" // 默认操作人
	}
	ip := c.ClientIP()
	userAgent := c.Request.UserAgent()

	err := h.configService.CreateApp(c.Request.Context(), app, operator, ip, userAgent)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    "ERROR",
			"message": "创建失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    "SUCCESS",
		"message": "创建成功",
		"data":    app,
	})
}

// UpdateApp 更新应用
func (h *ConfigHandler) UpdateApp(c *gin.Context) {
	appID := c.Param("app_id")

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "ERROR",
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	// 只允许更新特定字段
	allowedFields := map[string]bool{
		"app_name":     true,
		"app_secret":   true,
		"callback_url": true,
		"status":       true,
	}

	updates := make(map[string]interface{})
	for key, value := range req {
		if allowedFields[key] {
			updates[key] = value
		}
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "ERROR",
			"message": "没有可更新的字段",
		})
		return
	}

	// 获取操作人信息
	operator := c.GetString("operator")
	if operator == "" {
		operator = "admin"
	}
	ip := c.ClientIP()
	userAgent := c.Request.UserAgent()

	err := h.configService.UpdateApp(c.Request.Context(), appID, updates, operator, ip, userAgent)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    "ERROR",
			"message": "更新失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    "SUCCESS",
		"message": "更新成功",
	})
}

// DeleteApp 删除应用
func (h *ConfigHandler) DeleteApp(c *gin.Context) {
	appID := c.Param("app_id")

	// 获取操作人信息
	operator := c.GetString("operator")
	if operator == "" {
		operator = "admin"
	}
	ip := c.ClientIP()
	userAgent := c.Request.UserAgent()

	err := h.configService.DeleteApp(c.Request.Context(), appID, operator, ip, userAgent)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    "ERROR",
			"message": "删除失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    "SUCCESS",
		"message": "删除成功",
	})
}

// ========== 渠道配置管理 API ==========

// ListChannelConfigs 获取渠道配置列表
func (h *ConfigHandler) ListChannelConfigs(c *gin.Context) {
	appID := c.Param("app_id")

	configs, err := h.configService.ListChannelConfigs(c.Request.Context(), appID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    "ERROR",
			"message": "查询失败: " + err.Error(),
		})
		return
	}

	// 隐藏敏感配置信息（密钥等）
	for i := range configs {
		configs[i].Config = service.MaskSensitiveConfigJSON(configs[i].Config)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    "SUCCESS",
		"message": "查询成功",
		"data":    configs,
	})
}

// GetChannelConfig 获取渠道配置详情
func (h *ConfigHandler) GetChannelConfig(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	config, err := h.configService.GetChannelConfig(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    "ERROR",
			"message": err.Error(),
		})
		return
	}

	config.Config = service.MaskSensitiveConfigJSON(config.Config)

	c.JSON(http.StatusOK, gin.H{
		"code":    "SUCCESS",
		"message": "查询成功",
		"data":    config,
	})
}

// CreateChannelConfig 创建渠道配置
func (h *ConfigHandler) CreateChannelConfig(c *gin.Context) {
	appID := c.Param("app_id")

	var req struct {
		Channel string                 `json:"channel" binding:"required"`
		Config  map[string]interface{} `json:"config" binding:"required"`
		Status  string                 `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "ERROR",
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	// 默认状态为 active
	if req.Status == "" {
		req.Status = "active"
	}

	// 将 config 转为 JSON 字符串
	configJSON, err := json.Marshal(req.Config)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "ERROR",
			"message": "配置格式错误",
		})
		return
	}

	config := &models.ChannelConfig{
		AppID:   appID,
		Channel: req.Channel,
		Config:  string(configJSON),
		Status:  req.Status,
	}

	// 获取操作人信息
	operator := c.GetString("operator")
	if operator == "" {
		operator = "admin"
	}
	ip := c.ClientIP()
	userAgent := c.Request.UserAgent()

	err = h.configService.CreateChannelConfig(c.Request.Context(), config, operator, ip, userAgent)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    "ERROR",
			"message": "创建失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    "SUCCESS",
		"message": "创建成功",
		"data": &models.ChannelConfig{
			ID:        config.ID,
			AppID:     config.AppID,
			Channel:   config.Channel,
			Config:    service.MaskSensitiveConfigJSON(config.Config),
			Status:    config.Status,
			CreatedAt: config.CreatedAt,
			UpdatedAt: config.UpdatedAt,
		},
	})
}

// UpdateChannelConfig 更新渠道配置
func (h *ConfigHandler) UpdateChannelConfig(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "ERROR",
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	// 只允许更新特定字段
	allowedFields := map[string]bool{
		"config": true,
		"status": true,
	}

	updates := make(map[string]interface{})
	for key, value := range req {
		if allowedFields[key] {
			// 如果是 config 字段，转为 JSON 字符串
			if key == "config" {
				configJSON, err := json.Marshal(value)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{
						"code":    "ERROR",
						"message": "配置格式错误",
					})
					return
				}
				updates[key] = string(configJSON)
			} else {
				updates[key] = value
			}
		}
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "ERROR",
			"message": "没有可更新的字段",
		})
		return
	}

	// 获取操作人信息
	operator := c.GetString("operator")
	if operator == "" {
		operator = "admin"
	}
	ip := c.ClientIP()
	userAgent := c.Request.UserAgent()

	err := h.configService.UpdateChannelConfig(c.Request.Context(), id, updates, operator, ip, userAgent)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    "ERROR",
			"message": "更新失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    "SUCCESS",
		"message": "更新成功",
	})
}

// DeleteChannelConfig 删除渠道配置
func (h *ConfigHandler) DeleteChannelConfig(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	// 获取操作人信息
	operator := c.GetString("operator")
	if operator == "" {
		operator = "admin"
	}
	ip := c.ClientIP()
	userAgent := c.Request.UserAgent()

	err := h.configService.DeleteChannelConfig(c.Request.Context(), id, operator, ip, userAgent)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    "ERROR",
			"message": "删除失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    "SUCCESS",
		"message": "删除成功",
	})
}

// ========== 审计日志 API ==========

// ListAuditLogs 获取审计日志列表
func (h *ConfigHandler) ListAuditLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	resourceType := c.Query("resource_type")
	resourceID := c.Query("resource_id")
	operator := c.Query("operator")
	startDate := parseDateString(c.Query("start_date"))
	endDate := parseDateString(c.Query("end_date"))

	logs, total, err := h.configService.ListAuditLogs(c.Request.Context(), page, pageSize, resourceType, resourceID, operator, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    "ERROR",
			"message": "查询失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    "SUCCESS",
		"message": "查询成功",
		"data": gin.H{
			"total":     total,
			"page":      page,
			"page_size": pageSize,
			"logs":      logs,
		},
	})
}

// 辅助函数
func parseDateString(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}
	}
	return t
}
