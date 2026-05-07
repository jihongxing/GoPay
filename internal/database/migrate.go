package database

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

const defaultMigrationsPath = "file://migrations"

func normalizeMigrationsPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return defaultMigrationsPath
	}
	if strings.Contains(path, "://") {
		return path
	}
	return "file://" + path
}

func newMigrator(db *sql.DB, migrationsPath string) (*migrate.Migrate, error) {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		normalizeMigrationsPath(migrationsPath),
		"postgres",
		driver,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate instance: %w", err)
	}

	return m, nil
}

func closeMigrator(m *migrate.Migrate) error {
	sourceErr, databaseErr := m.Close()
	if sourceErr != nil {
		return sourceErr
	}
	return databaseErr
}

// RunMigrations 执行数据库迁移
func RunMigrations(db *sql.DB, migrationsPath string) error {
	m, err := newMigrator(db, migrationsPath)
	if err != nil {
		return err
	}
	defer closeMigrator(m)

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// RollbackLastMigration 回滚最近一次迁移
func RollbackLastMigration(db *sql.DB, migrationsPath string) error {
	m, err := newMigrator(db, migrationsPath)
	if err != nil {
		return err
	}
	defer closeMigrator(m)

	if err := m.Steps(-1); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to rollback migration: %w", err)
	}
	return nil
}

// MigrationVersion 获取当前迁移版本
func MigrationVersion(db *sql.DB, migrationsPath string) (uint, bool, error) {
	m, err := newMigrator(db, migrationsPath)
	if err != nil {
		return 0, false, err
	}
	defer closeMigrator(m)

	version, dirty, err := m.Version()
	if err == migrate.ErrNilVersion {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("failed to get migration version: %w", err)
	}
	return version, dirty, nil
}

// ForceMigrationVersion 强制设置迁移版本
func ForceMigrationVersion(db *sql.DB, migrationsPath string, version int) error {
	m, err := newMigrator(db, migrationsPath)
	if err != nil {
		return err
	}
	defer closeMigrator(m)

	if err := m.Force(version); err != nil {
		return fmt.Errorf("failed to force migration version: %w", err)
	}
	return nil
}
