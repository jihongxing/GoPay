# GoPay API 文档

欢迎使用 GoPay 统一支付网关 API 文档。

## 📋 概述

GoPay 提供简单、统一的 RESTful API 接口，支持微信支付和支付宝的多种支付方式。

**Base URL**: `http://your-domain.com`  
**API Version**: v1  
**Content-Type**: `application/json`

---

## 🔐 认证

GoPay 使用 `app_id` 进行身份认证。每个请求都需要在请求体中包含 `app_id` 字段。

```json
{
  "app_id": "your_app_id",
  ...
}
```

---

## 📡 接口列表

### 1. 创建支付订单

创建一个新的支付订单。

**接口地址**: `POST /api/v1/checkout`

**请求参数**:

| 参数 | 类型 | 必填 | 说明 |
|-----|------|------|------|
| app_id | string | 是 | 应用ID |
| out_trade_no | string | 是 | 商户订单号（唯一） |
| amount | integer | 是 | 支付金额（单位：分） |
| currency | string | 否 | 货币类型，默认 CNY |
| subject | string | 是 | 订单标题 |
| body | string | 否 | 订单描述 |
| channel | string | 是 | 支付渠道（见下表） |
| notify_url | string | 是 | 异步回调地址 |
| extra_data | object | 否 | 额外数据（根据渠道不同） |

**支付渠道**:

| 渠道代码 | 说明 | 适用场景 |
|---------|------|---------|
| wechat_native | 微信 Native 扫码 | PC 网站 |
| wechat_jsapi | 微信 JSAPI | 公众号/小程序 |
| wechat_h5 | 微信 H5 | 手机浏览器 |
| wechat_app | 微信 APP | 原生应用 |
| alipay_qr | 支付宝扫码 | PC 网站 |
| alipay_wap | 支付宝手机网站 | 手机浏览器 |
| alipay_app | 支付宝 APP | 原生应用 |
| alipay_face | 支付宝当面付 | 线下收银 |

**请求示例**:

```bash
curl -X POST http://localhost:8080/api/v1/checkout \
  -H "Content-Type: application/json" \
  -d '{
    "app_id": "your_app_id",
    "out_trade_no": "ORDER_20260416_001",
    "amount": 100,
    "subject": "测试商品",
    "body": "这是一个测试订单",
    "channel": "wechat_native",
    "notify_url": "https://your-domain.com/callback"
  }'
```

**响应参数**:

| 参数 | 类型 | 说明 |
|-----|------|------|
| code | integer | 状态码（0 表示成功） |
| message | string | 响应消息 |
| data | object | 响应数据 |
| data.order_no | string | GoPay 订单号 |
| data.pay_url | string | 支付链接（部分渠道） |
| data.qr_code | string | 二维码内容（扫码支付） |
| data.prepay_id | string | 预支付ID（APP 支付） |
| data.pay_info | object | 调起支付参数（APP 支付） |

**响应示例**:

```json
{
  "code": 0,
  "message": "订单创建成功",
  "data": {
    "order_no": "GO20260416123456789",
    "qr_code": "weixin://wxpay/bizpayurl?pr=abc123",
    "pay_url": "weixin://wxpay/bizpayurl?pr=abc123"
  }
}
```

---

### 2. 查询订单状态

查询订单的支付状态。

**接口地址**: `GET /api/v1/orders/:order_no`

**路径参数**:

| 参数 | 类型 | 必填 | 说明 |
|-----|------|------|------|
| order_no | string | 是 | GoPay 订单号 |

**请求示例**:

```bash
curl http://localhost:8080/api/v1/orders/GO20260416123456789
```

**响应参数**:

| 参数 | 类型 | 说明 |
|-----|------|------|
| code | integer | 状态码（0 表示成功） |
| message | string | 响应消息 |
| data | object | 订单信息 |
| data.order_no | string | GoPay 订单号 |
| data.app_id | string | 应用ID |
| data.out_trade_no | string | 商户订单号 |
| data.amount | integer | 支付金额（分） |
| data.currency | string | 货币类型 |
| data.subject | string | 订单标题 |
| data.body | string | 订单描述 |
| data.channel | string | 支付渠道 |
| data.status | string | 订单状态（pending/paid/failed/closed） |
| data.paid_at | string | 支付时间（ISO 8601） |
| data.created_at | string | 创建时间（ISO 8601） |
| data.updated_at | string | 更新时间（ISO 8601） |

**响应示例**:

```json
{
  "code": 0,
  "message": "查询成功",
  "data": {
    "order_no": "GO20260416123456789",
    "app_id": "your_app_id",
    "out_trade_no": "ORDER_20260416_001",
    "amount": 100,
    "currency": "CNY",
    "subject": "测试商品",
    "body": "这是一个测试订单",
    "channel": "wechat_native",
    "status": "paid",
    "paid_at": "2026-04-16T10:30:00Z",
    "created_at": "2026-04-16T10:25:00Z",
    "updated_at": "2026-04-16T10:30:00Z"
  }
}
```

---

### 3. 支付回调通知

支付成功后，GoPay 会向商户配置的 `notify_url` 发送异步通知。

**接口地址**: `POST {notify_url}`（商户配置的回调地址）

**请求参数**:

| 参数 | 类型 | 说明 |
|-----|------|------|
| order_no | string | GoPay 订单号 |
| out_trade_no | string | 商户订单号 |
| amount | integer | 支付金额（分） |
| currency | string | 货币类型 |
| channel | string | 支付渠道 |
| status | string | 订单状态 |
| paid_at | string | 支付时间（ISO 8601） |

