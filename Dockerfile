# 多阶段构建 Dockerfile
# Stage 1: 构建阶段
FROM golang:1.25-alpine AS builder

# 设置工作目录
WORKDIR /app

# 安装构建依赖
RUN apk add --no-cache git make

# 复制 go.mod 和 go.sum
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用和迁移工具
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.Version=${VERSION:-dev} -X main.BuildTime=$(date -u '+%Y-%m-%d_%H:%M:%S')" \
    -o /app/bin/gopay \
    ./cmd/gopay && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /app/bin/migrate \
    ./cmd/migrate

# Stage 2: 运行阶段
FROM alpine:3.19

# 安装运行时依赖（包括 postgresql-client 用于健康检查）
RUN apk add --no-cache ca-certificates tzdata postgresql-client bash wget

# 创建非 root 用户
RUN addgroup -g 1000 gopay && \
    adduser -D -u 1000 -G gopay gopay

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/bin/gopay /app/gopay
COPY --from=builder /app/bin/migrate /app/migrate

# 复制迁移脚本
COPY --from=builder /app/migrations /app/migrations

# 复制入口脚本
COPY docker-entrypoint.sh /app/docker-entrypoint.sh
RUN chmod +x /app/docker-entrypoint.sh

# 设置权限
RUN chown -R gopay:gopay /app && \
    chmod +x /app/gopay /app/migrate /app/docker-entrypoint.sh

# 切换到非 root 用户
USER gopay

# 暴露端口
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# 使用入口脚本
ENTRYPOINT ["/app/docker-entrypoint.sh"]

# 启动应用
CMD ["/app/gopay"]
