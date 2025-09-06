# WWPlugin 库部署指南

## 📦 准备发布

### 1. 检查代码质量

```bash
# 检查代码格式
go fmt ./...

# 运行测试
go test ./...

# 检查依赖
go mod tidy

# 验证编译
go build .
```

### 2. 更新文档

确保以下文档是最新的：
- [ ] README.md - 包含最新的使用示例
- [ ] docs/user-guide.md - 详细的使用文档
- [ ] VERSION - 更新版本号

### 3. 版本标记

```bash
# 创建版本标签
git tag -a v1.0.0 -m "Release version 1.0.0"
git push origin v1.0.0
```

## 🚀 发布到 GitHub

### 1. 创建 GitHub 仓库

1. 访问 GitHub，创建新仓库 `wwplugin`
2. 设置仓库为公开（Public）
3. 不要初始化 README、.gitignore 或 LICENSE（我们已经有了）

### 2. 推送代码

```bash
# 初始化 Git 仓库
git init

# 添加所有文件
git add .

# 提交初始版本
git commit -m "Initial commit: WWPlugin v1.0.0

- 基于 gRPC 的多进程插件框架
- 支持双向通信和插件间调用
- 自适应端口分配和心跳监控
- 完整的示例和文档"

# 添加远程仓库（替换为你的实际 GitHub 用户名）
git remote add origin https://github.com/yourusername/wwplugin.git

# 推送到 GitHub
git branch -M main
git push -u origin main

# 推送标签
git push origin --tags
```

### 3. 配置 GitHub 仓库

在 GitHub 仓库页面：

1. **About 部分**：
   - Description: "高性能的 Go 插件框架，支持多进程架构和双向通信"
   - Website: 设置为你的文档链接
   - Topics: 添加标签 `go`, `plugin`, `grpc`, `microservices`, `framework`

2. **README 徽章**：
   - Go Report Card
   - GoDoc 文档链接
   - License 徽章

## 📚 发布到 Go 模块

### 1. 确保模块路径正确

在 `go.mod` 中：
```go
module github.com/yourusername/wwplugin
```

### 2. 发布版本

```bash
# 创建并推送版本标签
git tag v1.0.0
git push origin v1.0.0
```

Go 模块系统会自动从 GitHub 拉取你的模块。

### 3. 验证发布

```bash
# 在其他项目中测试安装
go get github.com/yourusername/wwplugin@v1.0.0
```

## 🔧 持续集成（可选）

### GitHub Actions 工作流

创建 `.github/workflows/ci.yml`：

```yaml
name: CI

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    runs-on: ubuntu-latest
    
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Go
      uses: actions/setup-go@v3
      with:
        go-version: 1.21
        
    - name: Build
      run: go build -v ./...
      
    - name: Test
      run: go test -v ./...
      
    - name: Vet
      run: go vet ./...
```

## 📝 使用指南

### 用户如何使用你的库

1. **安装**：
```bash
go get github.com/yourusername/wwplugin
```

2. **导入**：
```go
import "github.com/yourusername/wwplugin"
```

3. **快速开始**：
参考 README.md 中的示例代码

### 示例项目结构

```
myproject/
├── go.mod
├── main.go          # 主程序
├── plugins/
│   └── myplugin/
│       ├── go.mod   
│       └── main.go  # 插件程序
└── README.md
```

## 🎯 推广建议

### 1. 社区推广

- 在 Reddit r/golang 分享
- 在 Gopher Slack 频道介绍
- 写技术博客文章
- 参与 Go 相关的技术会议

### 2. 文档完善

- 添加更多示例
- 创建视频教程
- 写最佳实践指南
- 提供性能基准测试

### 3. 生态建设

- 创建插件模板项目
- 提供 IDE 插件支持
- 建立社区插件仓库
- 创建官方网站

## 🚨 注意事项

1. **模块路径**：确保 go.mod 中的模块路径与 GitHub 仓库路径一致
2. **版本标签**：使用语义化版本控制 (semver)
3. **向后兼容**：主版本号变更时要考虑向后兼容性
4. **文档同步**：保持代码和文档的同步更新
5. **许可证**：确认 MIT 许可证适合你的使用场景

## 📞 支持

发布后，准备好处理以下事项：

- 回答 GitHub Issues
- 审查 Pull Requests  
- 维护文档更新
- 发布安全更新
- 社区支持

## 🎉 发布清单

发布前确认：

- [ ] 代码质量检查通过
- [ ] 所有测试通过
- [ ] 文档完整且最新
- [ ] 示例代码可运行
- [ ] 版本号已更新
- [ ] License 文件存在
- [ ] .gitignore 配置正确
- [ ] README 包含安装和使用说明
- [ ] GitHub 仓库配置完成
- [ ] 版本标签已创建
- [ ] Go 模块可正常安装

完成以上步骤后，你的 WWPlugin 库就可以供全世界的 Go 开发者使用了！🚀