// Package main 插件端单实例功能使用示例
// 演示如何在插件程序中集成Windows互斥体单实例功能
package main

import (
	"context"   // 上下文控制，用于函数调用的生命周期管理
	"fmt"       // 格式化输出，用于字符串处理和错误信息
	"log"       // 日志记录，用于输出运行信息和调试信息
	"os"        // 操作系统接口，用于命令行参数处理和信号
	"os/signal" // 系统信号处理，用于优雅退出
	"strconv"   // 字符串转换，用于数值类型转换
	"strings"   // 字符串处理，用于文本操作函数
	"syscall"   // 系统调用，用于信号处理
	"time"      // 时间处理，用于延时操作

	wwplugin "github.com/wwwlkj/wwhyplugin" // WWPlugin插件框架核心库
	"github.com/wwwlkj/wwhyplugin/proto"    // gRPC协议定义，用于参数和返回值
)

// 全局变量 - 插件实例和配置
var (
	globalPlugin     *wwplugin.Plugin           // 全局插件实例
	globalConfig     map[string]string          // 全局配置映射
	singletonManager *wwplugin.SingletonManager // 单实例管理器
)

// main 主函数 - 插件程序入口点，集成单实例功能
func main() {
	log.Println("🚀 启动示例插件（支持单实例）...")

	// 步骤1: 检查命令行参数中的特殊命令
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--info":
			// 信息查询模式：不启动服务，只输出插件信息
			handleInfoCommand()
			return
		case "--reload-config":
			// 重载配置命令：发送给已运行的实例
			log.Println("🔄 发送配置重载命令到运行中的插件实例...")
			// 这个命令会被单实例机制转发给运行中的实例
		case "--get-status":
			// 状态查询命令：发送给已运行的实例
			log.Println("📊 查询运行中插件实例的状态...")
			// 这个命令会被单实例机制转发给运行中的实例
		}
	}

	// 步骤2: 初始化单实例管理
	// 使用插件特定的互斥体名称，避免与主程序冲突
	pluginName := "SamplePlugin" // 可以从配置文件读取
	mutexName := fmt.Sprintf("WWPlugin_%s", pluginName)

	var err error
	singletonManager, err = wwplugin.NewSingletonManager(mutexName)
	if err != nil {
		log.Fatalf("❌ 创建单实例管理器失败: %v", err)
	}
	defer singletonManager.Close() // 确保资源清理

	// 步骤3: 检查是否为首个实例
	if !singletonManager.IsFirstInstance() {
		// 如果不是首个实例，程序会在NewSingletonManager中自动退出
		// 这行代码实际上不会执行到
		log.Println("🔄 命令已发送到运行中的插件实例")
		return
	}

	// 步骤4: 首个实例继续执行插件逻辑
	log.Printf("✅ 作为首个插件实例启动，监听地址: %s", singletonManager.GetListenerAddress())

	// 步骤5: 启动命令处理协程
	// 处理来自其他实例的命令
	go handlePluginCommands(singletonManager.GetCommandChannel())

	// 步骤6: 初始化插件配置
	globalConfig = initializePluginConfig()

	// 步骤7: 创建并启动插件
	globalPlugin = createSamplePlugin()
	if err := globalPlugin.Start(); err != nil {
		log.Fatalf("❌ 启动插件失败: %v", err)
	}

	log.Println("✅ 插件启动成功，等待主机连接...")

	// 步骤8: 等待退出信号
	waitForExitSignal()
}

// handleInfoCommand 处理信息查询命令
func handleInfoCommand() {
	// 创建插件实例但不启动服务，只输出信息
	plugin := createSamplePlugin()
	if err := plugin.StartWithInfo(); err != nil {
		log.Printf("❌ 输出插件信息失败: %v", err)
		os.Exit(1)
	}
	os.Exit(0)
}

