#!/bin/bash

# GoPay 测试覆盖率报告生成脚本
# 用途：生成详细的测试覆盖率报告，帮助识别未测试的代码

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置
COVERAGE_DIR="coverage"
COVERAGE_FILE="$COVERAGE_DIR/coverage.out"
COVERAGE_HTML="$COVERAGE_DIR/coverage.html"
COVERAGE_REPORT="$COVERAGE_DIR/coverage_report.txt"
MIN_COVERAGE=60

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  GoPay 测试覆盖率报告${NC}"
echo -e "${BLUE}========================================${NC}\n"

# 创建覆盖率目录
mkdir -p $COVERAGE_DIR

# 运行测试并生成覆盖率数据
echo -e "${YELLOW}正在运行测试...${NC}"
go test ./... -coverprofile=$COVERAGE_FILE -covermode=atomic -v 2>&1 | tee $COVERAGE_DIR/test_output.log

if [ $? -ne 0 ]; then
    echo -e "${RED}测试失败！${NC}"
    exit 1
fi

echo -e "\n${GREEN}测试完成！${NC}\n"

# 生成 HTML 报告
echo -e "${YELLOW}生成 HTML 覆盖率报告...${NC}"
go tool cover -html=$COVERAGE_FILE -o $COVERAGE_HTML
echo -e "${GREEN}HTML 报告已生成: $COVERAGE_HTML${NC}\n"

# 生成详细的文本报告
echo -e "${YELLOW}生成详细覆盖率报告...${NC}"
go tool cover -func=$COVERAGE_FILE > $COVERAGE_REPORT

# 计算总体覆盖率
TOTAL_COVERAGE=$(go tool cover -func=$COVERAGE_FILE | grep total | awk '{print $3}' | sed 's/%//')

echo -e "\n${BLUE}========================================${NC}"
echo -e "${BLUE}  覆盖率统计${NC}"
echo -e "${BLUE}========================================${NC}\n"

# 按模块统计覆盖率
echo -e "${YELLOW}各模块覆盖率：${NC}\n"

echo "模块                                覆盖率" > $COVERAGE_DIR/module_coverage.txt
echo "----------------------------------------" >> $COVERAGE_DIR/module_coverage.txt

# 统计各模块覆盖率
for module in "internal/config" "internal/models" "internal/handler" "internal/service" "internal/reconciliation" "internal/admin" "pkg/security" "pkg/errors" "pkg/logger" "pkg/middleware" "pkg/channel"; do
    if grep -q "$module" $COVERAGE_REPORT; then
        coverage=$(grep "$module" $COVERAGE_REPORT | awk '{sum+=$3; count++} END {if(count>0) printf "%.1f", sum/count; else print "0.0"}')
        printf "%-35s %6s%%\n" "$module" "$coverage" | tee -a $COVERAGE_DIR/module_coverage.txt
    fi
done

echo ""

# 显示总体覆盖率
echo -e "${BLUE}========================================${NC}"
if (( $(echo "$TOTAL_COVERAGE >= $MIN_COVERAGE" | bc -l) )); then
    echo -e "${GREEN}总体覆盖率: ${TOTAL_COVERAGE}% ✓${NC}"
    echo -e "${GREEN}已达到目标覆盖率 ${MIN_COVERAGE}%${NC}"
else
    echo -e "${YELLOW}总体覆盖率: ${TOTAL_COVERAGE}%${NC}"
    echo -e "${YELLOW}距离目标覆盖率 ${MIN_COVERAGE}% 还差 $(echo "$MIN_COVERAGE - $TOTAL_COVERAGE" | bc)%${NC}"
fi
echo -e "${BLUE}========================================${NC}\n"

# 找出覆盖率最低的文件
echo -e "${YELLOW}覆盖率最低的 10 个文件：${NC}\n"
grep -v "total" $COVERAGE_REPORT | sort -k3 -n | head -10 | awk '{printf "%-60s %6s\n", $2, $3}'

echo ""

# 找出未测试的函数
echo -e "${YELLOW}未测试的函数（覆盖率 0%）：${NC}\n"
grep "0.0%" $COVERAGE_REPORT | head -20 | awk '{printf "%-60s\n", $2}'

echo -e "\n${BLUE}========================================${NC}"
echo -e "${BLUE}  报告文件${NC}"
echo -e "${BLUE}========================================${NC}\n"
echo -e "HTML 报告:   ${GREEN}$COVERAGE_HTML${NC}"
echo -e "文本报告:   ${GREEN}$COVERAGE_REPORT${NC}"
echo -e "模块统计:   ${GREEN}$COVERAGE_DIR/module_coverage.txt${NC}"
echo -e "测试日志:   ${GREEN}$COVERAGE_DIR/test_output.log${NC}"

echo -e "\n${BLUE}提示：${NC}"
echo -e "  - 在浏览器中打开 ${GREEN}$COVERAGE_HTML${NC} 查看详细的覆盖率可视化报告"
echo -e "  - 使用 ${GREEN}go test -coverprofile=coverage.out -covermode=atomic ./...${NC} 运行测试"
echo -e "  - 使用 ${GREEN}go tool cover -html=coverage.out${NC} 查看 HTML 报告\n"

# 如果覆盖率低于目标，返回非零退出码（可选）
if (( $(echo "$TOTAL_COVERAGE < $MIN_COVERAGE" | bc -l) )); then
    exit 1
fi

exit 0
