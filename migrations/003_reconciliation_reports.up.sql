-- 对账报告表
CREATE TABLE IF NOT EXISTS reconciliation_reports (
    id SERIAL PRIMARY KEY,
    date DATE NOT NULL,
    channel VARCHAR(50) NOT NULL,
    total_orders INTEGER NOT NULL DEFAULT 0,
    matched_orders INTEGER NOT NULL DEFAULT 0,
    long_orders INTEGER NOT NULL DEFAULT 0,
    short_orders INTEGER NOT NULL DEFAULT 0,
    amount_mismatch INTEGER NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    file_path VARCHAR(500),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(date, channel)
);

-- 对账差异明细表
CREATE TABLE IF NOT EXISTS reconciliation_details (
    id SERIAL PRIMARY KEY,
    report_id INTEGER NOT NULL REFERENCES reconciliation_reports(id) ON DELETE CASCADE,
    order_no VARCHAR(64) NOT NULL,
    type VARCHAR(20) NOT NULL, -- 'long', 'short', 'amount_mismatch'
    internal_amount INTEGER,
    external_amount INTEGER,
    diff INTEGER,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_reconciliation_reports_date ON reconciliation_reports(date);
CREATE INDEX IF NOT EXISTS idx_reconciliation_reports_channel ON reconciliation_reports(channel);
CREATE INDEX IF NOT EXISTS idx_reconciliation_reports_status ON reconciliation_reports(status);
CREATE INDEX IF NOT EXISTS idx_reconciliation_details_report_id ON reconciliation_details(report_id);
CREATE INDEX IF NOT EXISTS idx_reconciliation_details_type ON reconciliation_details(type);
