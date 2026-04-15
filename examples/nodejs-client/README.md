# GoPay Node.js 客户端示例

这是一个使用 Node.js/TypeScript 接入 GoPay 支付网关的完整示例。

## 功能特性

- ✅ 创建支付订单
- ✅ 查询订单状态
- ✅ 处理支付回调
- ✅ TypeScript 类型支持
- ✅ Express.js 集成
- ✅ 错误处理

## 快速开始

### 1. 安装依赖

```bash
npm install
# 或
yarn install
```

### 2. 配置环境变量

```bash
cp .env.example .env
# 编辑 .env 文件，填入你的配置
```

### 3. 运行示例

```bash
# 开发模式
npm run dev

# 生产模式
npm run build
npm start

# 创建订单
npm run create

# 查询订单
npm run query ORDER_NO

# 启动回调服务器
npm run callback
```

## 代码示例

### 创建支付订单

```typescript
import { GopayClient } from './client';

const client = new GopayClient({
  baseURL: 'http://localhost:8080',
  appId: 'your_app_id',
});

async function createOrder() {
  try {
    const response = await client.createOrder({
      outTradeNo: `ORDER_${Date.now()}`,
      amount: 100, // 单位：分
      subject: '测试商品',
      channel: 'wechat_native',
      notifyUrl: 'https://your-domain.com/callback',
    });

    console.log('订单创建成功:', response.orderNo);
    console.log('支付链接:', response.payUrl);
  } catch (error) {
    console.error('创建订单失败:', error);
  }
}
```

### 查询订单状态

```typescript
async function queryOrder(orderNo: string) {
  try {
    const order = await client.queryOrder(orderNo);
    console.log('订单状态:', order.status);
    console.log('支付金额:', order.amount);
  } catch (error) {
    console.error('查询订单失败:', error);
  }
}
```

### 处理支付回调 (Express.js)

```typescript
import express from 'express';

const app = express();
app.use(express.json());

app.post('/callback', async (req, res) => {
  try {
    const callback = req.body;
    
    console.log('收到支付回调:', callback);
    
    // 处理业务逻辑
    if (callback.status === 'paid') {
      // 订单支付成功，进行后续处理
      console.log('订单支付成功:', callback.orderNo);
    }
    
    // 返回成功响应
    res.json({ code: 'SUCCESS', message: 'OK' });
  } catch (error) {
    console.error('处理回调失败:', error);
    res.status(500).json({ code: 'ERROR', message: error.message });
  }
});

app.listen(8081, () => {
  console.log('回调服务器启动在端口 8081');
});
```

## 项目结构

```
nodejs-client/
├── src/
│   ├── client.ts         # 客户端实现
│   ├── types.ts          # 类型定义
│   ├── create.ts         # 创建订单示例
│   ├── query.ts          # 查询订单示例
│   └── callback.ts       # 回调服务器
├── .env.example          # 配置示例
├── package.json          # 依赖管理
├── tsconfig.json         # TypeScript 配置
└── README.md             # 本文档
```

## API 文档

详细的 API 文档请参考 [GoPay API 文档](../../docs/api/README.md)。

## 许可证

MIT License
