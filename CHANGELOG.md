# 变更日志

本文档记录了 GoPay 项目的所有重要变更。

格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)，
版本号遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

---

## [Unreleased]

### 计划中
- Stripe 支付完整实现
- 银联支付支持
- 测试覆盖率提升至 80%+

---

## [2.1.0] - 2026-04-17

### 新增
- 退款回调通知：微信/支付宝退款 Webhook 收到后自动异步通知业务系统
- 本地 IP 限流中间件：基于内存实现，无 Redis 依赖，符合极简架构原则
- 证书有效期检查：启动时自动检查，每天定期检查，提前 30 天告警
- 日志脱敏：自动对手机号、身份证、银行卡、邮箱、API Key 等敏感信息脱敏
- W3C Trace Context 追踪中间件：兼容 OpenTelemetry/Jaeger
- RequestID 中间件挂载到主路由
- Stripe 支付渠道脚手架（接口定义、路由注册、ChannelManager 集成）
- 新增日志脱敏、证书检查、限流中间件的单元测试

### 修复
- 对账服务 sendAlert 从空实现改为真正接入 AlertNotifier 接口
- 移除 notify_service.go 中的 TODO 注释，alertOps 已正常工作

### 改进
- 清理死代码：移除未使用的 admin/handler.go（web_handler.go 已有完整实现）
- 清理 report.go 中的大段注释代码
- 删除根目录下的 stray coverage 文件
- 更新 README 路线图，反映实际完成状态
- 更新项目结构文档

### 技术细节
- 新增 `pkg/middleware/local_rate_limit.go`
- 新增 `pkg/middleware/trace.go`
- 新增 `pkg/security/cert_checker.go`
- 新增 `pkg/logger/sanitize.go`
- 新增 `pkg/channel/stripe/provider.go`
- 新增 `internal/handler/webhook_stripe.go`
- 修改 `internal/handler/webhook.go` - 退款回调通知
- 修改 `internal/service/notify_service.go` - NotifyRefundAsync + doNotifyPayload
- 修改 `internal/reconciliation/reconciliation.go` - sendAlert 实现
- 修改 `cmd/gopay/main.go` - 挂载新中间件和证书检查

---

## [2.0.0] - 2026-04-15

### 新增
- 支付宝支付支持
  - 扫码支付（PC 网站）
  - 手机网站支付（Wap）
  - APP 支付
  - 当面付（线下扫码）
- 多渠道架构验证
- 完整的支付宝集成测试指南
- 多渠道支付快速参考文档

### 改进
- 优化 ChannelManager 架构
- 完善错误处理机制
- 改进日志记录
- 更新文档结构

### 技术细节
- 新增 `pkg/channel/alipay/` 包
- 新增 4 个支付宝渠道常量
- 新增 5 个支付宝 Provider 实现
- 代码总量增加 ~580 行

---

## [1.0.0] - 2026-04-14

### 新增
- 微信支付支持
  - Native 扫码支付（PC 网站）
  - JSAPI 支付（公众号/小程序）
  - H5 支付（手机浏览器）
  - APP 支付（原生应用）
- 统一支付接口设计
- 多业务隔离机制（app_id）
- 异步回调机制（HTTP 回调 + 重试）
- 数据库设计和迁移脚本
- Docker Compose 部署支持
- 基础文档

### 技术栈
- Go 1.21+
- PostgreSQL 15+
- Gin Web 框架
- GORM ORM 库
- wechatpay-go 官方 SDK

### 核心功能
- 统一下单接口 `/api/v1/checkout`
- 微信支付回调处理 `/api/v1/webhook/wechat`
- 订单查询接口 `/api/v1/orders/:order_no`
- 健康检查接口 `/health`

### 安全特性
- AES-256-GCM 加密存储商户密钥
- 微信支付签名验证
- 数据库行锁防止并发问题
- 幂等性保障

### 文档
- 产品需求文档 (PRD)
- 技术架构与实施方案
- 阶段1 MVP 实施清单
- 微信支付配置指南
- 微信支付集成测试指南
- 年度维护指南

---

## [0.1.0] - 2026-04-08

### 新增
- 项目初始化
- 基础项目结构
- 数据库 Schema 设计
- 环境配置管理

---

## 版本说明

### 主版本号 (Major)
当你做了不兼容的 API 修改时递增

### 次版本号 (Minor)
当你做了向下兼容的功能性新增时递增

### 修订号 (Patch)
当你做了向下兼容的问题修正时递增

---

## 链接

- [Unreleased]: https://github.com/yourusername/gopay/compare/v2.1.0...HEAD
- [2.1.0]: https://github.com/yourusername/gopay/compare/v2.0.0...v2.1.0
- [2.0.0]: https://github.com/yourusername/gopay/compare/v1.0.0...v2.0.0
- [1.0.0]: https://github.com/yourusername/gopay/compare/v0.1.0...v1.0.0
- [0.1.0]: https://github.com/yourusername/gopay/releases/tag/v0.1.0
