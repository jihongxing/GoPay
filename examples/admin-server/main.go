package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"

	"gopay/internal/admin"
)

func main() {
	// 构建数据库连接字符串
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// 从环境变量构建连接字符串
		host := getEnv("DB_HOST", "localhost")
		port := getEnv("DB_PORT", "5432")
		user := getEnv("DB_USER", "gopay")
		password := getEnv("DB_PASSWORD", "gopay_dev_password")
		dbname := getEnv("DB_NAME", "gopay")
		sslmode := getEnv("DB_SSLMODE", "disable")

		dbURL = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			host, port, user, password, dbname, sslmode)
	}

	// 连接数据库
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}
	defer db.Close()

	// 测试连接
	if err := db.Ping(); err != nil {
		log.Fatal("数据库连接测试失败:", err)
	}

	// 创建 Gin 引擎
	r := gin.Default()

	// 注册管理后台路由
	webHandler := admin.NewWebHandler(db)
	webHandler.RegisterRoutes(r)

	// 启动服务
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("管理后台启动成功: http://localhost:%s/admin", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("服务启动失败:", err)
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
