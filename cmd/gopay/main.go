package main

import (
	"context"
	"log"
	"os"
	"strings"

	"gopay/internal/admin"
	"gopay/internal/config"
	"gopay/internal/database"
	"gopay/internal/handler"
	_ "gopay/internal/metrics"
	"gopay/internal/service"
	"gopay/pkg/alert"
	"gopay/pkg/logger"
	"gopay/pkg/middleware"
	"gopay/pkg/security"
	"gopay/pkg/version"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// collectCertPaths 收集需要检查的证书文件路径
func collectCertPaths() []string {
	var paths []string
	envKeys := []string{"WECHAT_CERT_PATH", "WECHAT_PLATFORM_CERT_PATH"}
	for _, key := range envKeys {
		if p := os.Getenv(key); p != "" {
			if _, err := os.Stat(p); err == nil {
				paths = append(paths, p)
			}
		}
	}
	return paths
}

func main() {
	// 加载环境变量
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	// 初始化配置
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load config: %v", err)
	}

	// 初始化日志
	logger.Init(cfg.LogLevel, cfg.LogFile)
	logger.Info("Starting GoPay server...")

	// 初始化数据库
	if err := database.Connect(cfg.Database); err != nil {
		logger.Fatal("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// 运行数据库迁移
	db := database.GetDB()
	if err := database.RunMigrations(db, "migrations"); err != nil {
		logger.Fatal("Failed to run migrations: %v", err)
	}

	// 初始化服务层
	channelManager := service.NewChannelManager(db)
	orderService := service.NewOrderService(db, channelManager)
	orderService.SetPublicBaseURL(cfg.PublicBaseURL)
	notifyService := service.NewNotifyService(db, orderService)
	if cfg.AlertWebhookURL != "" {
		notifyService.SetAlertManager(alert.NewAlertManager(cfg.AlertWebhookURL))
	}
	refundService := service.NewRefundService(db, orderService, channelManager)

	// 初始化 Handler
	handler.InitServices(orderService)
	handler.InitRefundService(refundService)
	handler.InitWebhookServices(channelManager, notifyService)

	// 设置 Gin 模式
	if cfg.ServerEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 启动证书有效期检查（后台定期检查）
	certPaths := collectCertPaths()
	if len(certPaths) > 0 {
		var alertFn func(context.Context, string) error
		if cfg.AlertWebhookURL != "" {
			am := alert.NewAlertManager(cfg.AlertWebhookURL)
			alertFn = func(ctx context.Context, msg string) error {
				return am.SendAlert(&alert.AlertMessage{
					Level:   alert.AlertLevelWarning,
					Title:   "证书过期告警",
					Content: msg,
				})
			}
		}
		certChecker := security.NewCertChecker(certPaths, 30, alertFn)
		go certChecker.StartPeriodicCheck(context.Background())
		logger.Info("Certificate checker started for %d cert(s)", len(certPaths))
	}

	// 创建路由
	router := gin.Default()
	router.Use(middleware.RequestID())
	router.Use(middleware.TraceContext())
	router.Use(middleware.PrometheusMetrics())
	router.Use(middleware.LocalRateLimit(middleware.LocalRateLimitConfig{
		Rate:  100,
		Burst: 200,
	}))

	// 健康检查
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"code":    "SUCCESS",
			"message": "服务正常",
			"data": gin.H{
				"status":  "healthy",
				"service": "gopay",
				"version": version.Version,
			},
		})
	})

	// 详细健康检查
	router.GET("/health/detail", func(c *gin.Context) {
		// 检查数据库连接
		dbStatus := "healthy"
		if err := db.Ping(); err != nil {
			dbStatus = "unhealthy"
		}

		c.JSON(200, gin.H{
			"code":    "SUCCESS",
			"message": "健康检查",
			"data": gin.H{
				"status":   "healthy",
				"service":  "gopay",
				"version":  version.Version,
				"database": dbStatus,
			},
		})
	})

	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API 路由组
	api := router.Group("/api/v1")

	// 签名验证中间件（保护业务接口）
	nonceChecker := middleware.NewInMemoryNonceChecker()
	signedAPI := api.Group("", middleware.SignatureAuth(db, nonceChecker))
	{
		// 支付相关（需要签名验证）
		signedAPI.POST("/checkout", handler.Checkout)
		signedAPI.GET("/orders/:order_no", handler.QueryOrder)
	}

	// Webhook 回调（不需要签名验证，由各支付平台自身的签名机制保护）
	{
		api.POST("/webhook/wechat", handler.WechatWebhook)
		api.POST("/webhook/alipay", handler.AlipayWebhook)
		api.POST("/webhook/stripe", handler.StripeWebhook)
	}

	// 内部管理接口（需要认证）
	internal := router.Group("/internal/api/v1")
	{
		// 配置认证中间件
		authConfig := middleware.NewAuthConfig()
		authConfig.AddAPIKey(cfg.AdminAPIKey)

		// 应用认证中间件
		internal.Use(middleware.APIKeyAuth(authConfig))

		// 查询失败订单
		internal.GET("/orders/failed", handler.ListFailedOrders)

		// 手动重试通知
		internal.POST("/orders/:order_no/retry", handler.RetryNotify)
		internal.POST("/orders/:order_no/refund", handler.RefundOrder)
		internal.GET("/orders/:order_no/refunds/:refund_no", handler.QueryRefund)
	}

	// 配置管理后台认证
	adminAuthConfig := middleware.NewAuthConfig()
	adminAuthConfig.AddAPIKey(cfg.AdminAPIKey)

	// 构建认证中间件列表
	var adminMiddlewares []gin.HandlerFunc

	// 可选：添加 IP 白名单
	ipWhitelist := cfg.AdminIPWhitelist
	if ipWhitelist != "" {
		// 支持逗号分隔的 IP 列表
		ips := strings.Split(ipWhitelist, ",")
		for _, ip := range ips {
			adminAuthConfig.AddIPWhitelist(strings.TrimSpace(ip))
		}
		adminMiddlewares = append(adminMiddlewares, middleware.IPWhitelist(adminAuthConfig))
	}

	// 添加 API Key 认证中间件
	adminMiddlewares = append(adminMiddlewares, middleware.APIKeyAuth(adminAuthConfig))

	// 管理后台（传入认证中间件）
	webHandler := admin.NewWebHandler(db)
	webHandler.RegisterRoutesWithAuth(router, adminMiddlewares...)

	// 配置管理（传入认证中间件）
	configHandler := admin.NewConfigHandler(db)
	configHandler.RegisterRoutesWithAuth(router, adminMiddlewares...)

	// 启动服务
	addr := ":" + cfg.ServerPort
	logger.Info("Server listening on %s", addr)
	if err := router.Run(addr); err != nil {
		logger.Fatal("Failed to start server: %v", err)
	}
}
