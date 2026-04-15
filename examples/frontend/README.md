# GoPay React 前端示例

这是一个使用 React 接入 GoPay 支付网关的完整示例。

## 功能特性

- ✅ 微信扫码支付组件
- ✅ 支付宝支付组件
- ✅ 支付状态轮询
- ✅ 二维码展示
- ✅ TypeScript 支持
- ✅ 响应式设计

## 快速开始

### 1. 安装依赖

```bash
npm install
# 或
yarn install
```

### 2. 配置环境变量

```bash
cp .env.example .env.local
# 编辑 .env.local 文件，填入你的配置
```

### 3. 运行示例

```bash
# 开发模式
npm run dev

# 生产构建
npm run build
npm run preview
```

## 组件使用

### 微信扫码支付

```tsx
import { WechatPayment } from './components/WechatPayment';

function App() {
  return (
    <WechatPayment
      appId="your_app_id"
      amount={100}
      subject="测试商品"
      onSuccess={(orderNo) => {
        console.log('支付成功:', orderNo);
      }}
      onError={(error) => {
        console.error('支付失败:', error);
      }}
    />
  );
}
```

### 支付宝支付

```tsx
import { AlipayPayment } from './components/AlipayPayment';

function App() {
  return (
    <AlipayPayment
      appId="your_app_id"
      amount={100}
      subject="测试商品"
      channel="alipay_qr" // 或 alipay_wap
      onSuccess={(orderNo) => {
        console.log('支付成功:', orderNo);
      }}
      onError={(error) => {
        console.error('支付失败:', error);
      }}
    />
  );
}
```

## 项目结构

```
frontend/
├── src/
│   ├── components/
│   │   ├── WechatPayment.tsx    # 微信支付组件
│   │   ├── AlipayPayment.tsx    # 支付宝支付组件
│   │   └── QRCode.tsx           # 二维码组件
│   ├── services/
│   │   └── gopay.ts             # GoPay API 服务
│   ├── types/
│   │   └── index.ts             # 类型定义
│   ├── App.tsx                  # 主应用
│   └── main.tsx                 # 入口文件
├── .env.example                 # 配置示例
├── package.json                 # 依赖管理
├── tsconfig.json                # TypeScript 配置
├── vite.config.ts               # Vite 配置
└── README.md                    # 本文档
```

## API 文档

详细的 API 文档请参考 [GoPay API 文档](../../docs/api/README.md)。

## 许可证

MIT License
