# T+1 自动对账系统

## 功能概述

GoPay 对账系统提供 T+1 自动对账功能，支持微信支付和支付宝两大支付渠道。系统每天凌晨 2 点自动执行前一天的对账任务，比对内部订单数据与支付渠道账单数据，生成对账报告并在发现差异时发送告警。

## 核心功能

### 1. 账单下载
- 支持微信支付账单下载（需集成微信支付 SDK）
- 支持支付宝账单下载（需集成支付宝 SDK）
- 自动解析 CSV 格式账单文件

### 2. 数据比对
- **双向比对**：同时检查内部订单和外部账单
- **长款检测**：外部有但内部无的订单（可能是漏单）
- **短款检测**：内部有但外部无的订单（严重问题，需立即处理）
- **金额校验**：检测订单金额是否一致

### 3. 报告生成
- 生成 CSV 格式对账报告
- 包含汇总信息、长款明细、短款明细、金额不匹配明细
- 报告保存在 `./reports` 目录

### 4. 告警通知
- 发现差异时自动发送告警
- 支持自定义告警通知器（邮件、钉钉、飞书等）

### 5. 定时调度
- 每天凌晨 2 点自动执行
- 支持手动触发对账任务
- 支持指定日期对账

## 架构设计

```
reconciliation/
├── reconciliation.go      # 对账服务主逻辑
├── wechat.go             # 微信对账器
├── alipay.go             # 支付宝对账器
├── repository.go         # 订单数据仓储
├── scheduler.go          # 定时调度器
├── report.go             # 报告生成器
└── reconciliation_test.go # 单元测试
```

## 使用方法

### 方式一：作为独立服务运行

```bash
# 启动定时对账服务
go run cmd/reconciliation/main.go \
  -db="postgres://user:pass@localhost:5432/gopay?sslmode=disable"

# 执行一次性对账（昨天）
go run cmd/reconciliation/main.go \
  -db="postgres://user:pass@localhost:5432/gopay?sslmode=disable" \
  -once

# 对指定日期执行对账
go run cmd/reconciliation/main.go \
  -db="postgres://user:pass@localhost:5432/gopay?sslmode=disable" \
  -once \
  -date="2026-04-15"
```

### 方式二：集成到主服务

```go
package main

import (
    "context"
    "database/sql"
    "gopay/internal/reconciliation"
)

func main() {
    // 连接数据库
    db, _ := sql.Open("postgres", dsn)
    
    // 创建告警通知器
    alertNotifier := &reconciliation.DummyAlertNotifier{}
    
    // 创建调度器
    scheduler := reconciliation.NewScheduler(db, alertNotifier)
    
    // 启动定时任务
    ctx := context.Background()
    go scheduler.Start(ctx)
    
    // 主服务继续运行...
}
```

### 方式三：手动触发对账

```go
// 创建对账服务
service := reconciliation.NewReconciliationService()

// 对指定日期和渠道执行对账
date := time.Date(2026, 4, 15, 0, 0, 0, 0, time.Local)
result, err := service.Reconcile(ctx, date, "wechat")

// 生成报告
reportPath, err := service.GenerateReport(ctx, result)
```

## 配置说明

### 数据库配置

对账系统需要访问订单数据库，查询已支付订单。确保数据库连接字符串正确：

```
postgres://username:password@host:port/database?sslmode=disable
```

### 告警配置

实现 `AlertNotifier` 接口来自定义告警方式：

```go
type AlertNotifier interface {
    SendAlert(ctx context.Context, message string) error
}

// 示例：钉钉告警
type DingTalkNotifier struct {
    webhook string
}

func (n *DingTalkNotifier) SendAlert(ctx context.Context, message string) error {
    // 调用钉钉 webhook API
    return nil
}
```

### 报告目录配置

默认报告保存在 `./reports` 目录，可以在创建 `ReportGenerator` 时自定义：

```go
generator := &ReportGenerator{
    reportDir: "/var/gopay/reports",
}
```

## 对账报告格式

### CSV 报告示例

```csv
GoPay 对账报告
对账日期,2026-04-16
支付渠道,wechat
生成时间,2026-04-17 02:05:30
对账状态,failed

汇总信息
总订单数,100
匹配订单数,98
长款订单数,1
短款订单数,0
金额不匹配数,1

长款明细（外部有但内部无）
订单号
ORDER_12345

金额不匹配明细
订单号,内部金额（分）,外部金额（分）,差额（分）
ORDER_67890,10000,9900,100
```

## 对账流程

```
1. 下载账单
   ├─ 调用微信/支付宝 API
   ├─ 下载前一天的交易账单
   └─ 解析 CSV 格式数据

2. 查询内部订单
   ├─ 从数据库查询已支付订单
   ├─ 筛选指定日期和渠道
   └─ 构建订单映射表

3. 双向比对
   ├─ 检查外部账单 → 查找长款
   ├─ 检查内部订单 → 查找短款
   └─ 比对金额 → 查找差异

4. 生成报告
   ├─ 汇总对账结果
   ├─ 列出差异明细
   └─ 保存 CSV 文件

5. 发送告警
   ├─ 判断是否有差异
   ├─ 构建告警消息
   └─ 发送通知
```

## 异常处理

### 长款（外部有但内部无）
- **原因**：可能是回调丢失、网络异常、系统故障
- **处理**：检查回调日志，补单处理

### 短款（内部有但外部无）
- **原因**：严重问题，可能是数据错误或欺诈
- **处理**：立即人工介入，核查订单真实性

### 金额不匹配
- **原因**：可能是退款、部分支付、数据错误
- **处理**：核对订单详情，修正数据

## 测试

```bash
# 运行所有测试
go test ./internal/reconciliation/... -v

# 查看测试覆盖率
go test ./internal/reconciliation/... -cover

# 生成覆盖率报告
go test ./internal/reconciliation/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

当前测试覆盖率：**38.3%**

## 待完成功能

### 高优先级
- [ ] 集成微信支付 SDK 实现账单下载
- [ ] 集成支付宝 SDK 实现账单下载
- [ ] 实现真实的告警通知（钉钉、邮件）
- [ ] 添加对账结果持久化（保存到数据库）

### 中优先级
- [ ] 支持 Excel 格式报告生成
- [ ] 添加对账历史查询接口
- [ ] 实现差异订单自动补单
- [ ] 添加对账任务监控和重试机制

### 低优先级
- [ ] 支持更多支付渠道（银联、PayPal 等）
- [ ] 实现对账数据可视化
- [ ] 添加对账规则配置化
- [ ] 支持分布式对账（多实例协同）

## 注意事项

1. **时区问题**：确保系统时区与支付渠道时区一致
2. **账单延迟**：支付渠道账单通常在 T+1 日凌晨生成，建议在凌晨 2 点后执行对账
3. **并发安全**：避免同时对同一日期执行多次对账
4. **数据备份**：定期备份对账报告和原始账单数据
5. **性能优化**：大量订单时考虑分批处理和并发优化

## 相关文档

- [微信支付对账单 API](https://pay.weixin.qq.com/wiki/doc/apiv3/apis/chapter3_1_6.shtml)
- [支付宝对账单 API](https://opendocs.alipay.com/open/028wob)
- [GoPay 项目文档](../../docs/)
