# GoPay - 轻量级统一支付网关

<div align="center">

[![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.21-blue)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)
[![GitHub](https://img.shields.io/badge/GitHub-jihongxing%2FGoPay-blue?logo=github)](https://github.com/jihongxing/GoPay)

一个专为一人公司和小团队设计的轻量级 Go 语言支付网关

[English](README_EN.md) | 简体中文

</div>

---

## ✨ 为什么选择 GoPay？

### 💰 成本优化
- **节省聚合支付费用**: 只支付第三方平台基础费率（0.6%），无需额外支付聚合支付服务商费用（节省 0.2-0.5%）
- **年交易额 100 万**: 可节省 2000-5000 元
- **一个商户号管理多业务**: 通过 `app_id` 机制降低商户申请成本

### 🚀 极简架构
- **单一依赖**: 只依赖 PostgreSQL，无需 Redis/RabbitMQ
- **低维护成本**: 年度维护时间约 3 天
- **官方 SDK**: 使用微信/支付宝官方 SDK，稳定可靠

### 🎯 开源友好
- **MIT License**: 完全开源，可自由使用和修改
- **标准化接口**: 易于扩展新的支付渠道
- **完整文档**: 提供详细的接入指南和示例代码

---

## 🎉 核心特性

- ✅ **多渠道支持**: 微信支付（Native/JSAPI/H5/APP）+ 支付宝（扫码/Wap/APP/当面付）
- ✅ **多业务隔离**: 一套商户号对应多个业务系统，配置独立
- ✅ **统一接口**: 业务系统无需关心底层渠道差异
- ✅ **异步回调**: 支持 HTTP 回调，带超时和重试机制
- ✅ **安全可靠**: AES-256-GCM 加密存储密钥，签名验证防伪造
- ✅ **高性能**: 单机支持 10k+ QPS，响应时间 < 5ms
- ✅ **易部署**: Docker Compose 一键部署

---

## 📦 支持的支付方式

| 支付渠道 | 支付方式 | 适用场景 | 状态 |
|---------|---------|---------|------|
| **微信支付** | Native 扫码 | PC 网站 | ✅ |
| | JSAPI 支付 | 公众号/小程序 | ✅ |
| | H5 支付 | 手机浏览器 | ✅ |
| | APP 支付 | 原生应用 | ✅ |
| **支付宝** | 扫码支付 | PC 网站 | ✅ |
| | 手机网站支付 | 手机浏览器 | ✅ |
| | APP 支付 | 原生应用 | ✅ |
| | 当面付 | 线下收银 | ✅ |

---

## 🚀 快速开始

### 前置要求

- Go 1.21+
- Docker & Docker Compose
- PostgreSQL 15+（或使用 Docker）

### 1. 克隆项目

```bash
git clone https://github.com/yourusername/gopay.git
cd gopay
```

### 2. 启动数据库

```bash
# 使用 Docker Compose 启动 PostgreSQL
docker-compose up -d postgres

# 或使用 Make 命令
make db-up
```

### 3. 配置环境变量

```bash
# 复制配置文件
cp .env.example .env

# 编辑 .env 文件，填入你的配置
# 必填项：
# - DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME
# - MASTER_KEY (用于加密商户密钥)
```

### 4. 运行数据库迁移

```bash
# 使用 Make 命令
make migrate

# 或手动运行
psql -h localhost -U gopay -d gopay -f migrations/001_init.sql
```

### 5. 启动服务

```bash
# 开发模式
make run

# 或直接运行
go run cmd/gopay/main.go
```

服务将在 `http://localhost:8080` 启动。

### 6. 测试接口

```bash
# 健康检查
curl http://localhost:8080/health

# 创建支付订单（需要先配置渠道）
curl -X POST http://localhost:8080/api/v1/checkout \
  -H "Content-Type: application/json" \
  -d '{
    "app_id": "your_app_id",
    "out_trade_no": "TEST_ORDER_001",
    "amount": 100,
    "subject": "测试商品",
    "channel": "wechat_native",
    "notify_url": "https://your-domain.com/callback"
  }'
```

---

## 📖 文档

### 用户文档
- [快速开始指南](docs/guides/quickstart.md)
- [配置指南](docs/guides/configuration.md)
- [接入指南](docs/guides/integration.md)
- [部署指南](docs/guides/deployment.md)
- [常见问题 FAQ](docs/faq.md)

### 技术文档
- [API 接口文档](docs/api/README.md)
- [架构设计](docs/architecture/overview.md)
- [数据库设计](docs/architecture/database.md)
- [故障排查](docs/troubleshooting.md)

### 开发文档
- [开发环境搭建](docs/development/setup.md)
- [贡献指南](CONTRIBUTING.md)
- [测试指南](docs/development/testing.md)
- [扩展开发指南](docs/development/extending.md)

---

## 🏗️ 项目结构

```
GoPay/
├── cmd/gopay/          # 主程序入口
├── internal/           # 内部代码（不对外暴露）
│   ├── config/         # 配置管理
│   ├── database/       # 数据库连接
│   ├── models/         # 数据模型
│   ├── service/        # 业务逻辑
│   └── handler/        # HTTP 处理器
├── pkg/                # 可复用的公共库
│   ├── channel/        # 支付渠道接口
│   │   ├── wechat/     # 微信支付实现
│   │   └── alipay/     # 支付宝实现
│   └── errors/         # 错误定义
├── examples/           # 示例项目
│   ├── go-client/      # Go 客户端示例
│   ├── nodejs-client/  # Node.js 客户端示例
│   └── python-client/  # Python 客户端示例
├── migrations/         # 数据库迁移脚本
├── docs/               # 项目文档
└── scripts/            # 工具脚本
```

---

## 🔧 配置说明

### 环境变量

| 变量名 | 说明 | 必填 | 默认值 |
|-------|------|------|--------|
| `SERVER_PORT` | 服务端口 | 否 | 8080 |
| `DB_HOST` | 数据库地址 | 是 | - |
| `DB_PORT` | 数据库端口 | 是 | 5432 |
| `DB_USER` | 数据库用户 | 是 | - |
| `DB_PASSWORD` | 数据库密码 | 是 | - |
| `DB_NAME` | 数据库名称 | 是 | - |
| `MASTER_KEY` | 主密钥（用于加密） | 是 | - |
| `LOG_LEVEL` | 日志级别 | 否 | info |

### 渠道配置

在数据库 `channel_configs` 表中配置支付渠道：

```sql
-- 微信支付配置示例
INSERT INTO channel_configs (app_id, channel, config, status)
VALUES (
  'your_app_id',
  'wechat_native',
  '{
    "mch_id": "1234567890",
    "api_v3_key": "your_api_v3_key",
    "serial_no": "your_serial_no",
    "private_key": "-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----"
  }',
  'active'
);
```

详细配置说明请参考 [配置指南](docs/guides/configuration.md)。

---

## 🎯 使用示例

### Go 客户端

```go
package main

import (
    "bytes"
    "encoding/json"
    "net/http"
)

type CheckoutRequest struct {
    AppID       string `json:"app_id"`
    OutTradeNo  string `json:"out_trade_no"`
    Amount      int64  `json:"amount"`
    Subject     string `json:"subject"`
    Channel     string `json:"channel"`
    NotifyURL   string `json:"notify_url"`
}

func main() {
    req := CheckoutRequest{
        AppID:      "your_app_id",
        OutTradeNo: "ORDER_20260415_001",
        Amount:     100, // 单位：分
        Subject:    "测试商品",
        Channel:    "wechat_native",
        NotifyURL:  "https://your-domain.com/callback",
    }

    data, _ := json.Marshal(req)
    resp, _ := http.Post(
        "http://localhost:8080/api/v1/checkout",
        "application/json",
        bytes.NewBuffer(data),
    )
    defer resp.Body.Close()

    // 处理响应...
}
```

更多示例请查看 [examples](examples/) 目录。

---

## 🧪 测试

```bash
# 运行所有测试
make test

# 运行单元测试
make test-unit

# 运行集成测试
make test-integration

# 查看测试覆盖率
make test-coverage
```

---

## 🐳 Docker 部署

### 使用 Docker Compose（推荐）

```bash
# 启动所有服务
docker-compose up -d

# 查看日志
docker-compose logs -f gopay

# 停止服务
docker-compose down
```

### 使用 Docker

```bash
# 构建镜像
docker build -t gopay:latest .

# 运行容器
docker run -d \
  --name gopay \
  -p 8080:8080 \
  -e DB_HOST=your_db_host \
  -e DB_USER=your_db_user \
  -e DB_PASSWORD=your_db_password \
  -e MASTER_KEY=your_master_key \
  gopay:latest
```

---

## 🤝 贡献

我们欢迎所有形式的贡献！

- 提交 Bug 报告或功能建议：[创建 Issue](https://github.com/yourusername/gopay/issues)
- 提交代码：[创建 Pull Request](https://github.com/yourusername/gopay/pulls)
- 完善文档：帮助我们改进文档

请阅读 [贡献指南](CONTRIBUTING.md) 了解更多信息。

---

## 📊 性能指标

- **响应时间**: < 5ms（网关内部逻辑，不含网络 I/O）
- **并发能力**: 10k+ QPS（单机）
- **可用性**: 99.9%+
- **测试覆盖率**: 80%+

---

## 🔒 安全特性

- **密钥加密**: 使用 AES-256-GCM 加密存储商户密钥
- **签名验证**: 所有回调通知进行签名验证，防止伪造
- **金额校验**: 回调时验证金额一致性，防止篡改
- **幂等处理**: 数据库唯一索引 + 行锁，防止重复处理
- **超时控制**: HTTP 回调强制 3 秒超时，避免资源占用

---

## 📝 维护成本

- **正常情况**: 3 天/年
  - 证书更新（1 天）
  - SDK 升级（1 天）
  - 监控检查（1 天）

- **极端情况**: 1 周（5-10 年一次的大版本升级）

详细维护指南请参考 [年度维护指南](docs/年度维护指南.md)。

---

## 🗺️ 路线图

### ✅ 已完成
- [x] 微信支付（Native/JSAPI/H5/APP）
- [x] 支付宝（扫码/Wap/APP/当面付）
- [x] 多业务隔离
- [x] 异步回调机制
- [x] Docker 部署

### 🚧 进行中
- [ ] 单元测试覆盖率提升
- [ ] API 文档完善
- [ ] 示例项目

### 📅 计划中
- [ ] T+1 自动对账
- [ ] 管理后台
- [ ] 银联支付
- [ ] Stripe 支付
- [ ] Prometheus 监控
- [ ] Kubernetes Helm Chart

---

## 📄 许可证

本项目采用 [MIT License](LICENSE) 开源协议。

---

## 🙏 致谢

- [wechatpay-go](https://github.com/wechatpay-apiv3/wechatpay-go) - 微信支付官方 Go SDK
- [alipay](https://github.com/smartwalle/alipay) - 支付宝 Go SDK
- [Gin](https://github.com/gin-gonic/gin) - Go Web 框架
- [GORM](https://gorm.io/) - Go ORM 库

---

## 📮 联系方式

- **Issue**: [GitHub Issues](https://github.com/yourusername/gopay/issues)
- **讨论**: [GitHub Discussions](https://github.com/yourusername/gopay/discussions)
- **邮件**: your-email@example.com

---

<div align="center">

**如果这个项目对你有帮助，请给我们一个 ⭐️ Star！**

Made with ❤️ by GoPay Team

</div>
