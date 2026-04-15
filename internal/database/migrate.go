package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopay/pkg/logger"
)

// RunMigrations 执行数据库迁移
func RunMigrations(db *sql.DB, migrationsDir string) error {
	// 创建 schema_migrations 表（如果不存在）
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	// 读取已应用的迁移
	appliedMigrations := make(map[string]bool)
	rows, err := db.Query("SELECT version FROM schema_migrations")
	if err != nil {
		return fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return fmt.Errorf("failed to scan migration version: %w", err)
		}
		appliedMigrations[version] = true
	}

	// 读取迁移文件
	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	// 过滤并排序 .up.sql 文件
	var upFiles []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".up.sql") {
			upFiles = append(upFiles, file.Name())
		}
	}
	sort.Strings(upFiles)

	// 执行未应用的迁移
	for _, filename := range upFiles {
		version := strings.TrimSuffix(filename, ".up.sql")

		if appliedMigrations[version] {
			logger.Info("Migration already applied: %s", version)
			continue
		}

		logger.Info("Applying migration: %s", version)

		// 读取迁移文件内容
		filePath := filepath.Join(migrationsDir, filename)
		content, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", filename, err)
		}

		// 执行迁移
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}

		_, err = tx.Exec(string(content))
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to execute migration %s: %w", version, err)
		}

		// 记录迁移版本
		_, err = tx.Exec("INSERT INTO schema_migrations (version) VALUES ($1)", version)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %s: %w", version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", version, err)
		}

		logger.Info("Migration applied successfully: %s", version)
	}

	logger.Info("All migrations completed")
	return nil
}
