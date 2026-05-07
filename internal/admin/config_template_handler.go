package admin

import (
	"net/http"
	"strconv"

	"gopay/internal/service"

	"github.com/gin-gonic/gin"
)

// GetConfigTemplates 获取配置模板列表
func (h *WebHandler) GetConfigTemplates(c *gin.Context) {
	channel := c.Query("channel")

	templateService := service.NewConfigTemplateService(h.db)
	templates, err := templateService.GetTemplateList(c.Request.Context(), channel)
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
		"data":    templates,
	})
}

// GetConfigTemplate 获取单个配置模板
func (h *WebHandler) GetConfigTemplate(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "INVALID_PARAMS",
			"message": "invalid template id",
		})
		return
	}

	templateService := service.NewConfigTemplateService(h.db)
	template, err := templateService.GetTemplateByID(c.Request.Context(), id)
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
		"data":    template,
	})
}

// GetConfigTemplateByChannel 根据渠道获取配置模板
func (h *WebHandler) GetConfigTemplateByChannel(c *gin.Context) {
	channel := c.Param("channel")
	if channel == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "INVALID_PARAMS",
			"message": "channel is required",
		})
		return
	}

	templateService := service.NewConfigTemplateService(h.db)
	template, err := templateService.GetTemplateByChannel(c.Request.Context(), channel)
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
		"data":    template,
	})
}

// QuickSetupApp 快速设置应用
func (h *WebHandler) QuickSetupApp(c *gin.Context) {
	var req service.QuickSetupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "INVALID_PARAMS",
			"message": err.Error(),
		})
		return
	}

	templateService := service.NewConfigTemplateService(h.db)
	if err := templateService.QuickSetupApp(c.Request.Context(), &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    "INTERNAL_ERROR",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    "SUCCESS",
		"message": "应用设置成功",
	})
}

// GetChannelList 获取支持的渠道列表
func (h *WebHandler) GetChannelList(c *gin.Context) {
	channels := []map[string]interface{}{
		{
			"channel":     "wechat_native",
			"name":        "微信 Native 扫码",
			"description": "适用于 PC 网站",
			"icon":        "wechat",
		},
		{
			"channel":     "wechat_jsapi",
			"name":        "微信 JSAPI",
			"description": "适用于公众号/小程序",
			"icon":        "wechat",
		},
		{
			"channel":     "wechat_h5",
			"name":        "微信 H5",
			"description": "适用于手机浏览器",
			"icon":        "wechat",
		},
		{
			"channel":     "wechat_app",
			"name":        "微信 APP",
			"description": "适用于原生应用",
			"icon":        "wechat",
		},
		{
			"channel":     "alipay_qr",
			"name":        "支付宝扫码",
			"description": "适用于 PC 网站",
			"icon":        "alipay",
		},
		{
			"channel":     "alipay_wap",
			"name":        "支付宝手机网站",
			"description": "适用于手机浏览器",
			"icon":        "alipay",
		},
		{
			"channel":     "alipay_app",
			"name":        "支付宝 APP",
			"description": "适用于原生应用",
			"icon":        "alipay",
		},
		{
			"channel":     "alipay_face",
			"name":        "支付宝当面付",
			"description": "适用于线下收银",
			"icon":        "alipay",
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    "SUCCESS",
		"message": "查询成功",
		"data":    channels,
	})
}
