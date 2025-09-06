# WWPlugin 使用指南

## 目录

- [快速开始](#快速开始)
- [主机配置](#主机配置)
- [插件开发](#插件开发)
- [插件间调用](#插件间调用)
- [消息系统](#消息系统)
- [高级特性](#高级特性)

## 快速开始

### 安装

```bash
go get github.com/yourname/wwplugin
```

### 创建主机程序

```go
package main

import (
    "log"
    "github.com/yourname/wwplugin"
)

func main() {
    // 创建主机配置
    config := wwplugin.DefaultHostConfig()
    config.DebugMode = true
    
    // 创建插件主机
    host, err := wwplugin.NewPluginHost(config)
    if err != nil {
        log.Fatal(err)
    }

    // 启动主机
    if err := host.Start(); err != nil {
        log.Fatal(err)
    }

    // 加载插件
    plugin, err := host.StartPluginByPath("./myplugin.exe")
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("插件已启动: %s", plugin.ID)

    // 等待退出信号
    host.Wait()
}
```

### 创建插件

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "github.com/yourname/wwplugin"
    "github.com/yourname/wwplugin/proto"
)

func main() {
    // 支持 --info 参数
    if len(os.Args) > 1 && os.Args[1] == "--info" {
        plugin := createPlugin()
        plugin.StartWithInfo()
        return
    }

    // 正常启动
    plugin := createPlugin()
    if err := plugin.Start(); err != nil {
        log.Fatal(err)
    }
}

func createPlugin() *wwplugin.Plugin {
    config := wwplugin.DefaultPluginConfig(
        "MyPlugin",
        "1.0.0", 
        "我的示例插件",
    )
    
    plugin := wwplugin.NewPlugin(config)
    
    // 注册函数
    plugin.RegisterFunction("Hello", helloFunction)
    
    return plugin
}

func helloFunction(ctx context.Context, params []*proto.Parameter) (*proto.Parameter, error) {
    name := "World"
    if len(params) > 0 {
        name = params[0].Value
    }
    
    return &proto.Parameter{
        Name:  "greeting",
        Type:  proto.ParameterType_STRING,
        Value: fmt.Sprintf("Hello, %s!", name),
    }, nil
}
```

## 主机配置

### 配置选项

```go
config := &wwplugin.HostConfig{
    Port:              50051,              // 指定端口（0表示自动分配）
    PortRange:         []int{50051, 50100}, // 端口范围
    DebugMode:         true,               // 调试模式
    LogLevel:          "info",             // 日志级别
    LogDir:            "./logs",           // 日志目录
    HeartbeatInterval: 10 * time.Second,   // 心跳间隔
    MaxHeartbeatMiss:  3,                  // 最大心跳丢失次数
    AutoRestartPlugin: true,               // 自动重启崩溃的插件
}

host, err := wwplugin.NewPluginHost(config)
```

### 注册主机函数

```go
// 注册自定义主机函数
host.RegisterHostFunction("MyHostFunction", func(ctx context.Context, params []*proto.Parameter) (*proto.Parameter, error) {
    return &proto.Parameter{
        Name:  "result",
        Type:  proto.ParameterType_STRING,
        Value: "来自主机的响应",
    }, nil
})
```

## 插件开发

### 插件函数

插件函数必须符合 `PluginFunction` 签名：

```go
type PluginFunction func(ctx context.Context, params []*proto.Parameter) (*proto.Parameter, error)
```

### 参数处理

```go
func myFunction(ctx context.Context, params []*proto.Parameter) (*proto.Parameter, error) {
    // 检查参数数量
    if len(params) < 1 {
        return nil, fmt.Errorf("需要至少1个参数")
    }
    
    // 获取参数值
    input := params[0].Value
    
    // 类型转换示例
    switch params[0].Type {
    case proto.ParameterType_INT:
        value, err := strconv.Atoi(input)
        if err != nil {
            return nil, fmt.Errorf("无效的整数: %v", err)
        }
        // 使用 value
    case proto.ParameterType_FLOAT:
        value, err := strconv.ParseFloat(input, 64)
        if err != nil {
            return nil, fmt.Errorf("无效的浮点数: %v", err)
        }
        // 使用 value
    case proto.ParameterType_STRING:
        // 直接使用 input
    }
    
    // 返回结果
    return &proto.Parameter{
        Name:  "result",
        Type:  proto.ParameterType_STRING,
        Value: "处理结果",
    }, nil
}
```

### 调用主机函数

```go
func callHostExample(plugin *wwplugin.Plugin) wwplugin.PluginFunction {
    return func(ctx context.Context, params []*proto.Parameter) (*proto.Parameter, error) {
        // 调用主机函数
        resp, err := plugin.CallHostFunction("GetSystemTime", []*proto.Parameter{})
        if err != nil {
            return nil, err
        }
        
        if !resp.Success {
            return nil, fmt.Errorf("主机函数调用失败: %s", resp.Message)
        }
        
        return &proto.Parameter{
            Name:  "time_from_host",
            Type:  proto.ParameterType_STRING,
            Value: resp.Result.Value,
        }, nil
    }
}
```

## 插件间调用

### 基本用法

```go
func interPluginCall(plugin *wwplugin.Plugin) wwplugin.PluginFunction {
    return func(ctx context.Context, params []*proto.Parameter) (*proto.Parameter, error) {
        targetPluginID := "target-plugin-id"
        functionName := "TargetFunction"
        
        // 准备参数
        callParams := []*proto.Parameter{
            {
                Name:  "input",
                Type:  proto.ParameterType_STRING,
                Value: "来自其他插件的参数",
            },
        }
        
        // 调用其他插件
        resp, err := plugin.CallOtherPlugin(targetPluginID, functionName, callParams)
        if err != nil {
            return nil, fmt.Errorf("插件间调用失败: %v", err)
        }
        
        if !resp.Success {
            return nil, fmt.Errorf("目标插件返回错误: %s", resp.Message)
        }
        
        return &proto.Parameter{
            Name:  "inter_plugin_result",
            Type:  proto.ParameterType_STRING,
            Value: fmt.Sprintf("来自 %s 的结果: %s", targetPluginID, resp.Result.Value),
        }, nil
    }
}
```

### 动态获取插件列表

```go
// 在主机中获取所有插件
plugins := host.GetAllPlugins()
for _, plugin := range plugins {
    if plugin.Status == wwplugin.StatusRunning {
        log.Printf("运行中的插件: %s (%s)", plugin.Name, plugin.ID)
    }
}
```

## 消息系统

### 发送消息到插件

```go
// 发送单个消息
resp, err := host.SendMessageToPlugin(
    "plugin-id",
    "notification",
    "消息内容",
    map[string]string{"priority": "high"},
)

// 广播消息到所有插件
results := host.BroadcastMessage(
    "system_update",
    "系统更新通知",
    nil,
)
```

### 处理消息（插件端）

```go
plugin.SetMessageHandler(func(msg *proto.MessageRequest) {
    switch msg.MessageType {
    case "notification":
        log.Printf("收到通知: %s", msg.Content)
    case "command":
        log.Printf("收到命令: %s", msg.Content)
        // 执行相应操作
    case "data":
        log.Printf("收到数据: %s", msg.Content)
        // 处理数据
    default:
        log.Printf("未知消息类型: %s", msg.MessageType)
    }
})
```

## 高级特性

### 插件信息查询

```go
// 不加载插件获取信息
info, err := host.GetPluginInfo("./plugin.exe")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("插件名称: %s\n", info.Name)
fmt.Printf("版本: %s\n", info.Version)
fmt.Printf("描述: %s\n", info.Description)
fmt.Printf("功能: %v\n", info.Functions)
```

### 插件状态监控

```go
// 获取插件状态
plugin, exists := host.GetPlugin("plugin-id")
if exists {
    fmt.Printf("插件状态: %s\n", plugin.Status)
    fmt.Printf("启动时间: %s\n", plugin.StartTime)
    fmt.Printf("最后心跳: %s\n", plugin.LastHeartbeat)
}
```

### 优雅关闭

```go
// 主机会自动处理优雅关闭
// 也可以手动停止特定插件
err := host.StopPlugin("plugin-id")
if err != nil {
    log.Printf("停止插件失败: %v", err)
}

// 停止所有插件
host.StopAllPlugins()

// 停止主机（会自动停止所有插件）
host.Stop()
```

## 最佳实践

1. **错误处理**: 总是检查函数调用的错误返回值
2. **参数验证**: 在插件函数中验证输入参数
3. **日志记录**: 使用适当的日志级别记录重要事件
4. **资源清理**: 确保插件正确实现关闭逻辑
5. **超时设置**: 为长时间运行的操作设置适当的超时
6. **状态检查**: 在调用插件前检查插件状态
7. **错误恢复**: 实现适当的错误恢复机制

## 故障排除

### 常见问题

1. **端口冲突**: 使用端口范围或自动分配避免冲突
2. **连接超时**: 检查网络配置和防火墙设置
3. **插件崩溃**: 启用自动重启功能
4. **心跳超时**: 调整心跳间隔和超时设置
5. **函数未找到**: 确保函数已正确注册

### 调试技巧

1. 启用调试模式: `config.DebugMode = true`
2. 降低日志级别: `config.LogLevel = "debug"`
3. 检查插件状态: 使用 `GetAllPlugins()` 查看插件状态
4. 监控日志输出: 查看详细的调用链信息