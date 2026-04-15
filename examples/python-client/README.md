# GoPay Python 客户端示例

这是一个使用 Python 接入 GoPay 支付网关的完整示例。

## 功能特性

- ✅ 创建支付订单
- ✅ 查询订单状态
- ✅ 处理支付回调
- ✅ 类型提示支持
- ✅ Flask/FastAPI 集成
- ✅ 异步支持

## 快速开始

### 1. 安装依赖

```bash
pip install -r requirements.txt
```

### 2. 配置环境变量

```bash
cp .env.example .env
# 编辑 .env 文件，填入你的配置
```

### 3. 运行示例

```bash
# 创建订单
python create_order.py

# 查询订单
python query_order.py ORDER_NO

# 启动 Flask 回调服务器
python flask_callback.py

# 启动 FastAPI 回调服务器
python fastapi_callback.py
```

## 代码示例

### 创建支付订单

```python
from gopay_client import GopayClient

# 创建客户端
client = GopayClient(
    base_url='http://localhost:8080',
    app_id='your_app_id'
)

# 创建订单
response = client.create_order(
    out_trade_no='ORDER_20260416_001',
    amount=100,  # 单位：分
    subject='测试商品',
    channel='wechat_native',
    notify_url='https://your-domain.com/callback'
)

print(f"订单创建成功: {response['order_no']}")
print(f"支付链接: {response.get('pay_url')}")
```

### 查询订单状态

```python
# 查询订单
order = client.query_order('ORDER_NO')

print(f"订单状态: {order['status']}")
print(f"支付金额: {order['amount']}")
```

### 处理支付回调 (Flask)

```python
from flask import Flask, request, jsonify

app = Flask(__name__)

@app.route('/callback', methods=['POST'])
def callback():
    data = request.get_json()
    
    print(f"收到支付回调: 订单号={data['order_no']}, 状态={data['status']}")
    
    # 处理业务逻辑
    if data['status'] == 'paid':
        # 订单支付成功
        print(f"订单支付成功: {data['order_no']}")
    
    return jsonify({'code': 'SUCCESS', 'message': 'OK'})

if __name__ == '__main__':
    app.run(port=8081)
```

### 处理支付回调 (FastAPI)

```python
from fastapi import FastAPI
from pydantic import BaseModel

app = FastAPI()

class CallbackData(BaseModel):
    order_no: str
    out_trade_no: str
    amount: int
    status: str
    channel: str

@app.post('/callback')
async def callback(data: CallbackData):
    print(f"收到支付回调: 订单号={data.order_no}, 状态={data.status}")
    
    # 处理业务逻辑
    if data.status == 'paid':
        print(f"订单支付成功: {data.order_no}")
    
    return {'code': 'SUCCESS', 'message': 'OK'}
```

## 项目结构

```
python-client/
├── gopay_client.py       # 客户端实现
├── types.py              # 类型定义
├── create_order.py       # 创建订单示例
├── query_order.py        # 查询订单示例
├── flask_callback.py     # Flask 回调服务器
├── fastapi_callback.py   # FastAPI 回调服务器
├── requirements.txt      # 依赖管理
├── .env.example          # 配置示例
└── README.md             # 本文档
```

## API 文档

详细的 API 文档请参考 [GoPay API 文档](../../docs/api/README.md)。

## 常见问题

### 1. 如何处理支付超时？

建议设置订单超时时间（如 30 分钟），超时后自动关闭订单。

### 2. 如何处理重复回调？

GoPay 会重试失败的回调，业务系统需要做好幂等处理。

### 3. 如何测试回调？

可以使用 ngrok 等工具将本地服务暴露到公网。

## 许可证

MIT License
