-- 回滚：删除 app_id 相关的修改

-- 删除索引
DROP INDEX IF EXISTS idx_reconciliation_reports_app_id;
DROP INDEX IF EXISTS idx_reconciliation_details_app_id;
DROP INDEX IF EXISTS idx_reconciliation_reports_unique;

-- 恢复原来的唯一约束
ALTER TABLE reconciliation_reports ADD CONSTRAINT reconciliation_reports_date_channel_key UNIQUE (date, channel);

-- 删除 app_id 字段
ALTER TABLE reconciliation_reports DROP COLUMN IF EXISTS app_id;
ALTER TABLE reconciliation_details DROP COLUMN IF EXISTS app_id;
