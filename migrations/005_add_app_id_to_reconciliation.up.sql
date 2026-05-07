-- 为对账报告表添加 app_id 字段
ALTER TABLE reconciliation_reports ADD COLUMN IF NOT EXISTS app_id VARCHAR(64);

-- 为对账差异明细表添加 app_id 字段
ALTER TABLE reconciliation_details ADD COLUMN IF NOT EXISTS app_id VARCHAR(64);

-- 创建索引以支持按 app_id 查询
CREATE INDEX IF NOT EXISTS idx_reconciliation_reports_app_id ON reconciliation_reports(app_id);
CREATE INDEX IF NOT EXISTS idx_reconciliation_details_app_id ON reconciliation_details(app_id);

-- 修改唯一约束，支持同一天同一渠道不同 app 的对账
ALTER TABLE reconciliation_reports DROP CONSTRAINT IF EXISTS reconciliation_reports_date_channel_key;
CREATE UNIQUE INDEX IF NOT EXISTS idx_reconciliation_reports_unique
ON reconciliation_reports(date, channel, COALESCE(app_id, ''));

-- 添加注释
COMMENT ON COLUMN reconciliation_reports.app_id IS '应用ID，用于按应用维度对账';
COMMENT ON COLUMN reconciliation_details.app_id IS '应用ID，关联订单所属应用';
