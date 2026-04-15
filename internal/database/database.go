package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"gopay/internal/config"
	"gopay/pkg/logger"
)

var DB *sql.DB

// Connect 连接数据库
func Connect(cfg config.DatabaseConfig) error {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	var err error
	DB, err = sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// 测试连接
	if err := DB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// 设置连接池参数（优化后的配置）
	DB.SetMaxOpenConns(100)                 // 最大打开连接数
	DB.SetMaxIdleConns(25)                  // 最大空闲连接数
	DB.SetConnMaxLifetime(time.Hour)        // 连接最大生命周期
	DB.SetConnMaxIdleTime(10 * time.Minute) // 空闲连接最大生命周期

	logger.Info("Database connected successfully")
	return nil
}

// GetDB 获取数据库连接
func GetDB() *sql.DB {
	return DB
}

// Close 关闭数据库连接
func Close() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}

// InitSchema 初始化数据库表结构
func InitSchema() error {
	schema := `
	-- 应用表
	CREATE TABLE IF NOT EXISTS apps (
		id SERIAL PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		app_id VARCHAR(50) UNIQUE NOT NULL,
		secret_key VARCHAR(100) NOT NULL,
		callback_url VARCHAR(500) NOT NULL,
		configs_json JSONB,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- 订单表
	CREATE TABLE IF NOT EXISTS orders (
		id SERIAL PRIMARY KEY,
		app_id INTEGER NOT NULL REFERENCES apps(id),
		biz_order_no VARCHAR(100) NOT NULL,
		platform_trade_no VARCHAR(100),
		amount INTEGER NOT NULL,
		status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
		notify_status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
		retry_count INTEGER NOT NULL DEFAULT 0,
		channel VARCHAR(50) NOT NULL,
		subject VARCHAR(200),
		extra_data JSONB,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		paid_at TIMESTAMP,
		notified_at TIMESTAMP,
		UNIQUE(app_id, biz_order_no)
	);

	-- 创建索引
	CREATE INDEX IF NOT EXISTS idx_orders_app_id ON orders(app_id);
	CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
	CREATE INDEX IF NOT EXISTS idx_orders_notify_status ON orders(notify_status);
	CREATE INDEX IF NOT EXISTS idx_orders_platform_trade_no ON orders(platform_trade_no);
	CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders(created_at);
	`

	_, err := DB.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to initialize schema: %w", err)
	}

	logger.Info("Database schema initialized successfully")
	return nil
}
