# 贡献指南

欢迎为 WWPlugin 贡献代码！我们很高兴有你的参与。

## 🤝 如何贡献

### 1. 报告问题

如果你发现了bug或有功能建议：

1. 在提交issue前，请先搜索已有的issues
2. 使用issue模板提供尽可能详细的信息
3. 包含重现步骤和环境信息

### 2. 代码贡献

1. **Fork仓库**
   ```bash
   git clone https://github.com/yourname/wwplugin.git
   cd wwplugin
   ```

2. **创建功能分支**
   ```bash
   git checkout -b feature/your-feature-name
   ```

3. **进行更改**
   - 遵循代码规范
   - 添加或更新测试
   - 更新文档

4. **提交更改**
   ```bash
   git add .
   git commit -m "feat: 简洁描述你的更改"
   ```

5. **推送并创建PR**
   ```bash
   git push origin feature/your-feature-name
   ```

## 📝 代码规范

### Go 代码风格

- 使用 `gofmt` 格式化代码
- 遵循 [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- 使用有意义的变量和函数名
- 添加必要的注释，特别是公共接口

### 提交信息规范

使用约定式提交格式：

```
<类型>: <描述>

[可选的正文]

[可选的脚注]
```

类型包括：
- `feat`: 新功能
- `fix`: 修复bug
- `docs`: 文档更新
- `style`: 代码格式修改
- `refactor`: 重构
- `test`: 测试相关
- `chore`: 构建过程或辅助工具的变动

### 示例
```
feat: 添加插件热重载功能

- 实现插件文件监控
- 支持运行时重新加载插件
- 添加相关配置选项

关闭 #123
```

## 🧪 测试要求

### 单元测试
- 新功能必须包含单元测试
- 测试覆盖率应保持在80%以上
- 使用表驱动测试模式

### 集成测试
- 对于gRPC接口，提供集成测试
- 测试插件加载和通信流程

### 运行测试
```bash
# 运行所有测试
go test ./...

# 运行测试并查看覆盖率
go test -cover ./...

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## 📚 文档要求

- 公共API必须有清晰的文档注释
- 复杂功能需要提供使用示例
- 重要变更需要更新README和用户指南

## 🔍 代码审查流程

1. **自我检查**
   - 代码是否遵循规范
   - 测试是否通过
   - 文档是否完整

2. **PR检查清单**
   - [ ] 功能完整实现
   - [ ] 包含单元测试
   - [ ] 文档已更新
   - [ ] 代码风格规范
   - [ ] 无冲突需要解决

3. **审查过程**
   - 维护者会在48小时内响应
   - 根据反馈进行修改
   - 获得批准后合并

## 🚀 发布流程

### 版本号规范

使用语义化版本：`MAJOR.MINOR.PATCH`

- `MAJOR`: 不兼容的API更改
- `MINOR`: 向后兼容的功能增加
- `PATCH`: 向后兼容的bug修复

### 发布步骤

1. 更新版本号和CHANGELOG
2. 创建发布标签
3. 构建和测试
4. 发布到Go模块仓库

## ❓ 需要帮助？

- 查看[用户指南](docs/user-guide.md)和[开发指南](docs/developer-guide.md)
- 在issues中提问
- 参考现有代码示例

## 🙏 致谢

感谢所有贡献者的努力！你们的贡献让WWPlugin变得更好。

## 📄 许可证

通过贡献代码，你同意你的贡献将在MIT许可证下授权。