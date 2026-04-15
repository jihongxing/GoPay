# 贡献指南

感谢你考虑为 GoPay 做出贡献！我们欢迎所有形式的贡献，包括但不限于：

- 报告 Bug
- 提出新功能建议
- 提交代码修复
- 完善文档
- 分享使用经验

---

## 📋 行为准则

在参与本项目之前，请阅读并遵守我们的 [行为准则](CODE_OF_CONDUCT.md)。

---

## 🐛 报告 Bug

### 在提交 Bug 之前

1. **检查现有 Issue**: 确保该 Bug 尚未被报告
2. **使用最新版本**: 确认 Bug 在最新版本中仍然存在
3. **收集信息**: 准备好复现步骤和相关日志

### 如何提交 Bug 报告

创建一个新的 Issue，并包含以下信息：

- **标题**: 简洁明了地描述问题
- **环境信息**:
  - GoPay 版本
  - Go 版本
  - 操作系统
  - 数据库版本
- **复现步骤**: 详细的步骤说明
- **期望行为**: 你期望发生什么
- **实际行为**: 实际发生了什么
- **日志**: 相关的错误日志或堆栈跟踪
- **截图**: 如果适用，提供截图

**示例**:

```markdown
## Bug 描述
微信支付回调验签失败

## 环境信息
- GoPay 版本: v1.0.0
- Go 版本: 1.21.0
- 操作系统: Ubuntu 22.04
- 数据库: PostgreSQL 15.2

## 复现步骤
1. 配置微信支付渠道
2. 创建支付订单
3. 模拟微信回调
4. 查看日志

## 期望行为
回调验签成功，订单状态更新

## 实际行为
回调验签失败，返回 400 错误

## 日志
```
ERROR: signature verification failed
```

---

## 💡 提出新功能

### 在提交功能建议之前

1. **检查现有 Issue**: 确保该功能尚未被提出
2. **考虑适用性**: 该功能是否适合大多数用户
3. **准备说明**: 清楚地描述功能的用途和价值

### 如何提交功能建议

创建一个新的 Issue，并包含以下信息：

- **标题**: 简洁明了地描述功能
- **问题描述**: 当前存在什么问题或限制
- **解决方案**: 你建议的解决方案
- **替代方案**: 其他可能的解决方案
- **使用场景**: 该功能的典型使用场景
- **优先级**: 你认为的优先级（高/中/低）

---

## 🔧 提交代码

### 开发流程

1. **Fork 仓库**: 点击右上角的 Fork 按钮
2. **克隆仓库**: `git clone https://github.com/your-username/gopay.git`
3. **创建分支**: `git checkout -b feature/your-feature-name`
4. **开发代码**: 编写代码并确保通过测试
5. **提交代码**: `git commit -m "feat: add your feature"`
6. **推送分支**: `git push origin feature/your-feature-name`
7. **创建 PR**: 在 GitHub 上创建 Pull Request

### 代码规范

#### Go 代码规范

