# 变更日志

本文档记录了 GoPay 项目的所有重要变更。

格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)，
版本号遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

---

## [Unreleased]

### 计划中
- T+1 自动对账功能
- 管理后台
- 银联支付支持
- Stripe 支付支持
- Prometheus 监控集成

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

- [Unreleased]: https://github.com/yourusername/gopay/compare/v2.0.0...HEAD
- [2.0.0]: https://github.com/yourusername/gopay/compare/v1.0.0...v2.0.0
- [1.0.0]: https://github.com/yourusername/gopay/compare/v0.1.0...v1.0.0
- [0.1.0]: https://github.com/yourusername/gopay/releases/tag/v0.1.0
