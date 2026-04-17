# GoPay API 文档

## 概述

GoPay 是独立部署的支付网关服务，业务系统通过 HTTP API 调用。

- Base URL: `https://pay.example.com`
- API 版本: v1
- Content-Type: `application/json`
- 字符编码: UTF-8
- 金额单位: 分（1 元 = 100）

## 接入流程

```
1. 管理员在 GoPay 管理后台创建应用 → 获得 app_id + app_secret
2. 管理员为应用配置支付渠道（微信/支付宝的商户号和密钥）
3. 业务系统使用 app_id + app_secret 签名调用 API
4. 业务系统实现回调接口接收支付/退款通知
```

## 签名认证

所有业务接口（下单、查询）需要签名验证。Webhook 回调接口不需要（由支付平台自身签名保护）。

### 请求头

| Header | 必填 | 说明 |
|--------|------|------|
| Content-Type | 是 | `application/json` |
| X-App-ID | 是 | 应用 ID |
| X-Timestamp | 是 | Unix 时间戳（秒），5 分钟内有效 |
| X-Nonce | 是 | 随机字符串，防重放 |
| X-Signature | 是 | HMAC-SHA256 签名 |

### 签名算法

```
signature = HMAC-SHA256(app_secret, request_body + "\n" + timestamp + "\n" + nonce)
```

### 签名示例（Python）

```python
import hmac, hashlib, time, uuid, json, requests

app_id = "your_app_id"
app_secret = "your_app_secret"

body = json.dumps({
    "app_id": app_id,
    "out_trade_no": "ORDER_001",
    "amount": 100,
    "subject": "测试商品",
    "channel": "wechat_native"
})

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
print(resp.json())
```

---

## 统一响应格式

### 成功

```json
{
  "code": "SUCCESS",
  "message": "订单创建成功",
  "data": { ... }
}
```

### 失败

```json
{
  "code": "APP_NOT_FOUND",
  "message": "应用不存在",
  "details": {
    "app_id": "invalid_app"
  }
}
```

完整错误码列表见 [ERROR_CODES.md](ERROR_CODES.md)。

---

## 接口列表

### 1. 创建支付订单

`POST /api/v1/checkout`

#### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| app_id | string | 是 | 应用 ID |
| out_trade_no | string | 是 | 商户订单号（同一 app_id 下唯一） |
| amount | int | 是 | 金额（分），必须 > 0 |
| subject | string | 是 | 订单标题 |
| body | string | 否 | 订单描述 |
| channel | string | 是 | 支付渠道（见下表） |
| notify_url | string | 否 | 自定义回调地址（默认使用应用配置的 callback_url） |
| extra_data | object | 否 | 渠道特定参数 |

#### 支付渠道

| channel | 说明 | 适用场景 |
|---------|------|---------|
| `wechat_native` | 微信 Native 扫码 | PC 网站 |
| `wechat_jsapi` | 微信 JSAPI | 公众号/小程序（需传 openid） |
| `wechat_h5` | 微信 H5 | 手机浏览器 |
| `wechat_app` | 微信 APP | 原生应用 |
| `alipay_qr` | 支付宝扫码 | PC 网站 |
| `alipay_wap` | 支付宝手机网站 | 手机浏览器 |
| `alipay_app` | 支付宝 APP | 原生应用 |
| `alipay_face` | 支付宝当面付 | 线下收银 |

#### 响应

```json
{
  "code": "SUCCESS",
  "message": "订单创建成功",
  "data": {
    "order_no": "GP20260417143052123456",
    "pay_url": "weixin://wxpay/bizpayurl?pr=abc123",
    "qr_code": "weixin://wxpay/bizpayurl?pr=abc123",
    "prepay_id": "",
    "pay_info": {}
  }
}
```

| 返回字段 | 说明 | 使用场景 |
|---------|------|---------|
| order_no | GoPay 订单号 | 后续查询/退款使用 |
| pay_url | 支付链接 | Native/H5/Wap 跳转 |
| qr_code | 二维码内容 | Native 扫码，前端生成二维码 |
| prepay_id | 预支付 ID | JSAPI/APP 调起支付 |
| pay_info | 调起支付参数 | JSAPI/APP 前端调起支付 |

---

### 2. 查询订单

`GET /api/v1/orders/:order_no`

#### 响应