// handlePluginCommands 处理来自其他插件实例的命令
// cmdChan: 命令消息通道
func handlePluginCommands(cmdChan <-chan *wwplugin.CommandMessage) {
	log.Println("🎯 开始监听来自其他插件实例的命令...")

	for message := range cmdChan {
		log.Printf("📨 收到来自进程 %d 的命令:", message.Pid)
		log.Printf("   📂 工作目录: %s", message.WorkDir)
		log.Printf("   📋 参数列表: %v", message.Args)
		log.Printf("   ⏰ 时间戳: %d", message.Timestamp)

		// 处理具体命令
		if len(message.Args) > 1 {
			switch message.Args[1] {
			case "--reload-config":
				handleReloadConfigCommand(message.Args)
			case "--get-status":
				handleGetStatusCommand(message.Args)
			case "--update-setting":
				handleUpdateSettingCommand(message.Args)
			case "--restart-connection":
				handleRestartConnectionCommand(message.Args)
			default:
				log.Printf("❓ 未知插件命令: %s", message.Args[1])
			}
		} else {
			// 无参数命令，显示插件状态
			log.Println("💡 显示插件运行状态")
			showPluginStatus()
		}
	}
}

// handleReloadConfigCommand 处理重载配置命令
// args: 命令行参数
func handleReloadConfigCommand(args []string) {
	log.Println("🔄 执行配置重载...")

	// 重新加载配置文件
	newConfig := loadConfigFromFile()
	if newConfig == nil {
		log.Println("❌ 配置重载失败：无法加载配置文件")
		return
	}

	// 更新全局配置
	globalConfig = newConfig
	log.Println("✅ 配置重载成功")

	// 如果需要，可以重启插件服务
	if globalConfig["auto_restart"] == "true" {
		log.Println("🔄 配置要求自动重启，正在重启插件服务...")
		restartPluginService()
	}
}

// handleGetStatusCommand 处理状态查询命令
// args: 命令行参数
func handleGetStatusCommand(args []string) {
	log.Println("📊 查询插件状态...")

	status := getPluginStatus()

	// 输出状态信息
	log.Printf("插件状态报告:")
	log.Printf("  - 插件名称: %s", status["name"])
	log.Printf("  - 运行状态: %s", status["status"])
	log.Printf("  - 启动时间: %s", status["start_time"])
	log.Printf("  - 连接状态: %s", status["connection"])
	log.Printf("  - 处理请求数: %s", status["request_count"])
	log.Printf("  - 内存使用: %s", status["memory_usage"])
}

// handleUpdateSettingCommand 处理设置更新命令
// args: 命令行参数（格式：--update-setting key=value）
func handleUpdateSettingCommand(args []string) {
	if len(args) < 3 {
		log.Println("❌ 设置更新命令格式错误，应为：--update-setting key=value")
		return
	}

	// 解析key=value格式
	setting := args[2]
	parts := strings.Split(setting, "=")
	if len(parts) != 2 {
		log.Printf("❌ 设置格式错误：%s，应为key=value格式", setting)
		return
	}

	key := parts[0]
	value := parts[1]

	log.Printf("🔧 更新设置：%s = %s", key, value)

	// 更新配置
	globalConfig[key] = value

	// 应用设置变更
	applySettingChange(key, value)

	log.Println("✅ 设置更新成功")
}

// handleRestartConnectionCommand 处理重启连接命令
// args: 命令行参数
func handleRestartConnectionCommand(args []string) {
	log.Println("🔄 重启与主机的连接...")

	if globalPlugin != nil {
		// 断开当前连接
		log.Println("📡 断开当前连接...")
		// 这里可以调用插件的断开连接方法

		// 等待一小段时间
		time.Sleep(2 * time.Second)

		// 重新连接
		log.Println("🔗 重新连接到主机...")
		// 这里可以调用插件的重连方法

		log.Println("✅ 连接重启完成")
	} else {
		log.Println("❌ 插件实例不存在，无法重启连接")
	}
}

