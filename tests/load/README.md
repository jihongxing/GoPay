# GoPay 压力测试指南

本目录包含 GoPay 支付网关的压力测试工具和脚本。

## 测试目标

- 验证系统能否达到 10k+ QPS 的性能指标
- 评估不同并发场景下的响应时间
- 发现性能瓶颈和资源限制
- 验证系统在高负载下的稳定性

## 快速开始

### 前置条件

1. GoPay 服务已启动并运行在 `http://localhost:8080`
2. 数据库已正确配置并可访问
3. 已创建测试应用（app_id: `test_app_001`）

### 运行基础压力测试

```bash
# 进入测试目录
cd tests/load

# 运行所有压力测试
go test -v -timeout 10m

# 运行特定测试
go test -v -run TestLoadCheckout
go test -v -run TestLoadQuery

# 跳过长时间测试
go test -v -short
```

### 运行 10k+ QPS 测试

```bash
# 完整的 10k QPS 压力测试（需要 1-2 分钟）
go test -v -run TestLoad10kQPS -timeout 5m
```

## 测试场景

### 1. 下单接口压力测试 (TestLoadCheckout)

**测试参数:**
- 目标 QPS: 1,000
- 并发数: 100
- 持续时间: 30 秒
- 预热时间: 5 秒

**性能指标:**
- 实际 QPS ≥ 800 (目标的 80%)
- 错误率 < 1%
- P95 响应时间 < 100ms

### 2. 查询接口压力测试 (TestLoadQuery)

**测试参数:**
- 目标 QPS: 5,000
- 并发数: 200
- 持续时间: 30 秒
- 预热时间: 5 秒

**性能指标:**
- P95 响应时间 < 50ms
- 错误率 < 0.5%

### 3. 10k+ QPS 压力测试 (TestLoad10kQPS)

**测试参数:**
- 目标 QPS: 10,000
- 并发数: 500
- 持续时间: 60 秒
- 预热时间: 10 秒

**性能指标:**
- 实际 QPS ≥ 10,000
- 错误率 < 0.1%
- P95 响应时间 < 100ms

## 自定义压力测试

你可以通过修改 `LoadTestConfig` 来自定义测试参数：

```go
config := &LoadTestConfig{
    BaseURL:     "http://localhost:8080",
    AppID:       "your_app_id",
    AppSecret:   "your_app_secret",
    Concurrency: 200,              // 并发 goroutine 数量
    Duration:    60 * time.Second, // 测试持续时间
    RampUpTime:  10 * time.Second, // 预热时间
    TargetQPS:   5000,             // 目标 QPS
    RequestType: "checkout",       // 请求类型: checkout 或 query
}

result := RunLoadTest(config)
PrintResult(result)
```

## 测试结果解读

### 输出示例

```
========== 压力测试结果 ==========
总请求数:       30000
成功请求数:     29950
失败请求数:     50
测试持续时间:   30.5s
实际 QPS:       983.61
错误率:         0.17%

响应时间统计:
  平均:         45ms
  最小:         2ms
  最大:         250ms
  P50:          40ms
  P95:          85ms
  P99:          120ms
==================================
```

### 关键指标说明

- **实际 QPS**: 实际达到的每秒请求数，应接近目标 QPS
- **错误率**: 失败请求占比，生产环境应 < 0.1%
- **P95 响应时间**: 95% 的请求响应时间，反映大多数用户体验
- **P99 响应时间**: 99% 的请求响应时间，反映极端情况

## 性能优化建议

### 如果 QPS 未达标

1. **数据库连接池优化**
   ```go
   db.SetMaxOpenConns(100)
   db.SetMaxIdleConns(50)
   db.SetConnMaxLifetime(time.Hour)
   ```

2. **增加 Worker Pool 大小**
   ```go
   // 在 notify_service.go 中
   workerPool := pool.NewWorkerPool(200) // 增加到 200
   ```

3. **启用数据库索引**
   - 确保所有查询字段都有索引
   - 使用 `EXPLAIN ANALYZE` 分析慢查询

