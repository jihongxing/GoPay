#!/bin/bash

# GoPay 服务启动脚本

set -e

COMPOSE_ENV_FILE="${COMPOSE_ENV_FILE:-.env}"
COMPOSE_CMD="${COMPOSE_CMD:-podman compose --env-file ${COMPOSE_ENV_FILE}}"

echo "=========================================="
echo "  启动 GoPay 支付网关服务"
echo "=========================================="
echo ""

# 检查环境变量文件
if [ ! -f "${COMPOSE_ENV_FILE}" ]; then
    echo "❌ 错误: ${COMPOSE_ENV_FILE} 文件不存在"
    echo "请复制 .env.example 并配置环境变量"
    exit 1
fi

# 检查数据库是否启动
echo "📦 检查数据库连接..."
if ! podman ps --format "{{.Names}}" | grep -q postgres; then
    echo "⚠️  数据库未启动，正在启动..."
    bash -lc "${COMPOSE_CMD} up -d postgres adminer"
    echo "⏳ 等待数据库启动..."
    sleep 5
fi

# 编译服务
echo ""
echo "🔨 编译 GoPay 服务..."
go build -o bin/gopay cmd/gopay/main.go

if [ $? -ne 0 ]; then
    echo "❌ 编译失败"
    exit 1
fi

echo "✅ 编译成功"
echo ""

# 启动服务
echo "🚀 启动 GoPay 服务..."
echo "=========================================="
echo ""

./bin/gopay
