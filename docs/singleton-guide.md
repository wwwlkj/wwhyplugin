# Windows 单实例功能使用指南

## 📖 功能介绍

WWPlugin 框架提供了 Windows 下的单实例管理功能，防止程序多开，并支持将命令参数转发到已运行的实例。

## 🚀 核心特性

- ✅ **互斥体机制**: 使用 Windows 互斥体确保只有一个实例运行
- ✅ **命令转发**: 后续实例的命令参数会转发到首个实例
- ✅ **自动退出**: 后续实例发送命令后自动退出
- ✅ **进程间通信**: 使用 TCP 进行可靠的进程间通信
- ✅ **跨平台兼容**: 在非 Windows 平台提供占位实现

## 🛠️ 快速集成

### 1. 基本使用

```go
package main

import (
    "log"
    wwplugin "github.com/wwwlkj/wwhyplugin"
)

func main() {
    // 创建单实例配置
    config := wwplugin.DefaultSingletonConfig("MyApp")
    
    // 检查单实例（非首个实例会自动退出）
    isFirst, listener, err := wwplugin.CheckSingleInstance(config)
    if err != nil {
        log.Fatal(err)
    }
    
    if !isFirst {
        // 这行代码不会执行，因为非首个实例会自动退出
        return
    }
    
    // 设置资源清理
    defer wwplugin.CleanupSingleton()
    
    // 处理来自其他实例的命令
    if listener != nil {
        go handleCommands(listener)
    }
    
    // 你的主程序逻辑
    runMainApplication()
}
```

### 2. 处理命令转发

```go
func handleCommands(listener net.Listener) {
    for {
        conn, err := listener.Accept()
        if err != nil {
            continue
        }
        
        go func(conn net.Conn) {
            // 解析命令消息
            message, err := wwplugin.HandleIPCConnection(conn)
            if err != nil {
                return
            }
            
            // 处理命令
            handleReceivedCommand(message.Args)
        }(conn)
    }
}

func handleReceivedCommand(args []string) {
    if len(args) > 1 {
        switch args[1] {
        case "--load":
            // 处理加载命令
        case "--status":
            // 处理状态查询
        default:
            // 默认操作，如显示主窗口
        }
    }
}
```

## ⚙️ 配置选项

### SingletonConfig 结构体

```go
type SingletonConfig struct {
    MutexName    string // 互斥体名称，建议使用应用程序唯一标识
    IPCPort      int    // 进程间通信端口，0表示自动分配
    Timeout      int    // 通信超时时间（秒）
    RetryCount   int    // 重试次数
}
```

### 默认配置

```go
config := wwplugin.DefaultSingletonConfig("MyApp")
// 生成的配置:
// MutexName: "Global\\MyApp_Mutex"
// IPCPort: 0 (自动分配)
// Timeout: 5 秒
// RetryCount: 3 次
```

### 自定义配置

```go
config := &wwplugin.SingletonConfig{
    MutexName:  "Global\\MyUniqueApp_Mutex",
    IPCPort:    12345,
    Timeout:    10,
    RetryCount: 5,
}
```

## 📨 命令消息格式

### CommandMessage 结构体

```go
type CommandMessage struct {
    Args      []string `json:"args"`       // 命令行参数列表
    Pid       int      `json:"pid"`        // 发送进程的进程ID
    Timestamp int64    `json:"timestamp"`  // 消息发送时间戳
    WorkDir   string   `json:"work_dir"`   // 工作目录路径
}
```

## 🎯 使用场景示例

### 场景1: GUI 应用程序

```go
func main() {
    config := wwplugin.DefaultSingletonConfig("MyGUIApp")
    isFirst, listener, err := wwplugin.CheckSingleInstance(config)
    
    if err != nil {
        showErrorDialog(err.Error())
        return
    }
    
    defer wwplugin.CleanupSingleton()
    
    if listener != nil {
        go handleCommands(listener)
    }
    
    // 启动GUI应用
    startGUIApplication()
}

func handleReceivedCommand(args []string) {
    // 将主窗口置前显示
    bringMainWindowToFront()
    
    // 根据参数执行特定操作
    if len(args) > 1 && args[1] == "--open-file" {
        if len(args) > 2 {
            openFile(args[2])
        }
    }
}
```

### 场景2: 服务应用程序

```go
func main() {
    config := wwplugin.DefaultSingletonConfig("MyService")
    isFirst, listener, err := wwplugin.CheckSingleInstance(config)
    
    if err != nil {
        log.Fatal(err)
    }
    
    defer wwplugin.CleanupSingleton()
    
    if listener != nil {
        go handleServiceCommands(listener)
    }
    
    // 启动服务
    startService()
}

func handleServiceCommands(listener net.Listener) {
    for {
        conn, err := listener.Accept()
        if err != nil {
            continue
        }
        
        go func(conn net.Conn) {
            message, err := wwplugin.HandleIPCConnection(conn)
            if err != nil {
                return
            }
            
            // 处理服务管理命令
            handleServiceCommand(message.Args)
        }(conn)
    }
}
```

## 🔧 注意事项

### 1. 平台兼容性
- 单实例功能仅在 Windows 平台可用
- 非 Windows 平台会返回错误，但不影响程序继续运行

### 2. 权限要求
- 需要创建全局互斥体的权限
- 需要绑定本地 TCP 端口的权限

### 3. 资源清理
- 务必调用 `wwplugin.CleanupSingleton()` 清理资源
- 建议使用 `defer` 确保资源清理

### 4. 错误处理
- 检查 `CheckSingleInstance` 的返回错误
- 处理网络通信可能的超时和失败

## 📝 完整示例

参考项目中的 `examples/singleton_example.go` 文件，包含了完整的集成示例。

## ❓ 常见问题

### Q: 如何自定义互斥体名称？
A: 使用 `SingletonConfig` 结构体自定义 `MutexName` 字段，建议包含应用程序的唯一标识符。

### Q: 如何处理端口冲突？
A: 设置 `IPCPort` 为 0 使用自动端口分配，或指定一个应用程序专用的端口号。

### Q: 非 Windows 平台如何处理？
A: 框架会返回错误提示，你可以选择忽略错误继续运行，或实现平台特定的单实例逻辑。

### Q: 如何调试单实例功能？
A: 启用详细日志记录，检查临时目录中的端口文件，使用网络调试工具监控 TCP 连接。