- 遵循 [Effective Go](https://go.dev/doc/effective_go) 指南
- 使用 `gofmt` 格式化代码
- 使用 `golint` 检查代码质量
- 使用 `go vet` 进行静态分析

```bash
# 格式化代码
make fmt

# 代码检查
make lint

# 静态分析
make vet
```

#### 命名规范

- **包名**: 小写，简短，有意义（如 `handler`, `service`）
- **文件名**: 小写，下划线分隔（如 `channel_manager.go`）
- **函数名**: 驼峰命名（如 `CreateOrder`, `handleWebhook`）
- **变量名**: 驼峰命名（如 `orderNo`, `channelConfig`）
- **常量名**: 驼峰命名或全大写（如 `MaxRetryCount`, `DEFAULT_TIMEOUT`）

#### 注释规范

- 所有导出的函数、类型、常量必须有注释
- 注释应该说明"为什么"而不是"是什么"
- 使用完整的句子，以句号结尾

```go
// CreateOrder creates a new payment order and returns the order details.
// It validates the request, creates a provider, and calls the channel API.
func CreateOrder(ctx context.Context, req *CreateOrderRequest) (*CreateOrderResponse, error) {
    // ...
}
```

#### 错误处理

- 使用自定义错误类型
- 提供有意义的错误信息
- 不要忽略错误

```go
// 好的做法
if err != nil {
    return nil, errors.NewInternalError("failed to create order", err)
}

// 不好的做法
if err != nil {
    return nil, err
}
```

### 提交信息规范

使用 [Conventional Commits](https://www.conventionalcommits.org/) 规范：

```
<type>(<scope>): <subject>

<body>

<footer>
```

**类型 (type)**:
- `feat`: 新功能
- `fix`: Bug 修复
- `docs`: 文档更新
- `style`: 代码格式（不影响代码运行）
- `refactor`: 重构（既不是新功能也不是 Bug 修复）
- `perf`: 性能优化
- `test`: 测试相关
- `chore`: 构建过程或辅助工具的变动

**示例**:

```bash
feat(alipay): add alipay wap payment support

Add support for Alipay mobile website payment (Wap).
This includes:
- New WapProvider implementation
- Configuration handling
- Integration tests

Closes #123
```

### 测试要求

- 所有新功能必须包含单元测试
- 测试覆盖率不应降低
- 确保所有测试通过

```bash
# 运行测试
make test

# 查看覆盖率
make test-coverage
```

### Pull Request 检查清单

在提交 PR 之前，请确保：

- [ ] 代码遵循项目的代码规范
- [ ] 所有测试通过
- [ ] 添加了必要的测试
- [ ] 更新了相关文档
- [ ] 提交信息符合规范
- [ ] PR 描述清晰，说明了改动内容
- [ ] 关联了相关的 Issue

### Pull Request 模板

```markdown
## 改动描述
简要描述这个 PR 做了什么

## 改动类型
- [ ] Bug 修复
- [ ] 新功能
- [ ] 重构
- [ ] 文档更新
- [ ] 其他

## 相关 Issue
Closes #123

## 测试
描述你如何测试这些改动

## 截图（如果适用）
添加截图帮助说明改动

## 检查清单
- [ ] 代码遵循项目规范
- [ ] 所有测试通过
- [ ] 添加了必要的测试
- [ ] 更新了相关文档
```

---

## 📚 文档贡献

文档同样重要！你可以通过以下方式贡献文档：

- 修复文档中的错误
- 改进文档的清晰度
- 添加示例代码
- 翻译文档

文档位于 `docs/` 目录，使用 Markdown 格式。

---

## 🔍 代码审查

所有的 Pull Request 都需要经过代码审查。审查者会关注：

- **代码质量**: 代码是否清晰、可维护
- **测试覆盖**: 是否有足够的测试
- **性能影响**: 是否影响性能
- **安全性**: 是否存在安全隐患
- **文档**: 是否更新了相关文档

### 如何响应审查意见

- 认真对待每一条审查意见
- 如果不同意，礼貌地说明理由
- 及时更新代码
- 标记已解决的评论

---

## 🎯 开发环境搭建

### 1. 安装依赖

```bash
# Go 1.21+
# Docker & Docker Compose
# PostgreSQL 15+
```

### 2. 克隆项目

```bash
git clone https://github.com/yourusername/gopay.git
cd gopay
```

### 3. 启动数据库

```bash
docker-compose up -d postgres
```

### 4. 配置环境变量

```bash
cp .env.example .env
# 编辑 .env 文件
```

### 5. 运行迁移

```bash
make migrate
```

### 6. 启动服务

```bash
make run
```

### 7. 运行测试

```bash
make test
```

详细说明请参考 [开发环境搭建](docs/development/setup.md)。

---

## 🛠️ 常用命令

```bash
# 格式化代码
make fmt

# 代码检查
make lint

# 运行测试
make test

# 查看覆盖率
make test-coverage

# 构建二进制
make build

# 清理构建产物
make clean

# 启动数据库
make db-up

# 停止数据库
make db-down

# 运行迁移
make migrate
```

---

## 📞 获取帮助

如果你在贡献过程中遇到问题，可以通过以下方式获取帮助：

- **GitHub Issues**: 提出问题
- **GitHub Discussions**: 参与讨论
- **邮件**: your-email@example.com

---

## 🎉 贡献者

感谢所有为 GoPay 做出贡献的人！

<!-- ALL-CONTRIBUTORS-LIST:START -->
<!-- ALL-CONTRIBUTORS-LIST:END -->

---

## 📜 许可证

通过贡献代码，你同意你的贡献将在 [MIT License](LICENSE) 下发布。

---

再次感谢你的贡献！🙏