// createSamplePlugin 创建示例插件（与原版本相同，但增加了单实例相关配置）
func createSamplePlugin() *wwplugin.Plugin {
	// 插件配置
	config := wwplugin.DefaultPluginConfig(
		"SamplePlugin",
		"1.0.0",
		"支持单实例管理的示例插件",
	)

	// 从全局配置读取能力列表
	config.Capabilities = []string{
		"text_processing",
		"math_calculation",
		"inter_plugin_call",
		"singleton_support", // 新增：单实例支持能力
	}

	// 从全局配置读取主机地址
	if hostAddr, exists := globalConfig["host_address"]; exists {
		config.HostAddress = hostAddr
	}

	// 创建插件
	plugin := wwplugin.NewPlugin(config)

	// 注册函数
	plugin.RegisterFunction("ReverseText", reverseText)
	plugin.RegisterFunction("UpperCase", upperCase)
	plugin.RegisterFunction("Add", add)
	plugin.RegisterFunction("GetPluginConfig", getPluginConfigFunction)       // 新增：获取配置函数
	plugin.RegisterFunction("UpdatePluginConfig", updatePluginConfigFunction) // 新增：更新配置函数

	// 设置消息处理器
	plugin.SetMessageHandler(messageHandler)

	return plugin
}

// initializePluginConfig 初始化插件配置
// 返回值：配置映射表
func initializePluginConfig() map[string]string {
	config := make(map[string]string)

	// 默认配置
	config["host_address"] = "localhost:50051"
	config["auto_restart"] = "false"
	config["debug_mode"] = "true"
	config["log_level"] = "info"
	config["heartbeat_interval"] = "10"

	// 尝试从配置文件加载
	if fileConfig := loadConfigFromFile(); fileConfig != nil {
		// 合并配置，文件配置优先
		for key, value := range fileConfig {
			config[key] = value
		}
	}

	log.Printf("📋 插件配置已初始化：%d 项配置", len(config))
	return config
}

// loadConfigFromFile 从文件加载配置
// 返回值：配置映射表，如果加载失败则返回nil
func loadConfigFromFile() map[string]string {
	// 这里可以实现从JSON、YAML或INI文件读取配置的逻辑
	// 为了简化示例，这里返回一个模拟的配置
	log.Println("📂 尝试从配置文件加载配置...")

	// 模拟配置文件内容
	config := map[string]string{
		"host_address": "localhost:50051",
		"debug_mode":   "true",
		"log_level":    "debug",
	}

	log.Println("✅ 配置文件加载成功")
	return config
}

// getPluginStatus 获取插件状态信息
// 返回值：状态信息映射表
func getPluginStatus() map[string]string {
	status := make(map[string]string)

	status["name"] = "SamplePlugin"
	status["status"] = "running"
	status["start_time"] = time.Now().Format("2006-01-02 15:04:05")
	status["connection"] = "connected"
	status["request_count"] = "42"    // 模拟数据
	status["memory_usage"] = "15.2MB" // 模拟数据

	return status
}

// applySettingChange 应用设置变更
// key: 设置键
// value: 设置值
func applySettingChange(key, value string) {
	switch key {
	case "debug_mode":
		// 应用调试模式变更
		log.Printf("🔧 应用调试模式变更: %s", value)
	case "log_level":
		// 应用日志级别变更
		log.Printf("🔧 应用日志级别变更: %s", value)
	case "heartbeat_interval":
		// 应用心跳间隔变更
		if interval, err := strconv.Atoi(value); err == nil {
			log.Printf("🔧 应用心跳间隔变更: %d秒", interval)
		}
	default:
		log.Printf("⚠️ 未知设置项: %s", key)
	}
}

// restartPluginService 重启插件服务
func restartPluginService() {
	if globalPlugin != nil {
		log.Println("🛑 停止当前插件服务...")
		// 这里可以调用插件的停止方法

		log.Println("🚀 启动新的插件服务...")
		// 重新创建并启动插件
		globalPlugin = createSamplePlugin()
		if err := globalPlugin.Start(); err != nil {
			log.Printf("❌ 重启插件服务失败: %v", err)
		} else {
			log.Println("✅ 插件服务重启成功")
		}
	}
}

// showPluginStatus 显示插件状态
func showPluginStatus() {
	status := getPluginStatus()
	log.Println("📊 当前插件状态:")
	for key, value := range status {
		log.Printf("   %s: %s", key, value)
	}
}

