#!/bin/bash
# Docker 部署脚本

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m'

# 配置
IMAGE_NAME="gopay"
VERSION=${VERSION:-"latest"}
REGISTRY=${REGISTRY:-""}
COMPOSE_ENV_FILE=${COMPOSE_ENV_FILE:-".env"}
CONTAINER_CLI=${CONTAINER_CLI:-"podman"}
COMPOSE_CMD=${COMPOSE_CMD:-"${CONTAINER_CLI} compose --env-file ${COMPOSE_ENV_FILE}"}

echo -e "${GREEN}=== GoPay Docker 部署脚本 ===${NC}"

# 检查 .env 文件
if [ ! -f "${COMPOSE_ENV_FILE}" ]; then
    echo -e "${YELLOW}警告: .env 文件不存在，使用 .env.example${NC}"
    cp .env.example "${COMPOSE_ENV_FILE}"
fi

# 构建镜像
build() {
    echo -e "${GREEN}构建容器镜像...${NC}"
    ${CONTAINER_CLI} build \
        --build-arg VERSION=${VERSION} \
        -t ${IMAGE_NAME}:${VERSION} \
        -t ${IMAGE_NAME}:latest \
        .
    echo -e "${GREEN}✅ 镜像构建完成${NC}"
}

# 推送镜像
push() {
    if [ -z "$REGISTRY" ]; then
        echo -e "${RED}错误: REGISTRY 未设置${NC}"
        exit 1
    fi

    echo -e "${GREEN}推送镜像到 ${REGISTRY}...${NC}"
    ${CONTAINER_CLI} tag ${IMAGE_NAME}:${VERSION} ${REGISTRY}/${IMAGE_NAME}:${VERSION}
    ${CONTAINER_CLI} tag ${IMAGE_NAME}:latest ${REGISTRY}/${IMAGE_NAME}:latest
    ${CONTAINER_CLI} push ${REGISTRY}/${IMAGE_NAME}:${VERSION}
    ${CONTAINER_CLI} push ${REGISTRY}/${IMAGE_NAME}:latest
    echo -e "${GREEN}✅ 镜像推送完成${NC}"
}

# 启动服务
up() {
    echo -e "${GREEN}启动服务...${NC}"
    bash -lc "${COMPOSE_CMD} up -d"
    echo -e "${GREEN}✅ 服务启动完成${NC}"
    echo ""
    echo "服务地址:"
    echo "  - GoPay API: http://localhost:8080"
    echo "  - Adminer: http://localhost:8081"
    echo "  - Health Check: http://localhost:8080/health"
}

# 停止服务
down() {
    echo -e "${GREEN}停止服务...${NC}"
    bash -lc "${COMPOSE_CMD} down"
    echo -e "${GREEN}✅ 服务已停止${NC}"
}

# 查看日志
logs() {
    bash -lc "${COMPOSE_CMD} logs -f gopay"
}

# 重启服务
restart() {
    down
    up
}

# 清理
clean() {
    echo -e "${YELLOW}清理容器资源...${NC}"
    bash -lc "${COMPOSE_CMD} down -v"
    ${CONTAINER_CLI} rmi ${IMAGE_NAME}:${VERSION} ${IMAGE_NAME}:latest || true
    echo -e "${GREEN}✅ 清理完成${NC}"
}

# 健康检查
health() {
    echo -e "${GREEN}检查服务健康状态...${NC}"

    # 检查 PostgreSQL
    if bash -lc "${COMPOSE_CMD} exec -T postgres pg_isready -U gopay" > /dev/null 2>&1; then
        echo -e "${GREEN}✅ PostgreSQL: 健康${NC}"
    else
        echo -e "${RED}❌ PostgreSQL: 不健康${NC}"
    fi

    # 检查 GoPay
    if curl -f http://localhost:8080/health > /dev/null 2>&1; then
        echo -e "${GREEN}✅ GoPay: 健康${NC}"
    else
        echo -e "${RED}❌ GoPay: 不健康${NC}"
    fi
}

# 显示帮助
help() {
    echo "用法: $0 {build|push|up|down|restart|logs|clean|health}"
    echo ""
    echo "命令:"
    echo "  build    - 构建 Docker 镜像"
    echo "  push     - 推送镜像到仓库"
    echo "  up       - 启动服务"
    echo "  down     - 停止服务"
    echo "  restart  - 重启服务"
    echo "  logs     - 查看日志"
    echo "  clean    - 清理资源"
    echo "  health   - 健康检查"
    echo ""
    echo "环境变量:"
    echo "  VERSION  - 镜像版本 (默认: latest)"
    echo "  REGISTRY - 镜像仓库地址"
    echo "  COMPOSE_ENV_FILE - compose 使用的 env 文件 (默认: .env)"
    echo "  CONTAINER_CLI - 容器命令 (默认: podman)"
}

# 主逻辑
case "$1" in
    build)
        build
        ;;
    push)
        push
        ;;
    up)
        up
        ;;
    down)
        down
        ;;
    restart)
        restart
        ;;
    logs)
        logs
        ;;
    clean)
        clean
        ;;
    health)
        health
        ;;
    *)
        help
        exit 1
        ;;
esac
