package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

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

	// 为 report_id=1 (微信支付，有差异) 添加明细
	details := []struct {
		reportID       int
		orderNo        string
		detailType     string
		internalAmount *int
		externalAmount *int
		diff           *int
	}{
		// 长款：第三方有，我们没有
		{1, "WX20260415001", "long", nil, intPtr(10000), intPtr(10000)},

		// 短款：我们有，第三方没有
		{1, "WX20260415002", "short", intPtr(5000), nil, intPtr(-5000)},

		// 为 report_id=3 (微信支付，有差异) 添加明细
		{3, "WX20260414001", "short", intPtr(8000), nil, intPtr(-8000)},

		// 为 report_id=6 (支付宝，有差异) 添加明细
		{6, "ALI20260413001", "long", nil, intPtr(15000), intPtr(15000)},
		{6, "ALI20260413002", "short", intPtr(12000), nil, intPtr(-12000)},
	}

	for _, detail := range details {
		_, err := db.Exec(`
			INSERT INTO reconciliation_details
			(report_id, order_no, type, internal_amount, external_amount, diff)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, detail.reportID, detail.orderNo, detail.detailType,
			detail.internalAmount, detail.externalAmount, detail.diff)

		if err != nil {
			log.Printf("Failed to insert detail: %v", err)
		} else {
			log.Printf("Inserted detail for report %d: %s (%s)",
				detail.reportID, detail.orderNo, detail.detailType)
		}
	}

	log.Println("Test detail data insertion completed")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func intPtr(i int) *int {
	return &i
}
