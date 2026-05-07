# Makefile for GoPay

# 变量定义
APP_NAME=gopay
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
ENV_FILE?=.env
CONTAINER_CLI?=podman
COMPOSE=$(CONTAINER_CLI) compose --env-file $(ENV_FILE)

# Go 相关变量
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

# 构建变量
BINARY_NAME=$(APP_NAME)
MAIN_PATH=./cmd/gopay
BUILD_DIR=./bin

# 链接标志
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

# 颜色输出
GREEN=\033[0;32m
YELLOW=\033[0;33m
NC=\033[0m

.PHONY: help build run test clean docker-up docker-down
.PHONY: fmt vet lint test-coverage db-up db-down migrate

help: ## 显示帮助信息
	@echo "GoPay 统一支付网关"
	@echo ""
	@echo "可用命令:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## 编译项目
	@echo "$(GREEN)Building $(APP_NAME)...$(NC)"
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "$(GREEN)Build complete: $(BUILD_DIR)/$(BINARY_NAME)$(NC)"

run: ## 运行项目
	@echo "$(GREEN)Running $(APP_NAME)...$(NC)"
	$(GOCMD) run $(MAIN_PATH)

test: ## 运行测试
	@echo "$(GREEN)Running tests...$(NC)"
	$(GOTEST) -v -race ./...

test-coverage: ## 运行测试并生成覆盖率报告
	@echo "$(GREEN)Running tests with coverage...$(NC)"
	$(GOTEST) -v -race -coverprofile=coverage.out -covermode=atomic ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report: coverage.html$(NC)"

fmt: ## 格式化代码
	@echo "$(GREEN)Formatting code...$(NC)"
	$(GOFMT) ./...

vet: ## 运行 go vet
	@echo "$(GREEN)Running go vet...$(NC)"
	$(GOVET) ./...

lint: ## 运行代码检查
	@echo "$(GREEN)Running linter...$(NC)"
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "$(YELLOW)golangci-lint not found, please install it$(NC)"; \
	fi

clean: ## 清理编译文件
	@echo "$(GREEN)Cleaning...$(NC)"
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

docker-up: ## 启动 Docker 服务（PostgreSQL）
	@echo "$(GREEN)Starting Docker services...$(NC)"
	$(COMPOSE) up -d

docker-down: ## 停止 Docker 服务
	@echo "$(GREEN)Stopping Docker services...$(NC)"
	$(COMPOSE) down

docker-logs: ## 查看 Docker 日志
	$(COMPOSE) logs -f

db-up: ## 启动数据库
	@echo "$(GREEN)Starting database...$(NC)"
	$(COMPOSE) up -d postgres

db-down: ## 停止数据库
	@echo "$(GREEN)Stopping database...$(NC)"
	$(COMPOSE) stop postgres

migrate: ## 运行数据库迁移
	@echo "$(GREEN)Running database migrations...$(NC)"
	@if [ -f .env ]; then \
		export $$(cat .env | grep -v '^#' | xargs) && \
		psql -h $$DB_HOST -p $$DB_PORT -U $$DB_USER -d $$DB_NAME -f migrations/001_init.sql; \
	else \
		echo ".env file not found"; \
		exit 1; \
	fi

mod-tidy: ## 整理依赖
	@echo "$(GREEN)Tidying dependencies...$(NC)"
	GOPROXY=https://goproxy.cn,direct $(GOMOD) tidy

mod-download: ## 下载依赖
	@echo "$(GREEN)Downloading dependencies...$(NC)"
	GOPROXY=https://goproxy.cn,direct $(GOMOD) download

check: fmt vet test ## 运行所有检查
	@echo "$(GREEN)All checks passed!$(NC)"
