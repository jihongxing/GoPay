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

- ✅ **多渠道支持**: 微信支付（Native/JSAPI/H5/APP）+ 支付宝（扫码/Wap/APP/当面付）+ Stripe（脚手架）
- ✅ **多业务隔离**: 一套商户号对应多个业务系统，配置独立
- ✅ **统一接口**: 业务系统无需关心底层渠道差异
- ✅ **异步回调**: 支持 HTTP 回调，带超时和重试机制
- ✅ **退款功能**: 全额/部分退款、退款查询、退款回调通知
- ✅ **T+1 对账**: 自动下载账单、比对差异、生成报告、告警通知
- ✅ **管理后台**: Web 界面管理订单、查看对账报告、操作日志
- ✅ **数据可视化**: 订单趋势图、渠道分布图、实时统计
- ✅ **安全可靠**: AES-256-GCM 加密、签名验证、日志脱敏、证书监控
- ✅ **可观测性**: Prometheus 监控、请求追踪、告警通知
- ✅ **高性能**: 单机支持 10k+ QPS，响应时间 < 5ms
- ✅ **易部署**: Docker Compose 一键部署，Helm Chart 支持

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
| **Stripe** | Checkout | 国际支付 | 🚧 脚手架 |

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
```

#### 必填配置项

**数据库配置**
```bash
DB_HOST=localhost
DB_PORT=5432
DB_USER=gopay
DB_PASSWORD=your_secure_password
DB_NAME=gopay
```

**主密钥（用于加密商户密钥）**
```bash
# 生成方法: openssl rand -base64 32
MASTER_KEY=your-master-key-change-in-production
```

**管理后台认证**
```bash
# 生成方法: openssl rand -base64 32
ADMIN_API_KEY=your-admin-api-key
# IP 白名单（可选，逗号分隔）
ADMIN_IP_WHITELIST=127.0.0.1,192.168.1.100
```

**告警配置**
```bash
ALERT_WEBHOOK_URL=https://your-alert-webhook
```

**支付宝配置**
```bash
ALIPAY_APP_ID=your_alipay_app_id
ALIPAY_APP_PRIVATE_KEY=your_alipay_private_key
ALIPAY_PUBLIC_KEY=alipay_public_key
ALIPAY_GATEWAY_URL=https://openapi.alipay.com/gateway.do
```

**微信支付配置**
```bash
WECHAT_MCH_ID=your_mch_id
WECHAT_APP_ID=your_app_id
WECHAT_API_V3_KEY=your_32_character_api_v3_key
WECHAT_SERIAL_NO=your_certificate_serial_number
WECHAT_PRIVATE_KEY_PATH=certs/wechat/apiclient_key.pem
WECHAT_CERT_PATH=certs/wechat/apiclient_cert.pem
WECHAT_NOTIFY_URL=https://your-domain.com/api/v1/webhook/wechat
WECHAT_API_URL=https://api.mch.weixin.qq.com
```

> 📖 完整配置说明请查看 `.env.example` 文件
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

附加接口：
- `GET /metrics`
- `POST /internal/api/v1/orders/:order_no/refund`
- `GET /internal/api/v1/orders/:order_no/refunds/:refund_no`

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

### 🚀 快速开始
- [配置指南](docs/配置指南.md) - **详细的配置说明和获取方法**
- [部署前检查清单](DEPLOYMENT_CHECKLIST.md) - **生产环境部署必读**
- [管理后台启动指南](docs/管理后台启动指南.md) - 管理后台使用说明
- [对账系统使用指南](docs/对账系统使用指南.md) - 对账功能说明

### 🔒 安全文档
- [安全修复报告](SECURITY_FIX_REPORT.md) - 安全审计和修复详情
- [密钥轮换指南](docs/密钥轮换指南.md) - 密钥轮换操作手册

### 📚 技术文档
- [技术架构文档](docs/GoPay%20统一支付网关%20-%20技术架构与实施方案.md) - 架构设计和实施方案
- [项目状态报告](docs/PROJECT_STATUS.md) - 当前实现状态和进度

---

## 🏗️ 项目结构

```
GoPay/
├── cmd/
│   ├── gopay/              # 主程序入口
│   ├── migrate/            # 数据库迁移工具
│   ├── reconciliation/     # 对账命令行工具
│   └── rotate-keys/        # 密钥轮换工具
├── internal/               # 内部代码（不对外暴露）
│   ├── admin/              # 管理后台（Web + 配置管理）
│   ├── config/             # 配置管理
│   ├── database/           # 数据库连接与迁移
│   ├── handler/            # HTTP 处理器
│   ├── metrics/            # Prometheus 指标
│   ├── models/             # 数据模型
│   ├── reconciliation/     # T+1 对账服务
│   └── service/            # 业务逻辑层
├── pkg/                    # 可复用的公共库
│   ├── alert/              # 告警通知（Webhook/钉钉）
│   ├── channel/            # 支付渠道接口
│   │   ├── alipay/         # 支付宝实现
│   │   ├── stripe/         # Stripe 实现（脚手架）
│   │   └── wechat/         # 微信支付实现
│   ├── errors/             # 统一错误定义
│   ├── logger/             # 日志（含脱敏）
│   ├── middleware/         # 中间件（认证/限流/追踪/监控）
│   ├── security/           # 加密与证书管理
│   └── version/            # 版本信息
├── examples/               # 多语言客户端示例
├── migrations/             # 数据库迁移脚本
├── docker/                 # Grafana/Prometheus 配置
├── helm/                   # Kubernetes Helm Chart
└── docs/                   # 项目文档
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
- **测试覆盖率**: 
  - 核心配置模块: 92.3%
  - 数据模型: 75.0%
  - Handler 层: 47.4%
  - 对账系统: 38.3%
  - 业务服务: 22.5%
  - 安全模块: 48.9%
  - 总体覆盖率: 47.4%（持续改进中）

