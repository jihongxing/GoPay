# 数据库 Schema 设计说明

## 概述

GoPay 使用 PostgreSQL 作为唯一的数据存储，遵循"简化架构"原则，不依赖 Redis 和 RabbitMQ。

## 核心设计原则

### 1. 使用行锁替代 Redis 分布式锁

```sql
-- 更新订单状态时使用 FOR UPDATE 行锁
SELECT * FROM orders WHERE order_no = $1 FOR UPDATE;
UPDATE orders SET status = 'paid' WHERE order_no = $1;
```

### 2. 遵循两条铁律

**铁律一：绝不跨网络请求持有数据库事务**
- 先提交事务，再异步通知业务系统
- 避免连接池耗尽

**铁律二：异步回调必须有严酷的超时与重试上限**
- 3秒超时
- 最多5次重试
- 指数退避策略

## 数据表设计

### 1. apps（应用表）

管理接入的业务系统。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGSERIAL | 主键 |
| app_id | VARCHAR(64) | 应用唯一标识 |
| app_name | VARCHAR(128) | 应用名称 |
| app_secret | VARCHAR(256) | 应用密钥（用于签名验证） |
| callback_url | VARCHAR(512) | 异步通知回调地址 |
| status | VARCHAR(20) | 状态：active/disabled |
| created_at | TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | 更新时间 |

**索引：**
- `idx_apps_app_id` - 应用ID查询
- `idx_apps_status` - 状态过滤

### 2. orders（订单表）

核心业务表，使用行锁保证并发安全。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGSERIAL | 主键 |
| order_no | VARCHAR(64) | GoPay 订单号（唯一） |
| app_id | VARCHAR(64) | 所属应用 |
| out_trade_no | VARCHAR(128) | 业务系统订单号 |
| channel | VARCHAR(32) | 支付渠道 |
| amount | BIGINT | 金额（分） |
| currency | VARCHAR(8) | 币种（默认CNY） |
| subject | VARCHAR(256) | 商品描述 |
| body | TEXT | 商品详情 |
| status | VARCHAR(32) | 订单状态 |
| notify_status | VARCHAR(32) | 通知状态 |
| retry_count | INT | 回调重试次数 |
| channel_order_no | VARCHAR(128) | 第三方订单号 |
| pay_url | TEXT | 支付二维码URL |
| paid_at | TIMESTAMP | 支付完成时间 |
| notified_at | TIMESTAMP | 通知业务系统时间 |
| expires_at | TIMESTAMP | 订单过期时间 |
| created_at | TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | 更新时间 |

**订单状态（status）：**
- `pending` - 待支付
- `paid` - 已支付
- `closed` - 已关闭
- `refunded` - 已退款

**通知状态（notify_status）：**
- `pending` - 待通知
- `notified` - 通知成功
- `failed_notify` - 通知失败

**索引：**
- `idx_orders_order_no` - 订单号查询（唯一）
- `idx_orders_app_id` - 应用订单查询
- `idx_orders_out_trade_no` - 业务订单号查询
- `idx_orders_status` - 状态过滤
- `idx_orders_notify_status` - 通知状态过滤
- `idx_orders_channel_order_no` - 第三方订单号查询
- `idx_orders_notify_pending` - 待通知订单查询（部分索引）

### 3. transactions（交易流水表）

用于对账和审计。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGSERIAL | 主键 |
| transaction_no | VARCHAR(64) | 交易流水号（唯一） |
| order_no | VARCHAR(64) | 关联订单号 |
| channel | VARCHAR(32) | 支付渠道 |
| channel_order_no | VARCHAR(128) | 第三方订单号 |
| type | VARCHAR(32) | 交易类型：payment/refund |
| amount | BIGINT | 金额（分） |
| status | VARCHAR(32) | 状态：success/failed/pending |
| raw_request | TEXT | 请求原始数据 |
| raw_response | TEXT | 响应原始数据 |
| created_at | TIMESTAMP | 创建时间 |

**索引：**
- `idx_transactions_transaction_no` - 流水号查询（唯一）
- `idx_transactions_order_no` - 订单流水查询
- `idx_transactions_channel_order_no` - 第三方订单号查询
- `idx_transactions_type` - 交易类型过滤
- `idx_transactions_created_at` - 时间范围查询

### 4. channel_configs（支付渠道配置表）

存储各渠道的商户配置。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGSERIAL | 主键 |
| app_id | VARCHAR(64) | 所属应用 |
| channel | VARCHAR(32) | 支付渠道 |
| config | JSONB | 渠道配置（商户号、密钥等） |
| status | VARCHAR(20) | 状态：active/disabled |
| created_at | TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | 更新时间 |

