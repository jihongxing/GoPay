# GoPay 服务验证指南

## 快速验证步骤

### 步骤 1: 启动数据库

```bash
cd D:\codeSpace\GoPay
docker-compose up -d
```

等待数据库启动（约5秒）。

### 步骤 2: 初始化数据库

```bash
# 运行数据库迁移
# 首次启动时会自动运行迁移
```

### 步骤 3: 启动 GoPay 服务

**Windows:**
```bash
cd D:\codeSpace\GoPay
start.bat
```

**Linux/Mac:**
```bash
cd /d/codeSpace/GoPay
chmod +x start.sh
./start.sh
```

服务将在 `http://localhost:8080` 启动。

### 步骤 4: 验证健康检查

```bash
curl http://localhost:8080/health
```

**预期响应：**
```json
{
  "code": "SUCCESS",
  "message": "服务正常",
  "data": {
    "status": "healthy",
    "service": "gopay",
    "version": "1.0.0"
  }
}
```

### 步骤 5: 初始化测试数据

```bash
cd D:\codeSpace\GoPay-Example-Shop

# Windows
init_gopay_data.bat

# Linux/Mac
./init_gopay_data.sh
```

这会在 GoPay 数据库中创建测试应用配置。

### 步骤 6: 测试创建订单（无微信配置）

```bash
curl -X POST http://localhost:8080/api/v1/checkout \
  -H "Content-Type: application/json" \
  -d '{
    "app_id": "test_app_001",
    "out_trade_no": "TEST_001",
    "amount": 100,
    "subject": "测试商品",
    "channel": "wechat_native"
  }'
```

**预期响应（如果未配置微信支付）：**
```json
{
  "code": "CHANNEL_NOT_FOUND",
  "message": "支付渠道不存在",
  "details": "channel: wechat_native"
}
```

或

```json
{
  "code": "PAYMENT_FAILED",
  "message": "支付失败",
  "details": "调用微信支付 API 失败"
}
```

这是正常的，因为还没有配置真实的微信支付参数。

### 步骤 7: 测试错误响应

#### 7.1 测试应用不存在

```bash
curl -X POST http://localhost:8080/api/v1/checkout \
  -H "Content-Type: application/json" \
  -d '{
    "app_id": "invalid_app",
    "out_trade_no": "TEST_002",
    "amount": 100,
    "subject": "测试商品",
    "channel": "wechat_native"
  }'
```

**预期响应：**
```json
{
  "code": "APP_NOT_FOUND",
  "message": "应用不存在",
  "details": "app_id: invalid_app"
}
```

#### 7.2 测试参数错误

```bash
curl -X POST http://localhost:8080/api/v1/checkout \
  -H "Content-Type: application/json" \
  -d '{
    "app_id": "test_app_001"
  }'
```

**预期响应：**
```json
{
  "code": "INVALID_REQUEST",
  "message": "请求参数错误",
  "details": "Key: 'CheckoutRequest.OutTradeNo' Error:Field validation for 'OutTradeNo' failed on the 'required' tag"
}
```

#### 7.3 测试重复订单

```bash
# 第一次创建
curl -X POST http://localhost:8080/api/v1/checkout \
  -H "Content-Type: application/json" \
  -d '{
    "app_id": "test_app_001",
    "out_trade_no": "DUPLICATE_TEST",
    "amount": 100,
    "subject": "测试商品",
    "channel": "wechat_native"
  }'

# 第二次创建（相同 out_trade_no）
curl -X POST http://localhost:8080/api/v1/checkout \
  -H "Content-Type: application/json" \
  -d '{
    "app_id": "test_app_001",
    "out_trade_no": "DUPLICATE_TEST",
    "amount": 100,
    "subject": "测试商品",
    "channel": "wechat_native"
  }'
```

**预期响应（第二次）：**
```json
{
  "code": "ORDER_EXISTS",
  "message": "订单已存在",
  "details": "out_trade_no: DUPLICATE_TEST"
}
```

