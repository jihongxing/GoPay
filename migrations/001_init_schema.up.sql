-- GoPay 数据库初始化脚本
-- 遵循铁律：使用 PostgreSQL 行锁替代 Redis

-- 应用表（业务系统）
CREATE TABLE IF NOT EXISTS apps (
    id BIGSERIAL PRIMARY KEY,
    app_id VARCHAR(64) UNIQUE NOT NULL,           -- 应用唯一标识
    app_name VARCHAR(128) NOT NULL,               -- 应用名称
    app_secret VARCHAR(256) NOT NULL,             -- 应用密钥（用于签名验证）
    callback_url VARCHAR(512) NOT NULL,           -- 异步通知回调地址
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- active/disabled
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_apps_app_id ON apps(app_id);
CREATE INDEX idx_apps_status ON apps(status);

-- 支付订单表
CREATE TABLE IF NOT EXISTS orders (
    id BIGSERIAL PRIMARY KEY,
    order_no VARCHAR(64) UNIQUE NOT NULL,         -- GoPay 订单号
    app_id VARCHAR(64) NOT NULL,                  -- 所属应用
    out_trade_no VARCHAR(128) NOT NULL,           -- 业务系统订单号
    channel VARCHAR(32) NOT NULL,                 -- 支付渠道：wechat_native/alipay_qr
    amount BIGINT NOT NULL,                       -- 金额（分）
    currency VARCHAR(8) NOT NULL DEFAULT 'CNY',   -- 币种
    subject VARCHAR(256) NOT NULL,                -- 商品描述
    body TEXT,                                    -- 商品详情

    -- 状态管理
    status VARCHAR(32) NOT NULL DEFAULT 'pending', -- pending/paid/closed/refunded
    notify_status VARCHAR(32) NOT NULL DEFAULT 'pending', -- pending/notified/failed_notify
    retry_count INT NOT NULL DEFAULT 0,           -- 回调重试次数

    -- 支付渠道相关
    channel_order_no VARCHAR(128),                -- 第三方订单号（微信/支付宝）
    pay_url TEXT,                                 -- 支付二维码URL
    paid_at TIMESTAMP,                            -- 支付完成时间
    notified_at TIMESTAMP,                        -- 通知业务系统时间

    -- 时间戳
    expires_at TIMESTAMP NOT NULL,                -- 订单过期时间
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- 约束
    CONSTRAINT fk_orders_app_id FOREIGN KEY (app_id) REFERENCES apps(app_id) ON DELETE RESTRICT
);

CREATE UNIQUE INDEX idx_orders_order_no ON orders(order_no);
CREATE UNIQUE INDEX idx_orders_app_out_trade_no ON orders(app_id, out_trade_no);
CREATE INDEX idx_orders_app_id ON orders(app_id);
CREATE INDEX idx_orders_out_trade_no ON orders(out_trade_no);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_orders_notify_status ON orders(notify_status);
CREATE INDEX idx_orders_channel_order_no ON orders(channel_order_no);
CREATE INDEX idx_orders_created_at ON orders(created_at);
-- 用于查询待通知订单
CREATE INDEX idx_orders_notify_pending ON orders(notify_status, retry_count) WHERE notify_status = 'pending';

-- 交易流水表（用于对账和审计）
CREATE TABLE IF NOT EXISTS transactions (
    id BIGSERIAL PRIMARY KEY,
    transaction_no VARCHAR(64) UNIQUE NOT NULL,   -- 交易流水号
    order_no VARCHAR(64) NOT NULL,                -- 关联订单号
    channel VARCHAR(32) NOT NULL,                 -- 支付渠道
    channel_order_no VARCHAR(128),                -- 第三方订单号

    -- 交易信息
    type VARCHAR(32) NOT NULL,                    -- payment/refund
    amount BIGINT NOT NULL,                       -- 金额（分）
    status VARCHAR(32) NOT NULL,                  -- success/failed/pending

    -- 原始数据（用于对账）
    raw_request TEXT,                             -- 请求原始数据
    raw_response TEXT,                            -- 响应原始数据

    -- 时间戳
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- 约束
    CONSTRAINT fk_transactions_order_no FOREIGN KEY (order_no) REFERENCES orders(order_no) ON DELETE RESTRICT
);

CREATE UNIQUE INDEX idx_transactions_transaction_no ON transactions(transaction_no);
CREATE INDEX idx_transactions_order_no ON transactions(order_no);
CREATE INDEX idx_transactions_channel_order_no ON transactions(channel_order_no);
CREATE INDEX idx_transactions_type ON transactions(type);
CREATE INDEX idx_transactions_created_at ON transactions(created_at);

-- 支付渠道配置表
CREATE TABLE IF NOT EXISTS channel_configs (
    id BIGSERIAL PRIMARY KEY,
    app_id VARCHAR(64) NOT NULL,                  -- 所属应用
    channel VARCHAR(32) NOT NULL,                 -- 支付渠道

    -- 配置信息（JSON格式存储）
    config JSONB NOT NULL,                        -- 渠道配置（商户号、密钥等）

    -- 状态
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- active/disabled

    -- 时间戳
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- 约束：一个应用的同一渠道只能有一个配置
    CONSTRAINT uk_channel_configs_app_channel UNIQUE (app_id, channel),
    CONSTRAINT fk_channel_configs_app_id FOREIGN KEY (app_id) REFERENCES apps(app_id) ON DELETE CASCADE
);

CREATE INDEX idx_channel_configs_app_id ON channel_configs(app_id);
CREATE INDEX idx_channel_configs_channel ON channel_configs(channel);

-- 通知日志表（用于排查问题）
CREATE TABLE IF NOT EXISTS notify_logs (
    id BIGSERIAL PRIMARY KEY,
    order_no VARCHAR(64) NOT NULL,                -- 关联订单号

    -- 通知信息
    callback_url VARCHAR(512) NOT NULL,           -- 回调地址
    request_body TEXT NOT NULL,                   -- 请求体
    response_status INT,                          -- HTTP状态码
    response_body TEXT,                           -- 响应体

    -- 结果
    success BOOLEAN NOT NULL,                     -- 是否成功
    error_msg TEXT,                               -- 错误信息
    duration_ms INT,                              -- 耗时（毫秒）

    -- 时间戳
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- 约束
    CONSTRAINT fk_notify_logs_order_no FOREIGN KEY (order_no) REFERENCES orders(order_no) ON DELETE CASCADE
);

CREATE INDEX idx_notify_logs_order_no ON notify_logs(order_no);
CREATE INDEX idx_notify_logs_success ON notify_logs(success);
CREATE INDEX idx_notify_logs_created_at ON notify_logs(created_at);

-- 创建更新时间触发器函数
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- 为需要自动更新 updated_at 的表创建触发器
CREATE TRIGGER update_apps_updated_at BEFORE UPDATE ON apps
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_orders_updated_at BEFORE UPDATE ON orders
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_channel_configs_updated_at BEFORE UPDATE ON channel_configs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 插入测试数据（开发环境使用）
INSERT INTO apps (app_id, app_name, app_secret, callback_url) VALUES
    ('test_app_001', '测试应用1', 'test_secret_123456', 'http://localhost:8080/callback')
ON CONFLICT (app_id) DO NOTHING;

COMMENT ON TABLE apps IS '应用表：管理接入的业务系统';
COMMENT ON TABLE orders IS '订单表：核心业务表，使用行锁保证并发安全';
COMMENT ON TABLE transactions IS '交易流水表：用于对账和审计';
COMMENT ON TABLE channel_configs IS '支付渠道配置表：存储各渠道的商户配置';
COMMENT ON TABLE notify_logs IS '通知日志表：记录异步回调的详细信息';