4. **考虑添加 Redis 缓存**
   - 缓存热点订单数据
   - 缓存渠道配置

### 如果响应时间过长

1. **检查数据库查询性能**
   ```bash
   # 查看慢查询日志
   tail -f /var/log/postgresql/postgresql.log | grep "duration"
   ```

2. **优化数据库查询**
   - 减少 JOIN 操作
   - 使用批量查询
   - 添加必要的索引

3. **减少外部 API 调用**
   - 使用异步处理
   - 增加超时控制

### 如果错误率过高

1. **检查错误日志**
   ```bash
   # 查看 GoPay 日志
   tail -f logs/gopay.log | grep ERROR
   ```

2. **常见错误原因**
   - 数据库连接池耗尽
   - 签名验证失败
   - 超时设置过短
   - 资源限制（文件描述符、内存）

3. **系统资源检查**
   ```bash
   # 检查文件描述符限制
   ulimit -n
   
   # 检查内存使用
   free -h
   
   # 检查 CPU 使用
   top
   ```

## 生产环境压力测试

### 注意事项

1. **使用独立测试环境**
   - 不要在生产环境直接压测
   - 使用与生产环境相同配置的测试环境

2. **逐步增加负载**
   - 从低 QPS 开始（如 100）
   - 逐步增加到目标 QPS
   - 观察系统指标变化

3. **监控系统指标**
   - CPU 使用率
   - 内存使用率
   - 数据库连接数
   - 网络带宽
   - 磁盘 I/O

4. **准备回滚方案**
   - 保存当前配置
   - 准备降级方案
   - 设置告警阈值

### 推荐测试流程

```bash
# 1. 基线测试（100 QPS，5 分钟）
go test -v -run TestLoadCheckout -timeout 10m

# 2. 中等负载测试（1000 QPS，10 分钟）
# 修改 TargetQPS 为 1000，Duration 为 10 分钟

# 3. 高负载测试（5000 QPS，15 分钟）
# 修改 TargetQPS 为 5000，Duration 为 15 分钟

# 4. 极限测试（10000+ QPS，30 分钟）
go test -v -run TestLoad10kQPS -timeout 35m
```

## 使用 wrk 进行压力测试（可选）

如果你更喜欢使用 wrk 工具：

```bash
# 安装 wrk
brew install wrk  # macOS
apt-get install wrk  # Ubuntu

# 运行压力测试
wrk -t12 -c400 -d30s --latency \
  -s scripts/checkout.lua \
  http://localhost:8080/api/v1/checkout
```

创建 `scripts/checkout.lua`:

```lua
wrk.method = "POST"
wrk.headers["Content-Type"] = "application/json"
wrk.headers["X-App-ID"] = "test_app_001"

request = function()
   local body = string.format([[{
     "app_id": "test_app_001",
     "out_trade_no": "LOAD_%s",
     "amount": 100,
     "subject": "测试商品",
     "channel": "wechat_native"
   }]], os.time() .. math.random(1000000))
   
   return wrk.format(nil, nil, nil, body)
end
```

## 故障排查

### 问题：连接被拒绝

```
Error: dial tcp 127.0.0.1:8080: connect: connection refused
```

**解决方案:**
1. 确认 GoPay 服务已启动
2. 检查端口是否正确
3. 检查防火墙设置

### 问题：签名验证失败

```
Error: status code: 401, body: {"code":"SIGNATURE_INVALID"}
```

**解决方案:**
1. 确认 app_id 和 app_secret 正确
2. 检查时间戳是否在有效范围内
3. 验证签名算法实现

### 问题：数据库连接池耗尽

```
Error: pq: sorry, too many clients already
```

**解决方案:**
1. 增加数据库最大连接数
2. 优化应用连接池配置
3. 检查是否有连接泄漏

## 相关文档

- [性能优化指南](../../docs/性能优化指南.md)
- [监控指南](../../docs/监控指南.md)
- [故障排查指南](../../docs/故障排查指南.md)
