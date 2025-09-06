# WWPlugin 项目代码规范

## 📝 中文注释规范

### 1. 文件头注释
每个Go文件都应该包含详细的包级别注释：

```go
// Package wwplugin 提供插件主机实现
// 负责插件的生命周期管理、通信协调和监控
package wwplugin

import (
    "context"      // 上下文控制，用于取消和超时管理
    "encoding/json" // JSON编解码，用于配置和数据交换
    // ... 其他导入
)
```

### 2. 结构体注释规范

#### 主要结构体
```go
// PluginHost 插件主机结构体 - 管理插件生命周期和通信
// 作为插件系统的中心控制器，负责协调所有插件的运行
type PluginHost struct {
    // === 核心组件 === //
    config        *HostConfig       // 主机配置 - 包含端口、日志等参数
    registry      *PluginRegistry   // 插件注册表 - 管理所有已加载的插件
    
    // === 控制组件 === //
    ctx          context.Context    // 全局上下文 - 用于统一取消操作
    cancel       context.CancelFunc // 取消函数 - 用于停止所有子操作
}
```

#### 配置结构体
```go
// HostConfig 主程序配置结构体
// 包含主机运行所需的所有配置参数
type HostConfig struct {
    // === 网络配置 === //
    Port      int   `json:"port"`       // gRPC服务端口（0表示自动分配）
    PortRange []int `json:"port_range"` // 端口范围 [start, end] - 自动分配时的范围
    
    // === 健康监控 === //
    HeartbeatInterval time.Duration `json:"heartbeat_interval"`  // 心跳间隔 - 检查插件健康的时间间隔
    MaxHeartbeatMiss  int           `json:"max_heartbeat_miss"`  // 最大心跳丢失次数 - 超过后认为插件崩溃
}
```

### 3. 函数注释规范

#### 构造函数
```go
// NewPluginHost 创建新的插件主机实例
// 初始化主机所需的所有组件：注册表、服务、上下文等
// 参数:
//   - config: 主机配置，如为nil则使用默认配置
// 返回:
//   - *PluginHost: 初始化完成的主机实例
//   - error: 创建过程中的错误
func NewPluginHost(config *HostConfig) (*PluginHost, error) {
    // 如果没有提供配置，使用默认配置
    if config == nil {
        config = DefaultHostConfig()
    }
    // ... 实现
}
```

#### 核心方法
```go
// Start 启动插件主机
// 这将启动gRPC服务器并开始监听插件连接
// 返回:
//   - error: 启动过程中的错误
func (ph *PluginHost) Start() error {
    log.Printf("🚀 启动插件主机...")
    
    // 启动gRPC服务器
    if err := ph.startGrpcServer(); err != nil {
        return fmt.Errorf("启动gRPC服务器失败: %v", err)
    }
    // ... 其他启动步骤
}
```

### 4. 变量和常量注释

#### 状态常量
```go
// 插件状态常量定义
const (
    StatusStopped  PluginStatus = "stopped"  // 插件已停止 - 初始状态或正常停止
    StatusStarting PluginStatus = "starting" // 插件正在启动中 - 过渡状态
    StatusRunning  PluginStatus = "running"  // 插件正常运行中 - 可接收调用
    StatusCrashed  PluginStatus = "crashed"  // 插件崩溃 - 可能需要重启
)
```

#### 版本信息
```go
// Version 库版本号
// 遵循语义化版本规范 (Semantic Versioning)
const Version = "1.0.0"
```

### 5. 示例代码注释

#### 主机示例
```go
// main 主函数 - 程序入口点
// 演示完整的插件主机生命周期：创建、启动、加载插件、测试功能
func main() {
    // 创建插件主机配置
    // 使用默认配置，可根据需要修改参数
    config := wwplugin.DefaultHostConfig()
    config.DebugMode = true // 开启调试模式，输出详细日志
    
    // 创建插件主机实例
    // 主机负责管理所有插件的生命周期
    host, err := wwplugin.NewPluginHost(config)
    if err != nil {
        log.Fatal(err) // 创建失败则退出程序
    }
}
```

#### 插件示例
```go
// main 主函数 - 插件程序入口点
// 处理命令行参数，支持--info查询模式和正常运行模式
func main() {
    // 检查命令行参数，支持信息查询模式
    // --info 参数用于获取插件元数据，而不启动服务
    if len(os.Args) > 1 && os.Args[1] == "--info" {
        // 信息查询模式：不启动服务，只输出插件信息
        // 主机可以使用此功能在不加载插件的情况下获取插件信息
        plugin := createSamplePlugin() // 创建插件实例但不启动服务
        if err := plugin.StartWithInfo(); err != nil {
            os.Exit(1) // 信息查询失败则退出
        }
        os.Exit(0) // 正常退出信息查询模式
    }
}
```

### 6. 特殊注释标记

#### 分组注释
```go
type PluginInfo struct {
    // === 基本信息 === //
    ID             string   // 插件唯一标识符 - 用于区分不同插件实例
    Name           string   // 插件名称 - 用户友好的显示名称
    
    // === 运行时信息 === //
    Process       *os.Process  // 插件进程对象 - 用于进程控制
    Connection    *grpc.ClientConn // gRPC连接对象 - 管理网络连接
    
    // === 配置参数 === //
    AutoRestart  bool // 是否在插件崩溃时自动重启 - 容错配置
    MaxRestarts  int  // 最大重启次数 - 防止无限重启
}
```

#### 重要提示注释
```go
// 注意：这些是类型别名，指向当前包中定义的实际类型
// 所有公共类型、常量和函数已在相应的源文件中定义，这里不需要重复导出
// 用户可以直接使用 wwplugin.PluginHost、wwplugin.NewPlugin() 等
```

### 7. 代码内联注释

#### 关键逻辑注释
```go
func (ph *PluginHost) startGrpcServer() error {
    startPort := ph.config.PortRange[0]
    maxPort := ph.config.PortRange[1]

    // 如果指定了具体端口，则只尝试该端口
    if ph.config.Port > 0 {
        startPort = ph.config.Port
        maxPort = ph.config.Port
    }

    // 自动寻找可用端口
    for port := startPort; port <= maxPort; port++ {
        address := fmt.Sprintf(":%d", port)
        listener, err = net.Listen("tcp", address)
        if err == nil {
            actualPort = port
            log.Printf("🎯 找到可用端口: %d", actualPort)
            break // 找到可用端口，退出循环
        }
        log.Printf("端口 %d 被占用，尝试下一个...", port)
    }
}
```

## 🎯 注释原则

1. **完整性**: 每个公共函数、结构体、接口都必须有注释
2. **准确性**: 注释内容必须与代码实现保持一致
3. **清晰性**: 使用简洁明了的中文描述功能和用途
4. **层次性**: 使用分组注释和不同级别的说明
5. **实用性**: 包含参数说明、返回值说明和使用示例

## 📋 检查清单

- [ ] 所有导入包都有用途说明
- [ ] 所有公共类型都有详细注释
- [ ] 所有公共函数都有参数和返回值说明
- [ ] 所有配置字段都有用途和默认值说明
- [ ] 所有常量都有含义说明
- [ ] 复杂逻辑都有内联注释解释
- [ ] 示例代码包含完整的使用说明

这些注释规范确保了代码的可读性和可维护性，方便其他开发者理解和使用WWPlugin框架。