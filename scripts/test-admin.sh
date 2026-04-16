#!/bin/bash

# GoPay 管理后台测试脚本

echo "=========================================="
echo "GoPay 管理后台测试"
echo "=========================================="
echo ""

# 检查服务是否运行
echo "1. 检查服务状态..."
if curl -s http://localhost:8080/health > /dev/null 2>&1; then
    echo "✅ 服务正在运行"
else
    echo "❌ 服务未运行，请先启动服务: go run cmd/gopay/main.go"
    exit 1
fi

echo ""
echo "2. 测试管理后台页面..."

# 测试数据概览页面
echo -n "   - 数据概览页面: "
if curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/admin | grep -q "200"; then
    echo "✅ 正常"
else
    echo "❌ 失败"
fi

# 测试订单管理页面
echo -n "   - 订单管理页面: "
if curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/admin/orders | grep -q "200"; then
    echo "✅ 正常"
else
    echo "❌ 失败"
fi

# 测试对账报告页面
echo -n "   - 对账报告页面: "
if curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/admin/reconciliation | grep -q "200"; then
    echo "✅ 正常"
else
    echo "❌ 失败"
fi

# 测试操作日志页面
echo -n "   - 操作日志页面: "
if curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/admin/logs | grep -q "200"; then
    echo "✅ 正常"
else
    echo "❌ 失败"
fi

echo ""
echo "3. 测试 API 接口..."

# 测试统计接口
echo -n "   - 统计数据接口: "
if curl -s http://localhost:8080/admin/stats | grep -q "SUCCESS"; then
    echo "✅ 正常"
else
    echo "❌ 失败"
fi

# 测试失败订单接口
echo -n "   - 失败订单接口: "
if curl -s "http://localhost:8080/admin/orders/failed?page=1&page_size=10" | grep -q "SUCCESS"; then
    echo "✅ 正常"
else
    echo "❌ 失败"
fi

echo ""
echo "4. 测试静态资源..."

# 测试 CSS
echo -n "   - CSS 文件: "
if curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/admin/static/css/main.css | grep -q "200"; then
    echo "✅ 正常"
else
    echo "❌ 失败"
fi

# 测试 JS
echo -n "   - JS 文件: "
if curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/admin/static/js/api.js | grep -q "200"; then
    echo "✅ 正常"
else
    echo "❌ 失败"
fi

echo ""
echo "=========================================="
echo "测试完成！"
echo "=========================================="
echo ""
echo "访问管理后台: http://localhost:8080/admin"
echo ""
