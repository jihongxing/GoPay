package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gopay/internal/config"
	"gopay/internal/database"
	"gopay/internal/handler"
	"gopay/internal/service"
	"gopay/pkg/logger"
)

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
	notifyService := service.NewNotifyService(db, orderService)

	// 初始化 Handler
	handler.InitServices(orderService)
	handler.InitWebhookServices(channelManager, notifyService)

	// 设置 Gin 模式
	if cfg.ServerEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建路由
	router := gin.Default()

	// 健康检查
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"code":    "SUCCESS",
			"message": "服务正常",
			"data": gin.H{
				"status":  "healthy",
				"service": "gopay",
				"version": "1.0.0",
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
				"version":  "1.0.0",
				"database": dbStatus,
			},
		})
	})

	// API 路由组
	api := router.Group("/api/v1")
	{
		// 支付相关
		api.POST("/checkout", handler.Checkout)
		api.GET("/orders/:order_no", handler.QueryOrder)

		// Webhook 回调
		api.POST("/webhook/wechat", handler.WechatWebhook)
		api.POST("/webhook/alipay", handler.AlipayWebhook)
	}

	// 内部管理接口
	internal := router.Group("/internal/api/v1")
	{
		// 查询失败订单
		internal.GET("/orders/failed", handler.ListFailedOrders)

		// 手动重试通知
		internal.POST("/orders/:order_no/retry", handler.RetryNotify)
	}

	// 启动服务
	addr := ":" + cfg.ServerPort
	logger.Info("Server listening on %s", addr)
	if err := router.Run(addr); err != nil {
		logger.Fatal("Failed to start server: %v", err)
	}
}
