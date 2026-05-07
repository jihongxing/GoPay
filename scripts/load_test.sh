#!/bin/bash
# GoPay 压力测试脚本
# 依赖: vegeta (go install github.com/tsenart/vegeta/v12@latest)
#
# 使用方法:
#   chmod +x scripts/load_test.sh
#   ./scripts/load_test.sh [target_url] [rate] [duration]
#
# 示例:
#   ./scripts/load_test.sh http://localhost:8080 1000 30s

TARGET=${1:-http://localhost:8080}
RATE=${2:-1000}
DURATION=${3:-30s}

echo "=== GoPay 压力测试 ==="
echo "目标: $TARGET"
echo "速率: ${RATE} req/s"
echo "持续: $DURATION"
echo ""

# 1. 健康检查接口
echo "--- 测试 /health ---"
echo "GET ${TARGET}/health" | vegeta attack -rate=${RATE} -duration=${DURATION} | vegeta report
echo ""

# 2. 详细健康检查
echo "--- 测试 /health/detail ---"
echo "GET ${TARGET}/health/detail" | vegeta attack -rate=${RATE} -duration=${DURATION} | vegeta report
echo ""

echo "=== 测试完成 ==="
echo ""
echo "提示: 如需测试支付接口，请确保数据库和支付渠道已配置。"
echo "可使用以下命令测试 checkout 接口:"
echo ""
echo 'echo "POST '${TARGET}'/api/v1/checkout' 
echo 'Content-Type: application/json' 
echo '@scripts/checkout_payload.json" | vegeta attack -rate=100 -duration=10s | vegeta report'
