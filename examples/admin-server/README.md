# 管理后台独立服务示例

这是一个独立运行管理后台的示例。

## 使用方法

```bash
# 设置环境变量
export DATABASE_URL="postgres://gopay:password@localhost:5432/gopay?sslmode=disable"

# 运行
go run examples/admin-server/main.go
```

## 访问

打开浏览器访问: http://localhost:8080/admin

## 说明

这个示例展示了如何将管理后台作为独立服务运行，与主支付服务分离部署。

优点:
- 独立部署，互不影响
- 可以单独扩展
- 安全隔离

缺点:
- 需要维护两个服务
- 数据库连接数增加