**请求示例**:

```json
{
  "order_no": "GO20260416123456789",
  "out_trade_no": "ORDER_20260416_001",
  "amount": 100,
  "currency": "CNY",
  "channel": "wechat_native",
  "status": "paid",
  "paid_at": "2026-04-16T10:30:00Z"
}
```

**响应要求**:

商户系统需要返回以下格式的响应：

```json
{
  "code": "SUCCESS",
  "message": "OK"
}
```

**重试机制**:

- 如果商户系统返回非 200 状态码或响应格式不正确，GoPay 会进行重试
- 重试策略：1s, 2s, 4s, 8s, 16s（最多 5 次）
- 商户系统需要做好幂等处理

---

### 4. 健康检查

检查服务是否正常运行。

**接口地址**: `GET /health`

**请求示例**:

```bash
curl http://localhost:8080/health
```

**响应示例**:

```json
{
  "status": "ok"
}
```

---

## 📝 错误码

| 错误码 | 说明 |
|-------|------|
| 0 | 成功 |
| 1001 | 参数错误 |
| 1002 | 应用不存在 |
| 1003 | 渠道配置不存在 |
| 1004 | 订单已存在 |
| 1005 | 订单不存在 |
| 2001 | 渠道错误 |
| 2002 | 创建订单失败 |
| 2003 | 查询订单失败 |
| 5000 | 内部错误 |

**错误响应示例**:

```json
{
  "code": 1001,
  "message": "参数错误: amount 必须大于 0",
  "data": null
}
```

---

## 🔍 支付渠道详细说明

### 微信 Native 扫码支付

**渠道代码**: `wechat_native`

**适用场景**: PC 网站

**返回字段**:
- `qr_code`: 二维码内容
- `pay_url`: 支付链接（与 qr_code 相同）

**前端处理**:
```javascript
// 生成二维码展示给用户
QRCode.toCanvas(canvas, response.data.qr_code);
```

---

### 微信 JSAPI 支付

**渠道代码**: `wechat_jsapi`

**适用场景**: 微信公众号、小程序

**extra_data 参数**:
```json
{
  "extra_data": {
    "openid": "用户的 openid"
  }
}
```

**返回字段**:
- `prepay_id`: 预支付ID
- `pay_info`: 调起支付参数

**前端处理**:
```javascript
// 调起微信支付
wx.chooseWXPay({
  ...response.data.pay_info,
  success: function(res) {
    console.log('支付成功');
  }
});
```

---

### 支付宝扫码支付

**渠道代码**: `alipay_qr`

**适用场景**: PC 网站

**返回字段**:
- `qr_code`: 二维码内容
- `pay_url`: 支付链接（与 qr_code 相同）

**前端处理**:
```javascript
// 生成二维码展示给用户
QRCode.toCanvas(canvas, response.data.qr_code);
```

---

### 支付宝手机网站支付

**渠道代码**: `alipay_wap`

**适用场景**: 手机浏览器

**extra_data 参数**:
```json
{
  "extra_data": {
    "return_url": "支付成功后跳转地址",
    "quit_url": "用户取消支付跳转地址"
  }
}
```

**返回字段**:
- `pay_url`: 支付链接

**前端处理**:
```javascript
// 直接跳转到支付页面
window.location.href = response.data.pay_url;
```

---

## 💡 最佳实践

### 1. 订单号生成

商户订单号 `out_trade_no` 需要保证唯一性，建议格式：

```
{业务前缀}_{时间戳}_{随机数}
```

示例：`ORDER_20260416_123456`

### 2. 金额处理

所有金额单位为**分**，避免浮点数精度问题：

```javascript
// 1元 = 100分
const amount = 1.00 * 100; // 100
```

### 3. 回调处理

- 验证订单号和金额
- 做好幂等处理
- 快速响应（< 3秒）
- 异步处理业务逻辑

```python
@app.route('/callback', methods=['POST'])
def callback():
    data = request.get_json()
    
    # 1. 验证订单
    order = get_order(data['out_trade_no'])
    if order.amount != data['amount']:
        return {'code': 'ERROR', 'message': 'Amount mismatch'}
    
    # 2. 幂等处理
    if order.status == 'paid':
        return {'code': 'SUCCESS', 'message': 'OK'}
    
    # 3. 更新订单状态
    update_order_status(order.id, 'paid')
    
    # 4. 异步处理业务逻辑（发货、通知等）
    async_process.delay(order.id)
    
    # 5. 快速响应
    return {'code': 'SUCCESS', 'message': 'OK'}
```

### 4. 订单状态轮询

前端可以轮询订单状态，建议间隔 2-3 秒：

```javascript
const pollOrderStatus = (orderNo) => {
  const interval = setInterval(async () => {
    const order = await queryOrder(orderNo);
    
    if (order.status === 'paid') {
      clearInterval(interval);
      // 支付成功处理
    }
  }, 2000);
  
  // 5分钟后停止轮询
  setTimeout(() => clearInterval(interval), 5 * 60 * 1000);
};
```

---

## 📚 相关文档

- [快速开始指南](../guides/quickstart.md)
- [配置指南](../guides/configuration.md)
- [接入指南](../guides/integration.md)
- [常见问题 FAQ](../faq.md)

---

## 📞 技术支持

如有问题，请通过以下方式联系我们：

- **GitHub Issues**: https://github.com/yourusername/gopay/issues
- **邮件**: your-email@example.com

---

**最后更新**: 2026-04-16  
**API 版本**: v1.0