**约束：**
- `uk_channel_configs_app_channel` - 一个应用的同一渠道只能有一个配置

**索引：**
- `idx_channel_configs_app_id` - 应用配置查询
- `idx_channel_configs_channel` - 渠道过滤

**配置示例（微信支付）：**
```json
{
  "mch_id": "1234567890",
  "serial_no": "ABC123...",
  "api_v3_key": "...",
  "private_key_path": "/path/to/key.pem"
}
```

### 5. notify_logs（通知日志表）

记录异步回调的详细信息，用于排查问题。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGSERIAL | 主键 |
| order_no | VARCHAR(64) | 关联订单号 |
| callback_url | VARCHAR(512) | 回调地址 |
| request_body | TEXT | 请求体 |
| response_status | INT | HTTP状态码 |
| response_body | TEXT | 响应体 |
| success | BOOLEAN | 是否成功 |
| error_msg | TEXT | 错误信息 |
| duration_ms | INT | 耗时（毫秒） |
| created_at | TIMESTAMP | 创建时间 |

**索引：**
- `idx_notify_logs_order_no` - 订单日志查询
- `idx_notify_logs_success` - 成功/失败过滤
- `idx_notify_logs_created_at` - 时间范围查询

## 数据库迁移

### 迁移文件结构

```
migrations/
├── 001_init_schema.up.sql    # 初始化表结构
└── 001_init_schema.down.sql  # 回滚脚本
```

### 运行迁移

应用启动时会自动运行迁移：

```go
db := database.GetDB()
if err := database.RunMigrations(db, "migrations"); err != nil {
    logger.Fatal("Failed to run migrations: %v", err)
}
```

### 迁移版本管理

使用 `schema_migrations` 表记录已应用的迁移：

```sql
CREATE TABLE schema_migrations (
    version VARCHAR(255) PRIMARY KEY,
    applied_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

## 并发控制

### 订单状态更新（使用行锁）

```go
// 开始事务
tx, _ := db.Begin()

// 锁定订单行
row := tx.QueryRow("SELECT * FROM orders WHERE order_no = $1 FOR UPDATE", orderNo)

// 更新状态
tx.Exec("UPDATE orders SET status = 'paid', paid_at = NOW() WHERE order_no = $1", orderNo)

// 提交事务（铁律一：先提交事务）
tx.Commit()

// 异步通知业务系统（铁律一：事务外执行）
go notifyBusiness(order)
```

### 防止重复通知

```sql
-- 使用 CAS（Compare-And-Set）模式
UPDATE orders 
SET notify_status = 'notified', notified_at = NOW()
WHERE order_no = $1 AND notify_status = 'pending'
RETURNING id;
```

## 性能优化

### 1. 连接池配置

```go
DB.SetMaxOpenConns(25)  // 最大连接数
DB.SetMaxIdleConns(5)   // 最大空闲连接数
```

### 2. 索引优化

- 为高频查询字段创建索引
- 使用部分索引优化特定查询（如 `idx_orders_notify_pending`）
- 定期分析查询计划：`EXPLAIN ANALYZE`

### 3. 分区表（未来优化）

当订单量达到千万级别时，可考虑按时间分区：

```sql
CREATE TABLE orders_2024_01 PARTITION OF orders
FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');
```

## 数据清理策略

### 1. 归档历史订单

```sql
-- 归档6个月前的已完成订单
INSERT INTO orders_archive 
SELECT * FROM orders 
WHERE status IN ('paid', 'closed', 'refunded') 
  AND created_at < NOW() - INTERVAL '6 months';

DELETE FROM orders 
WHERE status IN ('paid', 'closed', 'refunded') 
  AND created_at < NOW() - INTERVAL '6 months';
```

### 2. 清理通知日志

```sql
-- 删除3个月前的通知日志
DELETE FROM notify_logs 
WHERE created_at < NOW() - INTERVAL '3 months';
```

## 备份策略

### 1. 每日全量备份

```bash
pg_dump -h localhost -U gopay gopay > backup_$(date +%Y%m%d).sql
```

### 2. WAL 归档（生产环境）

```sql
-- postgresql.conf
wal_level = replica
archive_mode = on
archive_command = 'cp %p /path/to/archive/%f'
```

## 监控指标

### 1. 连接池监控

```sql
SELECT count(*) FROM pg_stat_activity WHERE datname = 'gopay';
```

### 2. 慢查询监控

```sql
SELECT query, mean_exec_time, calls 
FROM pg_stat_statements 
WHERE mean_exec_time > 1000 
ORDER BY mean_exec_time DESC;
```

### 3. 表大小监控

```sql
SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables 
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
```
