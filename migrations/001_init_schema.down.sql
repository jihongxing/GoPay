-- 回滚脚本：删除所有表和函数

DROP TRIGGER IF EXISTS update_channel_configs_updated_at ON channel_configs;
DROP TRIGGER IF EXISTS update_orders_updated_at ON orders;
DROP TRIGGER IF EXISTS update_apps_updated_at ON apps;

DROP FUNCTION IF EXISTS update_updated_at_column();

DROP TABLE IF EXISTS notify_logs;
DROP TABLE IF EXISTS channel_configs;
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS apps;
