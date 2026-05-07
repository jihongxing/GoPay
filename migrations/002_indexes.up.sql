-- 数据库索引优化

-- 订单表索引
CREATE INDEX IF NOT EXISTS idx_orders_channel ON orders(channel);
CREATE INDEX IF NOT EXISTS idx_orders_paid_at ON orders(paid_at);
CREATE INDEX IF NOT EXISTS idx_orders_app_channel_status ON orders(app_id, channel, status);
CREATE INDEX IF NOT EXISTS idx_orders_status_created ON orders(status, created_at DESC) WHERE status = 'closed';
CREATE INDEX IF NOT EXISTS idx_orders_channel_paid_at ON orders(channel, paid_at) WHERE status = 'paid';

-- 渠道配置表索引
CREATE INDEX IF NOT EXISTS idx_channel_configs_status ON channel_configs(status);

-- 对账与审计表索引
CREATE INDEX IF NOT EXISTS idx_reconciliation_reports_created_at ON reconciliation_reports(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_config_audit_logs_action ON config_audit_logs(action);