```json
{
  "code": "SUCCESS",
  "message": "查询成功",
  "data": {
    "order_no": "GP20260417143052123456",
    "app_id": "hotel_app",
    "out_trade_no": "ORDER_001",
    "channel": "wechat_native",
    "amount": 100,
    "currency": "CNY",
    "subject": "测试商品",
    "status": "paid",
    "notify_status": "notified",
    "channel_order_no": "4200001234202604170000000000",
    "paid_at": "2026-04-17T14:31:00+08:00",
    "created_at": "2026-04-17T14:30:52+08:00"
  }
}
```

#### 订单状态

| status | 说明 |
|--------|------|
| `pending` | 待支付 |
| `paid` | 已支付 |
| `closed` | 已关闭（过期） |
| `refunded` | 已退款 |

---
</text>
</invoke>

### 3. 发起退款

`POST /internal/api/v1/orders/:order_no/refund`

需要 `X-API-Key` 认证（管理员接口）。

#### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| amount | int | 否 | 退款金额（分），不传则全额退款 |
| reason | string | 否 | 退款原因 |

#### 响应

```json
{
  "code": "SUCCESS",
  "message": "退款已提交",
  "data": {
    "refund_no": "RFD_20260417143500_123456",
    "order_no": "GP20260417143052123456",
    "platform_trade_no": "4200001234202604170000000000",
    "platform_refund_no": "50000001234202604170000000000",
    "status": "PROCESSING",
    "amount": 100
  }
}
```

---

### 4. 查询退款

`GET /internal/api/v1/orders/:order_no/refunds/:refund_no`

需要 `X-API-Key` 认证。

---

### 5. 健康检查

`GET /health`

```json
{
  "code": "SUCCESS",
  "message": "服务正常",
  "data": {
    "status": "healthy",
    "service": "gopay",
    "version": "2.1.0"
  }
}
```

---

## 回调通知

### 支付成功通知

支付成功后，GoPay 向应用配置的 `callback_url` 发送 POST 请求。

#### 请求体

```json
{
  "order_no": "GP20260417143052123456",
  "out_trade_no": "ORDER_001",
  "amount": 100,
  "status": "paid",
  "paid_at": "2026-04-17T14:31:00+08:00",
  "channel": "wechat_native",
  "channel_order_no": "4200001234202604170000000000"
}
```

#### 业务系统响应要求

返回 HTTP 200 表示接收成功，GoPay 将停止重试。其他状态码视为失败。

#### 重试策略

失败后按指数退避重试：1s → 2s → 4s → 8s → 16s，最多 5 次。超过后标记为 `failed_notify`，可通过管理后台手动重试。

#### 幂等要求

网络抖动可能导致重复通知，业务系统必须根据 `out_trade_no` 做幂等处理。

---

### 退款成功通知

退款成功后，GoPay 同样向 `callback_url` 发送 POST 请求。

#### 请求体

```json
{
  "order_no": "GP20260417143052123456",
  "out_trade_no": "ORDER_001",
  "refund_no": "RFD_20260417143500_123456",
  "platform_refund_no": "50000001234202604170000000000",
  "amount": 100,
  "refund_amount": 100,
  "status": "refunded",
  "channel": "wechat_native",
  "refunded_at": "2026-04-17T15:00:00+08:00"
}
```

业务系统可通过 `status` 字段区分支付通知（`paid`）和退款通知（`refunded`）。

---

## 接入检查清单

业务系统接入 GoPay 前，请确认以下事项：

- [ ] 获得 `app_id` 和 `app_secret`
- [ ] 确认支付渠道已配置（联系管理员）
- [ ] 实现签名算法并验证通过
- [ ] 实现回调接口（支付通知 + 退款通知）
- [ ] 回调接口做好幂等处理
- [ ] 回调接口响应时间 < 3 秒
- [ ] 前端实现订单状态轮询（建议 2-3 秒间隔，5 分钟超时）
- [ ] 金额使用分为单位，避免浮点数

---

## 多语言客户端示例

项目 `examples/` 目录提供了完整的客户端示例：

| 语言 | 目录 | 说明 |
|------|------|------|
| Go | `examples/go-client/` | 完整的 Go 客户端 |
| Node.js | `examples/nodejs-client/` | TypeScript 客户端 |
| Python | `examples/python-client/` | Python 客户端 + Flask/FastAPI 回调示例 |
| React | `examples/frontend/` | 前端支付组件（微信/支付宝） |

---

**最后更新**: 2026-04-17
**API 版本**: v1
