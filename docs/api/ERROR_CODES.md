# GoPay 统一错误码参考

所有 API 接口返回统一的 JSON 格式。

## 响应格式

### 成功响应

```json
{
  "code": "SUCCESS",
  "message": "操作成功",
  "data": { ... }
}
```

### 错误响应

```json
{
  "code": "APP_NOT_FOUND",
  "message": "应用不存在",
  "details": {
    "app_id": "invalid_app"
  }
}
```

## 错误码列表

### 通用错误

| 错误码 | HTTP 状态码 | 说明 | 排查建议 |
|--------|-----------|------|---------|
| `INVALID_REQUEST` | 400 | 请求参数错误 | 检查请求体 JSON 格式和必填字段 |
| `UNAUTHORIZED` | 401 | 未授权 | 检查签名参数或 API Key |
| `FORBIDDEN` | 403 | 禁止访问 | 检查 IP 白名单或应用状态 |
| `NOT_FOUND` | 404 | 资源不存在 | 检查请求路径 |
| `CONFLICT` | 409 | 资源冲突 | 检查是否重复提交 |
| `TOO_MANY_REQUESTS` | 429 | 请求过于频繁 | 降低请求频率 |
| `INTERNAL_ERROR` | 500 | 服务器内部错误 | 联系运维，查看服务端日志 |

### 应用相关

| 错误码 | HTTP 状态码 | 说明 | 排查建议 |
|--------|-----------|------|---------|
| `APP_NOT_FOUND` | 404 | 应用不存在 | 检查 app_id 是否正确，是否已在管理后台创建 |
| `APP_INACTIVE` | 403 | 应用已禁用 | 联系管理员启用应用 |

### 支付渠道相关

| 错误码 | HTTP 状态码 | 说明 | 排查建议 |
|--------|-----------|------|---------|
| `CHANNEL_NOT_FOUND` | 404 | 支付渠道不存在 | 检查 channel 参数，确认已在管理后台配置该渠道 |
| `CHANNEL_INACTIVE` | 403 | 支付渠道已禁用 | 联系管理员启用渠道 |
| `INVALID_CHANNEL` | 400 | 支付渠道无效 | 检查 channel 参数是否为支持的渠道名称 |

### 订单相关

| 错误码 | HTTP 状态码 | 说明 | 排查建议 |
|--------|-----------|------|---------|
| `ORDER_NOT_FOUND` | 404 | 订单不存在 | 检查 order_no 是否正确 |
| `ORDER_EXISTS` | 409 | 订单已存在 | out_trade_no 重复，请使用新的业务订单号 |
| `ORDER_PAID` | 409 | 订单已支付 | 该订单已完成支付，无需重复操作 |
| `ORDER_CLOSED` | 409 | 订单已关闭 | 订单已过期或被关闭，请创建新订单 |

### 金额相关

| 错误码 | HTTP 状态码 | 说明 | 排查建议 |
|--------|-----------|------|---------|
| `INVALID_AMOUNT` | 400 | 金额无效 | 金额必须大于 0，单位为分 |

### 支付相关

| 错误码 | HTTP 状态码 | 说明 | 排查建议 |
|--------|-----------|------|---------|
| `PAYMENT_FAILED` | 500 | 支付渠道调用失败 | 检查渠道配置，查看 details 中的具体错误 |
| `NOTIFY_FAILED` | 500 | 异步通知失败 | 检查业务系统回调地址是否可达 |

### 签名相关

| 错误码 | HTTP 状态码 | 说明 | 排查建议 |
|--------|-----------|------|---------|
| `SIGNATURE_INVALID` | 401 | 签名验证失败 | 检查签名算法和 app_secret |

## 签名验证错误排查

调用 `/api/v1/checkout` 和 `/api/v1/orders/:order_no` 时需要携带签名，常见错误：

| 现象 | 原因 | 解决方案 |
|------|------|---------|
| "缺少签名参数" | 请求头缺少必要字段 | 确保携带 X-App-ID, X-Timestamp, X-Nonce, X-Signature |
| "签名已过期" | 时间戳超过 5 分钟 | 使用当前时间戳，检查服务器时钟同步 |
| "请求已被使用" | nonce 重复 | 每次请求使用不同的随机字符串 |
| "应用不存在" | app_id 错误 | 确认 app_id 已在管理后台创建 |
| "应用已禁用" | app 状态为 disabled | 联系管理员启用 |
| "签名验证失败" | 签名计算错误 | 检查签名算法：`HMAC-SHA256(app_secret, body + "\n" + timestamp + "\n" + nonce)` |

## 签名算法示例

### Python

```python
import hmac, hashlib, time, uuid, json, requests

app_id = "your_app_id"
app_secret = "your_app_secret"
body = json.dumps({"app_id": app_id, "out_trade_no": "ORDER_001", "amount": 100, "subject": "测试", "channel": "wechat_native"})
timestamp = str(int(time.time()))
nonce = str(uuid.uuid4())

message = body + "\n" + timestamp + "\n" + nonce
signature = hmac.new(app_secret.encode(), message.encode(), hashlib.sha256).hexdigest()

resp = requests.post("https://pay.example.com/api/v1/checkout",
    data=body,
    headers={
        "Content-Type": "application/json",
        "X-App-ID": app_id,
        "X-Timestamp": timestamp,
        "X-Nonce": nonce,
        "X-Signature": signature,
    })
```

### Go

```go
body := `{"app_id":"your_app_id","out_trade_no":"ORDER_001","amount":100,"subject":"测试","channel":"wechat_native"}`
timestamp := strconv.FormatInt(time.Now().Unix(), 10)
nonce := uuid.New().String()

message := body + "\n" + timestamp + "\n" + nonce
mac := hmac.New(sha256.New, []byte(appSecret))
mac.Write([]byte(message))
signature := hex.EncodeToString(mac.Sum(nil))
```

### Node.js

```javascript
const crypto = require('crypto');
const body = JSON.stringify({app_id: 'your_app_id', out_trade_no: 'ORDER_001', amount: 100, subject: '测试', channel: 'wechat_native'});
const timestamp = Math.floor(Date.now() / 1000).toString();
const nonce = crypto.randomUUID();

const message = body + '\n' + timestamp + '\n' + nonce;
const signature = crypto.createHmac('sha256', appSecret).update(message).digest('hex');
```
