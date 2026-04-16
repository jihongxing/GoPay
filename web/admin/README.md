# GoPay 管理后台前端

## 概述

GoPay 管理后台提供了一个简洁的 Web 界面，用于管理支付订单、查看对账报告和操作日志。

## 技术栈

- **HTML5 + CSS3**: 响应式布局
- **原生 JavaScript**: 无框架依赖，轻量级
- **Chart.js**: 数据可视化
- **Gin HTML Templates**: 服务端模板渲染

## 目录结构

```
web/admin/
├── static/
│   ├── css/
│   │   └── main.css          # 主样式文件
│   └── js/
│       ├── api.js            # API 封装
│       ├── utils.js          # 工具函数
│       └── app.js            # 应用初始化
└── templates/
    ├── layout.html           # 布局模板
    ├── dashboard.html        # 数据概览页面
    ├── orders.html           # 订单管理页面
    ├── reconciliation.html   # 对账报告页面
    └── logs.html             # 操作日志页面
```

## 功能模块

### 1. 数据概览 (Dashboard)

- **统计卡片**: 今日订单、成功订单、失败订单、通知失败数
- **订单趋势图**: 最近 7 天订单数量趋势
- **渠道分布图**: 微信支付和支付宝的订单分布
- **最近失败订单**: 显示最近 5 条失败订单

**访问路径**: `/admin`

### 2. 订单管理 (Orders)

- **订单列表**: 显示所有失败订单和通知失败的订单
- **筛选功能**: 
  - 支付渠道（微信/支付宝）
  - 订单状态（待支付/已支付/失败）
  - 通知状态（待通知/已通知/通知失败）
  - 日期范围
- **搜索功能**: 通过商户订单号精确查询
- **订单详情**: 查看订单完整信息
- **重试功能**: 
  - 单个订单重试
  - 批量订单重试
- **分页**: 每页显示 20 条记录

**访问路径**: `/admin/orders`

### 3. 对账报告 (Reconciliation)

- **报告列表**: 显示所有对账报告
- **筛选功能**:
  - 支付渠道
  - 对账状态（成功/有差异）
  - 日期范围
- **报告详情**:
  - 对账汇总信息
  - 长款明细（外部有但内部无）
  - 短款明细（内部有但外部无）
  - 金额不匹配明细
- **下载功能**: 下载对账报告 CSV 文件

**访问路径**: `/admin/reconciliation`

### 4. 操作日志 (Logs)

- **日志列表**: 显示所有操作记录
- **筛选功能**:
  - 操作类型（重试订单/批量重试/重试回调等）
  - 操作人
  - 日期范围
- **日志详情**: 操作时间、操作人、IP 地址、User Agent

**访问路径**: `/admin/logs`

## API 接口

### 统计接口

- `GET /admin/stats` - 获取统计数据
- `GET /admin/stats/orders?days=7` - 获取订单统计
- `GET /admin/stats/notifications?days=7` - 获取通知统计

### 订单接口

- `GET /admin/orders/failed` - 获取失败订单列表
- `GET /admin/orders/search?out_trade_no=xxx` - 搜索订单
- `GET /admin/orders/:order_no` - 获取订单详情
- `POST /admin/orders/:order_no/retry` - 重试订单通知
- `POST /admin/orders/batch-retry` - 批量重试

### 对账接口

- `GET /admin/reconciliation/reports` - 获取对账报告列表
- `GET /admin/reconciliation/:id` - 获取对账报告详情
- `GET /admin/reconciliation/:id/download` - 下载对账报告

## 使用说明

### 启动服务

```bash
# 启动 GoPay 服务
go run cmd/server/main.go
```

### 访问后台

在浏览器中访问: `http://localhost:8080/admin`

### 订单管理

1. **查看失败订单**
   - 进入"订单管理"页面
   - 使用筛选器按渠道、状态、日期筛选
   - 点击"详情"查看订单完整信息

2. **重试失败通知**
   - 在订单列表中找到通知失败的订单
   - 点击"重试"按钮
   - 或勾选多个订单，点击"批量重试"

3. **搜索订单**
   - 点击"搜索订单"按钮
   - 输入商户订单号
   - 查看订单详情

### 对账报告

1. **查看对账报告**
   - 进入"对账报告"页面
   - 使用筛选器按渠道、状态、日期筛选
   - 点击"查看详情"查看差异明细

2. **下载报告**
   - 在报告详情页面
   - 点击"下载报告"按钮
   - 保存 CSV 文件

### 操作日志

1. **查看操作记录**
   - 进入"操作日志"页面
   - 使用筛选器按操作类型、操作人、日期筛选
   - 查看操作详情

## 样式定制

### 修改主题色

编辑 `static/css/main.css`:

```css
/* 主色调 */
.btn-primary {
    background: #3498db;  /* 修改为你的主色 */
}

/* 侧边栏背景 */
.sidebar {
    background: #2c3e50;  /* 修改为你的侧边栏颜色 */
}
```

### 修改统计卡片

编辑 `templates/dashboard.html`:

```html
<div class="stat-card">
    <div class="stat-icon">📦</div>  <!-- 修改图标 -->
    <div class="stat-info">
        <div class="stat-label">今日订单</div>
        <div class="stat-value" id="today-orders">-</div>
    </div>
</div>
```

## 浏览器兼容性

- Chrome 90+
- Firefox 88+
- Safari 14+
- Edge 90+

## 性能优化

1. **CDN 加速**: Chart.js 使用 CDN 加载
2. **按需加载**: 图表仅在需要时初始化
3. **防抖节流**: 搜索和筛选使用防抖处理
4. **分页加载**: 大数据量分页显示

## 安全考虑

1. **XSS 防护**: 所有用户输入都经过转义
2. **CSRF 防护**: 建议添加 CSRF Token
3. **权限控制**: 建议添加登录认证
4. **操作日志**: 记录所有敏感操作

## 待实现功能

- [ ] 用户登录认证
- [ ] 权限管理（角色/权限）
- [ ] 实时数据推送（WebSocket）
- [ ] 导出功能（Excel）
- [ ] 高级搜索（多条件组合）
- [ ] 操作审计日志持久化
- [ ] 暗黑模式

## 故障排查

### 页面无法加载

1. 检查服务是否启动: `curl http://localhost:8080/admin`
2. 检查静态文件路径是否正确
3. 查看浏览器控制台错误信息

### API 请求失败

1. 打开浏览器开发者工具 -> Network
2. 查看请求状态码和响应内容
3. 检查后端日志

### 图表不显示

1. 检查 Chart.js 是否加载成功
2. 查看浏览器控制台是否有 JavaScript 错误
3. 确认 canvas 元素存在

## 贡献指南

欢迎提交 Issue 和 Pull Request！

## 许可证

MIT License
