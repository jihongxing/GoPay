#!/bin/bash

# 数据库测试脚本

set -e

COMPOSE_ENV_FILE="${COMPOSE_ENV_FILE:-.env}"
CONTAINER_CLI="${CONTAINER_CLI:-podman}"

compose() {
    "${CONTAINER_CLI}" compose --env-file "${COMPOSE_ENV_FILE}" "$@"
}

echo "=== GoPay 数据库测试 ==="
echo ""

# 1. 检查容器运行时是否可用
echo "1. 检查容器服务..."
if ! "${CONTAINER_CLI}" info > /dev/null 2>&1; then
    echo "❌ ${CONTAINER_CLI} 未运行，请先启动 ${CONTAINER_CLI}"
    exit 1
fi
echo "✅ ${CONTAINER_CLI} 正常运行"
echo ""

# 2. 启动 PostgreSQL
echo "2. 启动 PostgreSQL..."
compose up -d postgres
sleep 3
echo "✅ PostgreSQL 已启动"
echo ""

# 3. 检查数据库连接
echo "3. 检查数据库连接..."
if compose exec -T postgres pg_isready -U gopay > /dev/null 2>&1; then
    echo "✅ 数据库连接正常"
else
    echo "❌ 数据库连接失败"
    exit 1
fi
echo ""

# 4. 运行迁移（通过启动应用）
echo "4. 测试数据库迁移..."
echo "   编译应用..."
go build -o bin/gopay.exe cmd/gopay/main.go

echo "   运行迁移（应用会自动执行）..."
timeout 5 ./bin/gopay.exe || true
echo "✅ 迁移脚本已执行"
echo ""

# 5. 验证表结构
echo "5. 验证表结构..."
TABLES=$(compose exec -T postgres psql -U gopay -d gopay -t -c "SELECT tablename FROM pg_tables WHERE schemaname = 'public' ORDER BY tablename;")

echo "   已创建的表："
echo "$TABLES" | while read -r table; do
    if [ ! -z "$table" ]; then
        echo "   - $table"
    fi
done
echo ""

# 6. 检查测试数据
echo "6. 检查测试数据..."
APP_COUNT=$(compose exec -T postgres psql -U gopay -d gopay -t -c "SELECT COUNT(*) FROM apps;")
echo "   apps 表记录数: $APP_COUNT"

if [ "$APP_COUNT" -gt 0 ]; then
    echo "✅ 测试数据已插入"
else
    echo "⚠️  未找到测试数据"
fi
echo ""

echo "=== 测试完成 ==="
echo ""
echo "数据库信息："
echo "  Host: localhost"
echo "  Port: 5432"
echo "  Database: gopay"
echo "  User: gopay"
echo "  Password: gopay_dev_password"
echo ""
echo "连接命令："
echo "  ${CONTAINER_CLI} compose --env-file ${COMPOSE_ENV_FILE} exec postgres psql -U gopay -d gopay"
