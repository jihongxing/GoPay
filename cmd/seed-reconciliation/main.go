package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	// 加载 .env 文件
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found")
	}

	// 构建数据库连接字符串
	dbURL := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		getEnv("DB_HOST", "localhost"),
		getEnv("DB_PORT", "5432"),
		getEnv("DB_USER", "gopay"),
		getEnv("DB_PASSWORD", "gopay_dev_password"),
		getEnv("DB_NAME", "gopay"),
	)

	// 连接数据库
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("Connected to database successfully")

	// 插入测试数据
	testData := []struct {
		date            string
		channel         string
		totalOrders     int
		matchedOrders   int
		longOrders      int
		shortOrders     int
		amountMismatch  int
		status          string
	}{
		{time.Now().AddDate(0, 0, -1).Format("2006-01-02"), "wechat", 150, 148, 1, 1, 0, "failed"},
		{time.Now().AddDate(0, 0, -1).Format("2006-01-02"), "alipay", 200, 200, 0, 0, 0, "success"},
		{time.Now().AddDate(0, 0, -2).Format("2006-01-02"), "wechat", 180, 179, 0, 1, 0, "failed"},
		{time.Now().AddDate(0, 0, -2).Format("2006-01-02"), "alipay", 220, 220, 0, 0, 0, "success"},
		{time.Now().AddDate(0, 0, -3).Format("2006-01-02"), "wechat", 165, 165, 0, 0, 0, "success"},
		{time.Now().AddDate(0, 0, -3).Format("2006-01-02"), "alipay", 195, 193, 1, 1, 0, "failed"},
	}

	for _, data := range testData {
		_, err := db.Exec(`
			INSERT INTO reconciliation_reports
			(date, channel, total_orders, matched_orders, long_orders, short_orders, amount_mismatch, status)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (date, channel) DO NOTHING
		`, data.date, data.channel, data.totalOrders, data.matchedOrders,
		   data.longOrders, data.shortOrders, data.amountMismatch, data.status)

		if err != nil {
			log.Printf("Failed to insert test data: %v", err)
		} else {
			log.Printf("Inserted test data for %s - %s", data.date, data.channel)
		}
	}

	log.Println("Test data insertion completed")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