### 步骤 8: 启动示例电商系统

```bash
cd D:\codeSpace\GoPay-Example-Shop

# Windows
start.bat

# Linux/Mac
./start.sh
```

服务将在 `http://localhost:3000` 启动。

### 步骤 9: 测试完整流程

1. 访问 http://localhost:3000
2. 点击任意商品的"立即购买"
3. 查看订单详情页
4. 点击"立即支付"
5. 观察控制台输出

**预期行为：**
- 如果未配置微信支付，会显示错误提示
- 错误信息会包含 GoPay 返回的错误码和详细信息

## 验证清单

### GoPay 服务

- [ ] 数据库启动成功
- [ ] GoPay 服务启动成功
- [ ] 健康检查返回正常
- [ ] 测试数据初始化成功
- [ ] 创建订单 API 可访问
- [ ] 错误响应格式正确
- [ ] 错误码返回正确

### 示例电商系统

- [ ] 服务启动成功
- [ ] 商品列表显示正常
- [ ] 订单创建成功
- [ ] 支付接口调用成功
- [ ] 错误处理正常
- [ ] 控制台日志清晰

## 常见问题

### 1. 数据库连接失败

**错误：**
```
Failed to connect to database: dial tcp 127.0.0.1:5432: connect: connection refused
```

**解决：**
```bash
cd D:\codeSpace\GoPay
docker-compose up -d
# 等待5秒
docker ps | grep gopay-postgres
```

### 2. 端口被占用

**错误：**
```
Failed to start server: listen tcp :8080: bind: address already in use
```

**解决：**
```bash
# 查找占用端口的进程
netstat -ano | findstr :8080

# 杀死进程
taskkill /PID <进程ID> /F

# 或修改 .env 中的端口
SERVER_PORT=8081
```

### 3. 编译失败

**错误：**
```
package gopay/pkg/response is not in GOROOT
```

**解决：**
```bash
# 确保在项目根目录
cd D:\codeSpace\GoPay

# 下载依赖
go mod tidy

# 重新编译
go build -o bin/gopay cmd/gopay/main.go
```

### 4. 示例电商系统连接失败

**错误：**
```
支付失败: connect ECONNREFUSED 127.0.0.1:8080
```

**解决：**
- 确保 GoPay 服务已启动
- 检查 `.env` 中的 `GOPAY_URL` 配置
- 测试 `curl http://localhost:8080/health`

## 下一步

验证成功后，可以：

1. **配置真实微信支付参数**
   - 编辑 GoPay 数据库中的 `channel_configs` 表
   - 填入真实的商户号、证书等信息

2. **测试真实支付流程**
   - 创建订单
   - 生成二维码
   - 扫码支付
   - 接收回调

3. **添加更多支付渠道**
   - 支付宝
   - 银联
   - 等

4. **部署到生产环境**
   - 配置 HTTPS
   - 添加监控
   - 配置日志
   - 添加告警

## 验证成功标志

当你看到以下输出时，说明验证成功：

### GoPay 服务启动日志

```
[INFO] Starting GoPay server...
[INFO] Database connected successfully
[INFO] Running migrations...
[INFO] Migrations completed
[INFO] Server listening on :8080
```

### 健康检查响应

```json
{
  "code": "SUCCESS",
  "message": "服务正常",
  "data": {
    "status": "healthy",
    "service": "gopay",
    "version": "1.0.0"
  }
}
```

### 示例电商系统启动日志

```
示例电商系统运行在 http://localhost:3000
GoPay 地址: http://localhost:8080
App ID: test_app_001
示例商品已插入
```

## 相关文档

- [QUICKSTART.md](QUICKSTART.md) - 快速启动指南
- [docs/API错误响应规范.md](docs/API错误响应规范.md) - API 文档
- [docs/统一错误响应更新日志.md](docs/统一错误响应更新日志.md) - 更新日志
