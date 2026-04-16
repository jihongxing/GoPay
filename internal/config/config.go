package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config 应用配置
type Config struct {
	ServerPort       string
	ServerEnv        string
	PublicBaseURL    string
	AdminAPIKey      string
	AdminIPWhitelist string
	Database         DatabaseConfig
	LogLevel         string
	LogFile          string
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// Load 加载配置
func Load() (*Config, error) {
	cfg := &Config{
		ServerPort:       getEnv("SERVER_PORT", "8080"),
		ServerEnv:        getEnv("SERVER_ENV", "development"),
		PublicBaseURL:    getEnv("PUBLIC_BASE_URL", "http://localhost:8080"),
		AdminAPIKey:      getEnv("ADMIN_API_KEY", "default-insecure-key-change-me"),
		AdminIPWhitelist: getEnv("ADMIN_IP_WHITELIST", "127.0.0.1,::1"),
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "gopay"),
			Password: getEnvRequired("DB_PASSWORD"),
			DBName:   getEnv("DB_NAME", "gopay"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		LogLevel: getEnv("LOG_LEVEL", "info"),
		LogFile:  getEnv("LOG_FILE", ""),
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.Database.Password == "" {
		return fmt.Errorf("DB_PASSWORD is required")
	}
	if c.Database.User == "" {
		return fmt.Errorf("DB_USER is required")
	}
	if c.Database.DBName == "" {
		return fmt.Errorf("DB_NAME is required")
	}
	if c.ServerPort == "" {
		return fmt.Errorf("SERVER_PORT is required")
	}
	if c.ServerEnv == "production" {
		if c.AdminAPIKey == "" || c.AdminAPIKey == "default-insecure-key-change-me" {
			return fmt.Errorf("ADMIN_API_KEY is required in production")
		}
		if c.PublicBaseURL == "" || c.PublicBaseURL == "http://localhost:8080" {
			return fmt.Errorf("PUBLIC_BASE_URL must point to a public domain in production")
		}
	}
	return nil
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt 获取整数类型的环境变量
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvRequired 获取必需的环境变量
func getEnvRequired(key string) string {
	value := os.Getenv(key)
	if value == "" {
		// 在开发环境允许使用默认值，生产环境必须设置
		if os.Getenv("SERVER_ENV") == "production" {
			panic(fmt.Sprintf("Required environment variable %s is not set", key))
		}
		// 开发环境返回空字符串，由 Validate 检查
		return ""
	}
	return value
}
