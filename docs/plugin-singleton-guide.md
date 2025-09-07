# 插件端单实例功能使用指南

## 📖 功能介绍

插件端的单实例功能主要用于防止同一个插件被多次启动，并支持通过命令行向运行中的插件发送控制命令。

## 🎯 使用场景

### 1. 防止插件多开
- 确保同一个插件只有一个实例在运行
- 避免资源冲突和重复注册问题

### 2. 插件热配置更新
- 通过命令行向运行中的插件发送配置更新命令
- 无需重启插件即可应用新配置

### 3. 插件状态查询
- 查询插件的运行状态和统计信息
- 远程调试和监控插件

### 4. 插件连接管理
- 重启与主机的连接
- 处理网络异常恢复

## 🚀 快速集成

### 1. 基础集成代码

```go
package main

import (
    "log"
    wwplugin "github.com/wwwlkj/wwhyplugin"
)

func main() {
    // 1. 创建插件专用的单实例管理器
    pluginName := "MyPlugin" 
    mutexName := fmt.Sprintf("WWPlugin_%s", pluginName)
    
    manager, err := wwplugin.NewSingletonManager(mutexName)
    if err != nil {
        log.Fatal(err)
    }
    defer manager.Close()
    
    // 2. 检查是否为首个实例（非首个会自动退出）
    if !manager.IsFirstInstance() {
        return // 不会执行到这里
    }
    
    // 3. 处理来自其他实例的命令
    go handlePluginCommands(manager.GetCommandChannel())
    
    // 4. 启动插件逻辑
    plugin := createPlugin()
    plugin.Start()
    
    // 5. 等待退出
    waitForExit()
}
```

### 2. 处理插件命令

```go
func handlePluginCommands(cmdChan <-chan *wwplugin.CommandMessage) {
    for message := range cmdChan {
        args := message.Args
        
        if len(args) > 1 {
            switch args[1] {
            case "--reload-config":
                reloadPluginConfig()
            case "--get-status":
                showPluginStatus()
            case "--restart-connection":
                restartConnection()
            case "--update-setting":
                if len(args) > 2 {
                    updateSetting(args[2]) // key=value格式
                }
            }
        }
    }
}
```

## 🛠️ 支持的命令

### 配置管理命令

```bash
# 重载插件配置
MyPlugin.exe --reload-config

# 更新单个设置
MyPlugin.exe --update-setting debug_mode=true
MyPlugin.exe --update-setting log_level=debug
```

### 状态查询命令

```bash
# 查询插件状态
MyPlugin.exe --get-status

# 获取插件信息（不启动服务）
MyPlugin.exe --info
```

### 连接管理命令

```bash
# 重启与主机的连接
MyPlugin.exe --restart-connection

# 显示运行状态（无参数）
MyPlugin.exe
```

## ⚙️ 配置建议

### 1. 互斥体命名规范

```go
// 推荐格式：WWPlugin_插件名称
mutexName := fmt.Sprintf("WWPlugin_%s", pluginName)

// 避免冲突的完整格式
mutexName := fmt.Sprintf("WWPlugin_%s_%s", pluginName, version)
```

### 2. 端口分配策略

```go
config := &wwplugin.SingletonConfig{
    MutexName:  mutexName,
    IPCPort:    0,        // 自动分配，避免端口冲突
    Timeout:    5,        // 5秒超时
    RetryCount: 3,        // 重试3次
}
```

## 📋 实现要点

### 1. 命令处理函数

```go
// 配置重载
func reloadPluginConfig() {
    log.Println("🔄 重载插件配置...")
    newConfig := loadConfigFromFile()
    applyNewConfig(newConfig)
    log.Println("✅ 配置重载完成")
}

// 状态查询
func showPluginStatus() {
    status := map[string]string{
        "name":          "MyPlugin",
        "status":        "running",
        "start_time":    startTime.Format("15:04:05"),
        "request_count": strconv.Itoa(requestCount),
    }
    
    for key, value := range status {
        log.Printf("%s: %s", key, value)
    }
}

// 设置更新
func updateSetting(setting string) {
    parts := strings.Split(setting, "=")
    if len(parts) == 2 {
        key, value := parts[0], parts[1]
        globalConfig[key] = value
        applySettingChange(key, value)
        log.Printf("✅ 设置已更新: %s = %s", key, value)
    }
}
```

### 2. 与主机通信的函数

```go
// 可被主机调用的配置获取函数
func getConfigFunction(ctx context.Context, params []*proto.Parameter) (*proto.Parameter, error) {
    configJSON := serializeConfig(globalConfig)
    return &proto.Parameter{
        Name:  "config",
        Type:  proto.ParameterType_JSON,
        Value: configJSON,
    }, nil
}

// 可被主机调用的配置更新函数
func updateConfigFunction(ctx context.Context, params []*proto.Parameter) (*proto.Parameter, error) {
    if len(params) >= 2 {
        key := params[0].Value
        value := params[1].Value
        
        globalConfig[key] = value
        applySettingChange(key, value)
        
        return &proto.Parameter{
            Name:  "result",
            Type:  proto.ParameterType_STRING,
            Value: fmt.Sprintf("配置 %s 已更新", key),
        }, nil
    }
    return nil, fmt.Errorf("参数不足")
}
```

## 🎯 最佳实践

### 1. 配置管理
- 使用配置文件存储插件设置
- 支持热重载，避免重启插件
- 提供默认配置作为fallback

### 2. 状态监控
- 记录插件运行统计信息
- 提供健康检查接口
- 支持远程状态查询

### 3. 错误处理
- 命令处理要有错误边界
- 网络异常时自动重连
- 记录详细的错误日志

### 4. 资源管理
- 正确清理单实例资源
- 优雅处理退出信号
- 避免资源泄漏

## 📝 完整示例

参考项目中的 `examples/plugin_with_singleton.go` 文件，包含了完整的插件端单实例功能实现。

## ❓ 常见问题

### Q: 插件和主机都用单实例会冲突吗？
A: 不会，使用不同的互斥体名称即可。建议插件使用 `WWPlugin_插件名` 格式。

### Q: 如何处理插件崩溃重启？
A: 新启动的插件实例会自动成为首个实例，因为原实例的互斥体已释放。

### Q: 多个不同插件可以同时运行吗？
A: 可以，每个插件使用不同的互斥体名称，互不干扰。

### Q: 如何在GUI插件中使用？
A: 同样的方式，在窗口创建前集成单实例检查，后续实例可以发送命令让首个实例显示窗口。