---

## 🔒 安全特性

- **密钥加密**: 使用 AES-256-GCM 加密存储商户密钥
- **签名验证**: 使用官方 SDK 验证 Webhook RSA-SHA256 签名，防止伪造
- **金额校验**: 回调时验证金额一致性，防止篡改
- **幂等处理**: 数据库唯一索引 + 行锁，防止重复处理
- **超时控制**: HTTP 回调强制 3 秒超时，避免资源占用
- **管理后台认证**: API Key + IP 白名单双重保护
- **密钥轮换**: 提供密钥轮换工具，建议每 90 天轮换
- **API 限流**: 基于内存的 IP 限流，防止接口滥用
- **日志脱敏**: 自动对手机号、身份证、银行卡等敏感信息脱敏
- **证书监控**: 自动检查证书有效期，提前 30 天告警
- **请求追踪**: W3C Trace Context 兼容，支持分布式追踪
- **依赖扫描**: CI/CD 集成 govulncheck 自动扫描漏洞

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
- [x] 多业务隔离（app_id 机制）
- [x] 异步回调机制（3s 超时 + 5 次指数退避重试）
- [x] 退款功能（全额/部分退款 + 退款查询 + 退款回调通知）
- [x] T+1 自动对账系统（微信/支付宝账单下载、差异检测、报告生成）
- [x] 管理后台（订单管理、通知重试、对账报告、操作日志、数据可视化）
- [x] 配置管理（应用管理、渠道配置、审计日志）
- [x] Prometheus 监控指标 + Grafana Dashboard
- [x] 告警通知（Webhook/钉钉，支持支付失败、通知失败、对账异常、证书过期）
- [x] API 限流（基于内存的 IP 限流，无 Redis 依赖）
- [x] 请求追踪（W3C Trace Context 兼容，支持 OpenTelemetry/Jaeger）
- [x] 日志脱敏（手机号、身份证、银行卡、API Key 等敏感信息自动脱敏）
- [x] 证书有效期检查（提前 30 天告警）
- [x] Docker Compose 一键部署
- [x] Kubernetes Helm Chart
- [x] 多语言客户端示例（Go/Node.js/Python/React）

### 📅 计划中
- [ ] Stripe 支付（脚手架已就位）
- [ ] 银联支付
- [ ] 提升测试覆盖率至 80%+
- [ ] 分布式追踪（Jaeger 完整集成）
- [ ] Redis 缓存层（可选）

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
