package main

import (
	"fmt"
	"log"
	"os"

	"gopay/internal/config"
	"gopay/internal/database"

	"github.com/joho/godotenv"
)

func main() {
	// 加载 .env 文件
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if err := database.Connect(cfg.Database); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	migrationsPath := getEnv("MIGRATIONS_PATH", "file://migrations")
	db := database.GetDB()
	defer database.Close()

	// 解析命令
	command := "up"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	switch command {
	case "up":
		log.Println("Running migrations up...")
		if err := database.RunMigrations(db, migrationsPath); err != nil {
			log.Fatalf("Failed to run migrations: %v", err)
		}
		log.Println("Migrations completed successfully")

	case "down":
		log.Println("Rolling back last migration...")
		if err := database.RollbackLastMigration(db, migrationsPath); err != nil {
			log.Fatalf("Failed to rollback migration: %v", err)
		}
		log.Println("Rollback completed successfully")

	case "version":
		version, dirty, err := database.MigrationVersion(db, migrationsPath)
		if err != nil {
			log.Fatalf("Failed to get version: %v", err)
		}
		log.Printf("Current version: %d (dirty: %v)", version, dirty)

	case "force":
		if len(os.Args) < 3 {
			log.Fatal("Usage: migrate force <version>")
		}
		var version int
		if _, err := fmt.Sscanf(os.Args[2], "%d", &version); err != nil {
			log.Fatalf("Invalid version: %v", err)
		}
		if err := database.ForceMigrationVersion(db, migrationsPath, version); err != nil {
			log.Fatalf("Failed to force version: %v", err)
		}
		log.Printf("Forced version to %d", version)

	default:
		log.Fatalf("Unknown command: %s (available: up, down, version, force)", command)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
