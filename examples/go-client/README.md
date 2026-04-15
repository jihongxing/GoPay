# GoPay Go 客户端示例

这是一个使用 Go 语言接入 GoPay 支付网关的完整示例。

## 功能特性

- ✅ 创建支付订单
- ✅ 查询订单状态
- ✅ 处理支付回调
- ✅ 错误处理
- ✅ 日志记录

## 快速开始

### 1. 安装依赖

```bash
go mod download
```

### 2. 配置环境变量

```bash
cp .env.example .env
# 编辑 .env 文件，填入你的配置
```

### 3. 运行示例

```bash
# 创建支付订单
go run main.go create

# 查询订单状态
go run main.go query ORDER_NO

# 启动回调服务器
go run main.go callback
```

## 代码示例

### 创建支付订单

```go
package main

import (
    "github.com/yourusername/gopay/examples/go-client/client"
)

func main() {
    // 创建客户端
    c := client.NewClient("http://localhost:8080", "your_app_id")
    
    // 创建订单
    resp, err := c.CreateOrder(&client.CreateOrderRequest{
        OutTradeNo: "ORDER_20260415_001",
        Amount:     100, // 单位：分
        Subject:    "测试商品",
        Channel:    "wechat_native",
        NotifyURL:  "https://your-domain.com/callback",
    })
    
    if err != nil {
        log.Fatal(err)
    }
    
    log.Printf("订单创建成功: %s", resp.OrderNo)
    log.Printf("支付链接: %s", resp.PayURL)
}
```

### 查询订单状态

```go
// 查询订单
order, err := c.QueryOrder("ORDER_NO")
if err != nil {
    log.Fatal(err)
}

log.Printf("订单状态: %s", order.Status)
log.Printf("支付金额: %d", order.Amount)
```

### 处理支付回调

```go
// 启动回调服务器
http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
    // 解析回调数据
    var callback client.CallbackData
    if err := json.NewDecoder(r.Body).Decode(&callback); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }
    
    // 验证签名（可选，GoPay 已验证）
    
    // 处理业务逻辑
    log.Printf("收到支付回调: 订单号=%s, 状态=%s", callback.OrderNo, callback.Status)
    
    // 返回成功
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"code": "SUCCESS"})
})

log.Fatal(http.ListenAndServe(":8081", nil))
```

## 项目结构

```
go-client/
├── main.go           # 主程序
├── client/           # 客户端库
│   ├── client.go     # 客户端实现
│   ├── types.go      # 类型定义
│   └── errors.go     # 错误处理
├── .env.example      # 配置示例
├── go.mod            # 依赖管理
└── README.md         # 本文档
```

## API 文档

详细的 API 文档请参考 [GoPay API 文档](../../docs/api/README.md)。

## 常见问题

### 1. 如何处理支付超时？

建议设置订单超时时间（如 30 分钟），超时后自动关闭订单。

### 2. 如何处理重复回调？

GoPay 会重试失败的回调，业务系统需要做好幂等处理。

### 3. 如何测试回调？

可以使用 ngrok 等工具将本地服务暴露到公网，或使用 GoPay 提供的测试工具。

## 许可证

MIT License
