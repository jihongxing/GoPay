# GoPay 部署前检查清单

**项目状态**: P0 / P1 生产整改已完成，部署前请按本清单逐项核验  
**最后更新**: 2026-05-07

---

## 部署前必检

### 1. 基础环境

- [ ] 已准备 PostgreSQL，且生产环境启用 `DB_SSLMODE=require`
- [ ] 已准备 `MASTER_KEY`、`ADMIN_API_KEY`、`DB_PASSWORD`
- [ ] 已显式设置 `SERVER_ENV=production`
- [ ] 已显式设置 `PUBLIC_BASE_URL=https://your-domain.example`
- [ ] 已配置 `ADMIN_IP_WHITELIST`，支持单 IP 和 CIDR，例如 `192.168.1.100,10.0.0.0/8`

### 2. 支付渠道密钥

- [ ] 支付宝已配置 `ALIPAY_APP_ID`
- [ ] 支付宝私钥/公钥已准备完成，推荐使用文件路径变量
  ```bash
  ALIPAY_APP_PRIVATE_KEY_PATH=certs/alipay/app_private_key.pem
  ALIPAY_PUBLIC_KEY_PATH=certs/alipay/alipay_public_key.pem
  ```
- [ ] 微信支付已配置 `WECHAT_MCH_ID`、`WECHAT_APP_ID`、`WECHAT_API_V3_KEY`
- [ ] 微信商户私钥已准备完成
  ```bash
  WECHAT_PRIVATE_KEY_PATH=certs/wechat/apiclient_key.pem
  WECHAT_SERIAL_NO=your-cert-serial
  ```

### 3. 数据库迁移

- [ ] 使用唯一正式入口执行迁移
  ```bash
  go run cmd/migrate/main.go up
  ```
- [ ] 已确认迁移版本
  ```bash
  go run cmd/migrate/main.go version
  ```
  预期当前版本至少为 `7`，且 `dirty: false`

### 4. 容器部署

- [ ] 生产环境 `.env.prod` 已填写完成
- [ ] 已验证生产 compose 配置
  ```bash
  podman compose --env-file .env.prod -f docker-compose.prod.yml config
  ```
- [ ] 已验证 Helm 模板可渲染
  ```bash
  helm template gopay ./helm/gopay
  ```

---

## 部署前回归

### 1. 代码健康检查

```bash
go build ./...
go test ./...
staticcheck ./...
```

### 2. 容器与迁移检查

```bash
podman compose --env-file .env.prod -f docker-compose.prod.yml config
go run cmd/migrate/main.go version
```

### 3. 健康检查

```bash
curl http://localhost:8080/health
curl http://localhost:8080/health/detail
```

预期响应包含：

- `code=SUCCESS`
- `data.status=healthy`

### 4. 核心支付链路

```bash
curl -X POST http://localhost:8080/api/v1/checkout \
  -H "Content-Type: application/json" \
  -d '{
    "app_id": "your_app_id",
    "channel": "alipay_qr",
    "amount": 1,
    "subject": "部署回归测试",
    "out_trade_no": "deploy_test_'$(date +%s)'"
  }'
```

```bash
curl -X POST http://localhost:8080/api/v1/checkout \
  -H "Content-Type: application/json" \
  -d '{
    "app_id": "your_app_id",
    "channel": "wechat_native",
    "amount": 1,
    "subject": "部署回归测试",
    "out_trade_no": "deploy_test_'$(date +%s)'"
  }'
```

检查点：

- 返回有效支付链接/二维码字段
- 支付成功后订单状态能更新
- 业务回调可收到通知

### 5. 管理接口认证

```bash
curl -H "X-API-Key: your-admin-api-key" \
  http://localhost:8080/admin/api/v1/stats
```

检查点：

- 未带 `X-API-Key` 时应返回 `401`
- 非白名单 IP 应返回 `403`
- 合法请求可正常返回统计信息

---

## 推荐部署方式

### Podman Compose

```bash
podman compose --env-file .env.prod -f docker-compose.prod.yml up -d
podman compose --env-file .env.prod -f docker-compose.prod.yml logs -f gopay
```

### Helm

```bash
helm upgrade --install gopay ./helm/gopay \
  --set env.SERVER_ENV=production \
  --set env.PUBLIC_BASE_URL=https://your-domain.example
```

---

## 上线后验证

- [ ] `/health` 和 `/health/detail` 正常
- [ ] 能创建测试订单
- [ ] Webhook 可正常验签入库
- [ ] 异步通知成功或失败可在 `notify_logs` 中追踪
- [ ] 管理后台失败订单查询、详情、重试功能可正常使用
- [ ] 对账报告列表、详情、下载功能可正常使用

---

## 相关文档

- [README](README.md)
- [配置指南](docs/配置指南.md)
- [运维手册](docs/运维手册.md)
- [密钥轮换指南](docs/密钥轮换指南.md)
- [生产上线整改清单](docs/生产上线整改清单.md)
