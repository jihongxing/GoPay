-- 配置模板表
CREATE TABLE IF NOT EXISTS config_templates (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    channel VARCHAR(50) NOT NULL,
    config_schema JSONB NOT NULL,
    default_values JSONB,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_config_templates_channel ON config_templates(channel);
CREATE INDEX IF NOT EXISTS idx_config_templates_is_active ON config_templates(is_active);

-- 插入默认模板
INSERT INTO config_templates (name, description, channel, config_schema, default_values) VALUES
(
    '微信支付 Native 扫码',
    '适用于 PC 网站的微信扫码支付',
    'wechat_native',
    '{
        "required": ["app_id", "mch_id", "api_v3_key", "serial_no", "private_key"],
        "properties": {
            "app_id": {"type": "string", "description": "微信 AppID"},
            "mch_id": {"type": "string", "description": "商户号"},
            "api_v3_key": {"type": "string", "description": "API v3 密钥（32位）"},
            "serial_no": {"type": "string", "description": "证书序列号"},
            "private_key": {"type": "string", "description": "商户私钥（PEM格式）"}
        }
    }',
    '{
        "app_id": "",
        "mch_id": "",
        "api_v3_key": "",
        "serial_no": "",
        "private_key": ""
    }'
),
(
    '微信支付 JSAPI',
    '适用于公众号和小程序的微信支付',
    'wechat_jsapi',
    '{
        "required": ["mch_id", "app_id", "api_v3_key", "serial_no", "private_key"],
        "properties": {
            "mch_id": {"type": "string", "description": "商户号"},
            "app_id": {"type": "string", "description": "公众号/小程序 AppID"},
            "api_v3_key": {"type": "string", "description": "API v3 密钥（32位）"},
            "serial_no": {"type": "string", "description": "证书序列号"},
            "private_key": {"type": "string", "description": "商户私钥（PEM格式）"}
        }
    }',
    '{
        "mch_id": "",
        "app_id": "",
        "api_v3_key": "",
        "serial_no": "",
        "private_key": ""
    }'
),
(
    '支付宝扫码支付',
    '适用于 PC 网站的支付宝扫码支付',
    'alipay_qr',
    '{
        "required": ["app_id", "private_key", "alipay_public_key"],
        "properties": {
            "app_id": {"type": "string", "description": "支付宝应用 ID"},
            "private_key": {"type": "string", "description": "应用私钥（RSA2）"},
            "alipay_public_key": {"type": "string", "description": "支付宝公钥"},
            "gateway": {"type": "string", "description": "网关地址", "default": "https://openapi.alipay.com/gateway.do"}
        }
    }',
    '{
        "app_id": "",
        "private_key": "",
        "alipay_public_key": "",
        "gateway": "https://openapi.alipay.com/gateway.do"
    }'
),
(
    '支付宝手机网站支付',
    '适用于手机浏览器的支付宝支付',
    'alipay_wap',
    '{
        "required": ["app_id", "private_key", "alipay_public_key"],
        "properties": {
            "app_id": {"type": "string", "description": "支付宝应用 ID"},
            "private_key": {"type": "string", "description": "应用私钥（RSA2）"},
            "alipay_public_key": {"type": "string", "description": "支付宝公钥"},
            "gateway": {"type": "string", "description": "网关地址", "default": "https://openapi.alipay.com/gateway.do"}
        }
    }',
    '{
        "app_id": "",
        "private_key": "",
        "alipay_public_key": "",
        "gateway": "https://openapi.alipay.com/gateway.do"
    }'
);

-- 添加注释
COMMENT ON TABLE config_templates IS '配置模板表，用于简化新应用接入';
COMMENT ON COLUMN config_templates.name IS '模板名称';
COMMENT ON COLUMN config_templates.description IS '模板描述';
COMMENT ON COLUMN config_templates.channel IS '支付渠道';
COMMENT ON COLUMN config_templates.config_schema IS '配置字段定义（JSON Schema）';
COMMENT ON COLUMN config_templates.default_values IS '默认配置值';
COMMENT ON COLUMN config_templates.is_active IS '是否启用';
