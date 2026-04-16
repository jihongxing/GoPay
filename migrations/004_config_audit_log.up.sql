-- 配置变更审计日志表
CREATE TABLE IF NOT EXISTS config_audit_logs (
    id BIGSERIAL PRIMARY KEY,
    operator VARCHAR(128) NOT NULL,              -- 操作人（用户名/邮箱）
    action VARCHAR(32) NOT NULL,                 -- create/update/delete/disable
    resource_type VARCHAR(32) NOT NULL,          -- app/channel_config
    resource_id VARCHAR(128) NOT NULL,           -- 资源标识（app_id 或 config_id）

    -- 变更内容
    old_value JSONB,                             -- 变更前的值
    new_value JSONB,                             -- 变更后的值

    -- 元数据
    ip_address VARCHAR(64),                      -- 操作IP
    user_agent TEXT,                             -- User-Agent

    -- 时间戳
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_config_audit_logs_resource ON config_audit_logs(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_config_audit_logs_operator ON config_audit_logs(operator);
CREATE INDEX IF NOT EXISTS idx_config_audit_logs_created_at ON config_audit_logs(created_at);

COMMENT ON TABLE config_audit_logs IS '配置变更审计日志：记录所有配置的增删改操作';
