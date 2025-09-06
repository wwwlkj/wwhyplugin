# WWPlugin - 通用插件框架库

[![Go Report Card](https://goreportcard.com/badge/github.com/wwplugin/wwplugin)](https://goreportcard.com/report/github.com/wwplugin/wwplugin)
[![GoDoc](https://godoc.org/github.com/wwplugin/wwplugin?status.svg)](https://godoc.org/github.com/wwplugin/wwplugin)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

WWPlugin 是一个基于 gRPC 的高性能、跨平台插件框架，支持多进程架构和双向通信。

## 🚀 特性

- ✅ **多进程架构**: 每个插件运行在独立进程中，提高稳定性
- ✅ **双向通信**: 主程序和插件可以相互调用函数
- ✅ **插件间调用**: 支持插件之间通过主程序中介进行调用
- ✅ **消息推送**: 支持主程序向插件推送实时消息流
- ✅ **动态管理**: 动态加载、启动、停止和监控插件
- ✅ **心跳检测**: 自动检测插件状态，支持自动重启
- ✅ **gRPC 通信**: 高性能、类型安全的通信协议
- ✅ **自适应端口**: 自动分配可用端口，避免冲突
- ✅ **优雅关闭**: 支持优雅关闭和资源清理

## 📦 安装

```bash
go get github.com/wwplugin/wwplugin
```

## 🎯 快速开始

### 创建主程序

```go
package main

import (
    "log"
    "github.com/wwplugin/wwplugin"
)

func main() {
    // 创建插件主机
    host, err := wwplugin.NewPluginHost(&wwplugin.HostConfig{
        Port:      50051,
        DebugMode: true,
    })
    if err != nil {
        log.Fatal(err)
    }

    // 启动主机
    if err := host.Start(); err != nil {
        log.Fatal(err)
    }

    // 加载并启动插件
    plugin, err := host.LoadPlugin("./myplugin.exe")
    if err != nil {
        log.Fatal(err)
    }

    // 调用插件函数
    result, err := host.CallPluginFunction(plugin.ID, "MyFunction", params)
    if err != nil {
        log.Fatal(err)
    }

    // 等待
    host.Wait()
}
```

### 创建插件

```go
package main

import (
    "context"
    "github.com/wwplugin/wwplugin"
    "github.com/wwplugin/wwplugin/proto"
)

func main() {
    // 创建插件
    plugin := wwplugin.NewPlugin(&wwplugin.PluginConfig{
        Name:        "MyPlugin",
        Version:     "1.0.0",
        Description: "示例插件",
    })

    // 注册函数
    plugin.RegisterFunction("MyFunction", myFunction)

    // 启动插件
    if err := plugin.Start(); err != nil {
        log.Fatal(err)
    }
}

func myFunction(ctx context.Context, params []*proto.Parameter) (*proto.Parameter, error) {
    // 实现你的函数逻辑
    return &proto.Parameter{
        Name:  "result",
        Type:  proto.ParameterType_STRING,
        Value: "Hello from plugin!",
    }, nil
}
```

## 📖 文档

详细文档请参阅：
- [API 文档](https://godoc.org/github.com/wwplugin/wwplugin)
- [用户指南](docs/user-guide.md)
- [开发指南](docs/developer-guide.md)
- [示例代码](examples/)

## 🔧 架构

```
┌─────────────────┐     gRPC      ┌─────────────────┐
│   主程序         │ ◄──────────► │    插件 1        │
│  (Host Process) │               │ (Plugin Process) │
│                 │     gRPC      ├─────────────────┤
│ - 插件管理器     │ ◄──────────► │    插件 2        │
│ - gRPC 服务端   │               │ (Plugin Process) │
│ - gRPC 客户端   │     gRPC      ├─────────────────┤
│ - 消息推送      │ ◄──────────► │    插件 N        │
└─────────────────┘               │ (Plugin Process) │
                                  └─────────────────┘
```

## 📝 许可证

本项目基于 MIT 许可证开源 - 详见 [LICENSE](LICENSE) 文件。

## 🤝 贡献

欢迎贡献代码！请先阅读 [贡献指南](CONTRIBUTING.md)。

## ⭐ 支持

如果这个项目对你有帮助，请给我们一个 ⭐！