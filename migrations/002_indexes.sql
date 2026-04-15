-- 数据库索引优化

-- 订单表索引
CREATE INDEX IF NOT EXISTS idx_orders_app_id ON orders(app_id);
CREATE INDEX IF NOT EXISTS idx_orders_out_trade_no ON orders(out_trade_no);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_channel ON orders(channel);
CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders(created_at);
CREATE INDEX IF NOT EXISTS idx_orders_paid_at ON orders(paid_at);
CREATE INDEX IF NOT EXISTS idx_orders_app_channel_status ON orders(app_id, channel, status);

-- 复合索引：用于查询失败订单
CREATE INDEX IF NOT EXISTS idx_orders_status_created ON orders(status, created_at DESC) WHERE status = 'failed';

-- 复合索引：用于对账
CREATE INDEX IF NOT EXISTS idx_orders_channel_paid_at ON orders(channel, paid_at) WHERE status = 'paid';

-- 渠道配置表索引
CREATE INDEX IF NOT EXISTS idx_channel_configs_app_id ON channel_configs(app_id);
CREATE INDEX IF NOT EXISTS idx_channel_configs_channel ON channel_configs(channel);
CREATE INDEX IF NOT EXISTS idx_channel_configs_status ON channel_configs(status);
CREATE INDEX IF NOT EXISTS idx_channel_configs_app_channel ON channel_configs(app_id, channel);

-- 对账记录表索引
CREATE INDEX IF NOT EXISTS idx_reconciliation_date ON reconciliation_logs(date);
CREATE INDEX IF NOT EXISTS idx_reconciliation_channel ON reconciliation_logs(channel);
CREATE INDEX IF NOT EXISTS idx_reconciliation_status ON reconciliation_logs(status);

-- 操作日志表索引
CREATE INDEX IF NOT EXISTS idx_operation_logs_user ON operation_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_operation_logs_action ON operation_logs(action);
CREATE INDEX IF NOT EXISTS idx_operation_logs_created ON operation_logs(created_at DESC);

-- 分析表统计信息
ANALYZE orders;
ANALYZE channel_configs;
ANALYZE reconciliation_logs;

-- 查看索引使用情况
-- SELECT schemaname, tablename, indexname, idx_scan, idx_tup_read, idx_tup_fetch
-- FROM pg_stat_user_indexes
-- WHERE schemaname = 'public'
-- ORDER BY idx_scan DESC;

-- 查看表大小
-- SELECT
--     schemaname,
--     tablename,
--     pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
-- FROM pg_tables
-- WHERE schemaname = 'public'
-- ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
