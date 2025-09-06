# WWPlugin 开发指南

## 🏗️ 开发环境搭建

### 环境要求
- Go 1.21+
- Protocol Buffers 编译器
- gRPC 工具链

### 安装依赖
```bash
go mod tidy
```

## 📦 项目结构

```
wwplugin/
├── wwplugin.go          # 主入口文件
├── types.go             # 类型定义
├── host.go              # 插件主机实现
├── plugin.go            # 插件实现
├── host_service.go      # 主机gRPC服务
├── proto/               # Protocol Buffers定义
├── examples/            # 示例代码
└── docs/               # 文档目录
```

## 🔧 开发工作流

### 1. 修改Protocol Buffers

如果需要修改gRPC接口：

```bash
# 编译proto文件
protoc --go_out=. --go_grpc_out=. proto/plugin.proto
```

### 2. 开发主机应用

```go
package main

import (
    "github.com/wwwlkj/wwhyplugin"
)

func main() {
    host, err := wwplugin.NewPluginHost(wwplugin.DefaultHostConfig())
    if err != nil {
        panic(err)
    }
    
    host.Start()
    host.Wait()
}
```

### 3. 开发插件

```go
package main

import (
    "context"
    "github.com/wwwlkj/wwhyplugin"
    "github.com/wwplugin/wwplugin/proto"
)

func main() {
    plugin := wwplugin.NewPlugin(wwplugin.DefaultPluginConfig(
        "MyPlugin", "1.0.0", "插件描述",
    ))
    
    plugin.RegisterFunction("MyFunction", myFunction)
    plugin.Start()
}

func myFunction(ctx context.Context, params []*proto.Parameter) (*proto.Parameter, error) {
    // 实现插件功能
    return &proto.Parameter{
        Name:  "result",
        Type:  proto.ParameterType_STRING,
        Value: "Hello from plugin!",
    }, nil
}
```

## 🧪 测试

### 单元测试
```bash
go test ./...
```

### 集成测试
```bash
# 构建示例
go build -o examples/host/host.exe examples/host/main.go
go build -o examples/sample_plugin/plugin.exe examples/sample_plugin/main.go

# 运行测试
./examples/host/host.exe
```

## 📋 开发最佳实践

### 1. 错误处理
- 所有gRPC调用都应该有超时设置
- 实现适当的重试机制
- 记录详细的错误日志

### 2. 插件设计
- 插件应该是无状态的
- 实现优雅关闭机制
- 提供健康检查端点

### 3. 性能优化
- 使用连接池管理gRPC连接
- 实现参数缓存机制
- 监控内存使用情况

## 🔍 调试技巧

### 1. 启用调试日志
```go
config := wwplugin.DefaultHostConfig()
config.DebugMode = true
```

### 2. 使用gRPC调试工具
```bash
grpcurl -plaintext localhost:50051 list
```

### 3. 监控插件状态
```go
plugins := host.GetAllPlugins()
for _, plugin := range plugins {
    fmt.Printf("Plugin %s: %s\n", plugin.ID, plugin.Status)
}
```

## 🚀 部署

参考 [DEPLOYMENT.md](../DEPLOYMENT.md) 文档。