// getPluginConfigFunction 获取插件配置的函数（可被主机调用）
func getPluginConfigFunction(ctx context.Context, params []*proto.Parameter) (*proto.Parameter, error) {
	// 将配置转换为JSON格式返回
	configJSON := "{"
	i := 0
	for key, value := range globalConfig {
		if i > 0 {
			configJSON += ","
		}
		configJSON += fmt.Sprintf(`"%s":"%s"`, key, value)
		i++
	}
	configJSON += "}"

	return &proto.Parameter{
		Name:  "plugin_config",
		Type:  proto.ParameterType_JSON,
		Value: configJSON,
	}, nil
}

// updatePluginConfigFunction 更新插件配置的函数（可被主机调用）
func updatePluginConfigFunction(ctx context.Context, params []*proto.Parameter) (*proto.Parameter, error) {
	if len(params) < 2 {
		return nil, fmt.Errorf("需要key和value参数")
	}

	key := params[0].Value
	value := params[1].Value

	// 更新配置
	globalConfig[key] = value

	// 应用变更
	applySettingChange(key, value)

	return &proto.Parameter{
		Name:  "update_result",
		Type:  proto.ParameterType_STRING,
		Value: fmt.Sprintf("配置 %s 已更新为 %s", key, value),
	}, nil
}

// 原有的插件函数实现...
// reverseText 反转文本
func reverseText(ctx context.Context, params []*proto.Parameter) (*proto.Parameter, error) {
	if len(params) == 0 {
		return nil, fmt.Errorf("需要文本参数")
	}

	text := params[0].Value
	runes := []rune(text)

	// 反转字符串
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}

	return &proto.Parameter{
		Name:  "reversed_text",
		Type:  proto.ParameterType_STRING,
		Value: string(runes),
	}, nil
}

// upperCase 转换为大写
func upperCase(ctx context.Context, params []*proto.Parameter) (*proto.Parameter, error) {
	if len(params) == 0 {
		return nil, fmt.Errorf("需要文本参数")
	}

	text := params[0].Value

	return &proto.Parameter{
		Name:  "upper_text",
		Type:  proto.ParameterType_STRING,
		Value: strings.ToUpper(text),
	}, nil
}

// add 加法计算
func add(ctx context.Context, params []*proto.Parameter) (*proto.Parameter, error) {
	if len(params) < 2 {
		return nil, fmt.Errorf("需要至少2个数字参数")
	}

	var sum float64 = 0
	for _, param := range params {
		val, err := strconv.ParseFloat(param.Value, 64)
		if err != nil {
			return nil, fmt.Errorf("参数 %s 不是有效数字: %v", param.Value, err)
		}
		sum += val
	}

	return &proto.Parameter{
		Name:  "sum",
		Type:  proto.ParameterType_FLOAT,
		Value: fmt.Sprintf("%.2f", sum),
	}, nil
}

// messageHandler 消息处理器
func messageHandler(msg *proto.MessageRequest) {
	switch msg.MessageType {
	case "notification":
		log.Printf("📢 收到通知: %s", msg.Content)
	case "config_update":
		log.Printf("🔧 收到配置更新: %s", msg.Content)
		// 可以在这里处理配置更新逻辑
	case "restart_request":
		log.Printf("🔄 收到重启请求: %s", msg.Content)
		// 可以在这里处理重启请求
	default:
		log.Printf("📨 收到未知类型消息 %s: %s", msg.MessageType, msg.Content)
	}
}

// waitForExitSignal 等待退出信号
func waitForExitSignal() {
	// 创建信号通道
	sigChan := make(chan os.Signal, 1)

	// 监听中断信号
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 等待信号
	sig := <-sigChan
	log.Printf("🛑 收到退出信号: %v", sig)
	log.Println("👋 插件正在优雅退出...")

	// 清理资源
	if globalPlugin != nil {
		// 这里可以调用插件的停止方法
		log.Println("🧹 清理插件资源...")
	}
}
