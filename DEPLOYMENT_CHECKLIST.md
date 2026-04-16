# GoPay 部署前检查清单

**项目状态**: ✅ 已完成安全审计和修复  
**测试状态**: ✅ 所有测试通过 (8/8 packages)  
**代码文件**: 71 个 Go 文件  
**最后更新**: 2026-04-16

---

## 📋 部署前必检项

### 1. 安全配置 ✅

- [ ] **生成强 MASTER_KEY**
  ```bash
  openssl rand -base64 32
  ```
  
- [ ] **生成强 ADMIN_API_KEY**
  ```bash
  openssl rand -base64 32
  ```

- [ ] **配置 ADMIN_IP_WHITELIST**
  ```bash
  # 限制管理后台访问 IP
  ADMIN_IP_WHITELIST=192.168.1.100,10.0.0.0/8
  ```

- [ ] **设置强数据库密码**
  ```bash
  DB_PASSWORD=your_strong_password_here
  ```

- [ ] **启用数据库 SSL**
  ```bash
  DB_SSLMODE=require  # 生产环境必须
  ```

### 2. 支付宝配置 ✅

- [ ] **获取应用 ID**
  - 登录 [支付宝开放平台](https://open.alipay.com/)
  - 创建应用并通过审核
  - 复制 `ALIPAY_APP_ID`

- [ ] **生成 RSA2 密钥对**
  ```bash
  openssl genrsa -out alipay_private_key.pem 2048
  openssl rsa -in alipay_private_key.pem -pubout -out alipay_public_key.pem
  ```

- [ ] **上传公钥到支付宝**
  - 应用详情 → 开发信息 → 接口加签方式
  - 上传 `alipay_public_key.pem` 内容

- [ ] **获取支付宝公钥**
  - 保存后获取「支付宝公钥」（不是应用公钥！）
  - 配置到 `ALIPAY_PUBLIC_KEY`

- [ ] **切换到正式环境**
  ```bash
  ALIPAY_GATEWAY_URL=https://openapi.alipay.com/gateway.do
  ```

### 3. 微信支付配置 ✅

- [ ] **获取商户号和 AppID**
  - 登录 [微信支付商户平台](https://pay.weixin.qq.com/)
  - 复制 `WECHAT_MCH_ID` 和 `WECHAT_APP_ID`

- [ ] **设置 API v3 密钥**
  ```bash
  # 生成 32 位密钥
  openssl rand -hex 16
  # 在商户平台设置: 账户中心 > API安全 > 设置APIv3密钥
  ```

- [ ] **申请 API 证书**
  - 账户中心 → API 安全 → 申请 API 证书
  - 下载 `apiclient_cert.pem` 和 `apiclient_key.pem`

- [ ] **查看证书序列号**
  ```bash
  openssl x509 -in apiclient_cert.pem -noout -serial
  ```

- [ ] **放置证书文件**
  ```bash
  mkdir -p certs/wechat
  cp apiclient_cert.pem certs/wechat/
  cp apiclient_key.pem certs/wechat/
  chmod 600 certs/wechat/apiclient_key.pem
  ```

- [ ] **配置 Webhook 通知地址**
  ```bash
  WECHAT_NOTIFY_URL=https://your-domain.com/api/v1/webhook/wechat
  ```

### 4. 数据库配置 ✅

- [ ] **创建数据库**
  ```bash
  createdb -U postgres gopay
  ```

- [ ] **运行数据库迁移**
  ```bash
  go run cmd/migrate/main.go up
  # 或使用 Docker
  docker-compose exec gopay /app/migrate up
  ```

- [ ] **验证迁移版本**
  ```bash
  go run cmd/migrate/main.go version
  # 应显示: Current version: 4 (dirty: false)
  ```

### 5. HTTPS 配置（生产环境必须）

- [ ] **获取 SSL 证书**
  - 使用 Let's Encrypt 或购买证书
  - 或使用云服务商提供的证书

- [ ] **配置 HTTPS**
  ```bash
  ENABLE_HTTPS=true
  TLS_CERT_PATH=/path/to/cert.pem
  TLS_KEY_PATH=/path/to/key.pem
  ```

### 6. 限流和安全配置

- [ ] **启用限流**
  ```bash
  RATE_LIMIT_ENABLED=true
  RATE_LIMIT_REQUESTS=100
  RATE_LIMIT_DURATION=1m
  ```

- [ ] **配置 CORS**
  ```bash
  CORS_ALLOWED_ORIGINS=https://your-frontend.com
  CORS_ALLOWED_METHODS=GET,POST,PUT,DELETE
  CORS_ALLOWED_HEADERS=Content-Type,Authorization,X-API-Key
  ```

### 7. 监控和日志

- [ ] **配置日志级别**
  ```bash
  LOG_LEVEL=info  # 生产环境使用 info 或 warn
  LOG_FILE=logs/gopay.log
  ```

- [ ] **配置 Sentry（可选）**
  ```bash
  SENTRY_DSN=https://xxx@sentry.io/xxx
  SENTRY_ENVIRONMENT=production
  ```

- [ ] **启用对账任务**
  ```bash
  RECONCILIATION_ENABLED=true
  RECONCILIATION_SCHEDULE=0 2 * * *  # 每天凌晨2点
  ```

---

## 🧪 部署前测试

### 1. 编译检查

```bash
# 编译所有代码
go build ./...

# 应无错误输出
```

### 2. 运行测试

```bash
# 运行所有测试
go test ./... -v

# 预期: PASS (8/8 packages)
```

### 3. 漏洞扫描

```bash
# 安装 govulncheck
go install golang.org/x/vuln/cmd/govulncheck@latest

# 扫描依赖漏洞
govulncheck ./...

# 预期: 无 HIGH/CRITICAL 漏洞
```

### 4. 测试支付宝连接

```bash
curl -X POST http://localhost:8080/api/v1/pay \
  -H "Content-Type: application/json" \
  -d '{
    "app_id": "your_app_id",
    "channel": "alipay_qr",
    "amount": 1,
    "subject": "测试订单",
    "out_trade_no": "test_'$(date +%s)'"
  }'

# 预期: 返回支付二维码 URL
```

### 5. 测试微信支付连接

```bash
curl -X POST http://localhost:8080/api/v1/pay \
  -H "Content-Type: application/json" \
  -d '{
    "app_id": "your_app_id",
    "channel": "wechat_native",
    "amount": 1,
    "subject": "测试订单",
    "out_trade_no": "test_'$(date +%s)'"
  }'

# 预期: 返回支付二维码 URL
```

### 6. 测试管理后台认证

```bash
# 无 API Key 应拒绝
curl http://localhost:8080/admin/orders
# 预期: 401 Unauthorized

# 有效 API Key 应通过
curl -H "X-API-Key: your-api-key" http://localhost:8080/admin/orders
# 预期: 200 OK
```

### 7. 测试 Webhook 签名验证

```bash
# 无效签名应拒绝
curl -X POST http://localhost:8080/api/v1/webhook/wechat \
  -H "Wechatpay-Timestamp: $(date +%s)" \
  -H "Wechatpay-Nonce: test123" \
  -H "Wechatpay-Signature: invalid" \
  -H "Wechatpay-Serial: test" \
  -d '{"test":"data"}'

# 预期: 签名验证失败
```

---

## 🚀 部署步骤

### 方式一：Docker Compose（推荐）

```bash
# 1. 构建镜像
docker-compose build

# 2. 启动服务
docker-compose up -d

# 3. 查看日志
docker-compose logs -f gopay

# 4. 运行迁移（如果 RUN_MIGRATIONS=false）
docker-compose exec gopay /app/migrate up

# 5. 验证服务
curl http://localhost:8080/health
```

### 方式二：直接运行

```bash
# 1. 编译
go build -o bin/gopay cmd/gopay/main.go

# 2. 运行迁移
go run cmd/migrate/main.go up

# 3. 启动服务
./bin/gopay

# 4. 验证服务
curl http://localhost:8080/health
```

---

## 📊 部署后验证

### 1. 健康检查

```bash
curl http://localhost:8080/health
# 预期: {"status":"ok"}
```

### 2. 数据库连接

```bash
curl -H "X-API-Key: your-api-key" \
  http://localhost:8080/admin/api/v1/stats
# 预期: 返回统计数据
```

### 3. 创建测试订单

```bash
# 支付宝测试订单
curl -X POST http://localhost:8080/api/v1/pay \
  -H "Content-Type: application/json" \
  -d '{
    "app_id": "test_app",
    "channel": "alipay_qr",
    "amount": 1,
    "subject": "生产环境测试",
    "out_trade_no": "prod_test_'$(date +%s)'"
  }'

# 微信支付测试订单
curl -X POST http://localhost:8080/api/v1/pay \
  -H "Content-Type: application/json" \
  -d '{
    "app_id": "test_app",
    "channel": "wechat_native",
    "amount": 1,
    "subject": "生产环境测试",
    "out_trade_no": "prod_test_'$(date +%s)'"
  }'
```

### 4. 查看订单

```bash
curl -H "X-API-Key: your-api-key" \
  "http://localhost:8080/admin/orders?page=1&page_size=10"
```

### 5. 测试支付回调

使用支付宝/微信支付的沙箱环境完成一笔测试支付，验证：
- Webhook 签名验证通过
- 订单状态正确更新
- 业务系统收到回调通知

---

## 🔐 安全加固建议

### 1. 密钥轮换计划

设置定期提醒：

```bash
# 添加到 crontab
0 0 1 * * /app/scripts/check-key-age.sh

# 或使用密钥轮换工具
go run cmd/rotate-keys/main.go --check-age
```

**轮换周期**：
- 管理后台 API Key: 90 天
- 支付渠道密钥: 90 天
- 数据库密码: 180 天
- MASTER_KEY: 180 天

### 2. 访问控制

- 限制管理后台访问 IP
- 使用 VPN 或堡垒机访问
- 启用双因素认证（如果支持）

### 3. 监控告警

设置以下告警规则：

- 5 分钟内 10 次认证失败 → 告警
- Webhook 签名验证失败率 > 5% → 告警
- 密钥年龄 > 85 天 → 提醒轮换
- 发现 HIGH/CRITICAL CVE → 立即告警
- 异常交易金额 → 告警
- 数据库连接失败 → 告警

### 4. 备份策略

```bash
# 数据库备份（每天）
pg_dump -U gopay gopay > backup_$(date +%Y%m%d).sql

# 配置文件备份（加密）
gpg -c .env

# 证书文件备份
tar czf certs_backup_$(date +%Y%m%d).tar.gz certs/
```

### 5. 日志审计

定期审查：
- 管理后台访问日志
- 支付订单日志
- 异常错误日志
- Webhook 回调日志

---

## 📚 相关文档

- [配置指南](docs/配置指南.md) - 详细的配置说明
- [密钥轮换指南](docs/密钥轮换指南.md) - 密钥轮换操作手册
- [安全修复报告](SECURITY_FIX_REPORT.md) - 安全审计和修复详情
- [管理后台启动指南](docs/管理后台启动指南.md) - 管理后台使用说明
- [对账系统使用指南](docs/对账系统使用指南.md) - 对账功能说明

---

## ✅ 检查清单总结

### 必须完成（生产环境）

- [ ] 生成并配置所有强密钥
- [ ] 配置支付宝正式环境
- [ ] 配置微信支付正式环境
- [ ] 启用 HTTPS
- [ ] 配置 IP 白名单
- [ ] 运行数据库迁移
- [ ] 完成所有测试验证
- [ ] 配置监控告警
- [ ] 设置备份策略
- [ ] 准备应急响应流程

### 推荐完成

- [ ] 配置 Sentry 错误追踪
- [ ] 启用限流保护
- [ ] 配置 CORS
- [ ] 启用对账任务
- [ ] 设置密钥轮换提醒
- [ ] 配置日志归档
- [ ] 准备灾难恢复计划

---

**部署负责人**: _______________  
**审核人**: _______________  
**部署日期**: _______________  
**签字确认**: _______________
