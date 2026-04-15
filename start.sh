#!/bin/bash

# GoPay 服务启动脚本

set -e

echo "=========================================="
echo "  启动 GoPay 支付网关服务"
echo "=========================================="
echo ""

# 检查环境变量文件
if [ ! -f .env ]; then
    echo "❌ 错误: .env 文件不存在"
    echo "请复制 .env.example 并配置环境变量"
    exit 1
fi

# 检查数据库是否启动
echo "📦 检查数据库连接..."
if ! docker ps | grep -q gopay-postgres; then
    echo "⚠️  数据库未启动，正在启动..."
    docker-compose up -d
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
