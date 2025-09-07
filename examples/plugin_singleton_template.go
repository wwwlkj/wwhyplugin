// Package main 插件端单实例功能简化模板
// 这是一个可以直接使用的插件单实例功能模板
package main

import (
	"context"   // 上下文控制，用于函数调用管理
	"fmt"       // 格式化输出，用于字符串处理
	"log"       // 日志记录，用于运行信息输出
	"os"        // 操作系统接口，用于信号处理
	"os/signal" // 系统信号处理，用于优雅退出
	"syscall"   // 系统调用，用于信号处理

	wwplugin "github.com/wwwlkj/wwhyplugin" // WWPlugin插件框架核心库
	"github.com/wwwlkj/wwhyplugin/proto"    // gRPC协议定义
)

// 配置你的插件信息
const (
	PLUGIN_NAME        = "MyPlugin" // 插件名称（修改为你的插件名）
	PLUGIN_VERSION     = "1.0.0"    // 插件版本
	PLUGIN_DESCRIPTION = "我的插件描述"   // 插件描述
)

// 全局变量
var (
	plugin *wwplugin.Plugin  // 插件实例
	config map[string]string // 插件配置
)

func main() {
	log.Printf("🚀 启动插件: %s v%s", PLUGIN_NAME, PLUGIN_VERSION)

	// === 第1步：处理特殊命令 ===
	if len(os.Args) > 1 && os.Args[1] == "--info" {
		showPluginInfo()
		return
	}

	// === 第2步：单实例检查 ===
	mutexName := fmt.Sprintf("WWPlugin_%s", PLUGIN_NAME)
	manager, err := wwplugin.NewSingletonManager(mutexName)
	if err != nil {
		log.Fatalf("❌ 单实例管理器创建失败: %v", err)
	}
	defer manager.Close()

	// 如果不是首个实例，程序会自动退出
	if !manager.IsFirstInstance() {
		return
	}

	log.Printf("✅ 作为首个实例启动")

	// === 第3步：启动命令处理 ===
	go handleCommands(manager.GetCommandChannel())

	// === 第4步：初始化插件 ===
	initConfig()
	plugin = createPlugin()

	if err := plugin.Start(); err != nil {
		log.Fatalf("❌ 插件启动失败: %v", err)
	}

	log.Println("✅ 插件启动成功")

	// === 第5步：等待退出 ===
	waitForExit()
}

// handleCommands 处理来自其他实例的命令
func handleCommands(cmdChan <-chan *wwplugin.CommandMessage) {
	for message := range cmdChan {
		if len(message.Args) > 1 {
			switch message.Args[1] {
			case "--reload":
				log.Println("🔄 重载配置...")
				reloadConfig()
			case "--status":
				log.Println("📊 显示状态...")
				showStatus()
			default:
				log.Printf("❓ 未知命令: %s", message.Args[1])
			}
		} else {
			log.Println("💡 显示插件信息")
			showStatus()
		}
	}
}

// initConfig 初始化配置
func initConfig() {
	config = make(map[string]string)
	config["debug_mode"] = "true"
	config["log_level"] = "info"
	// 在这里添加你的默认配置
	log.Println("📋 配置已初始化")
}

// reloadConfig 重载配置
func reloadConfig() {
	// 在这里实现你的配置重载逻辑
	log.Println("✅ 配置重载完成")
}

// showStatus 显示状态
func showStatus() {
	log.Printf("插件状态:")
	log.Printf("  名称: %s", PLUGIN_NAME)
	log.Printf("  版本: %s", PLUGIN_VERSION)
	log.Printf("  状态: 运行中")
	// 在这里添加更多状态信息
}

// showPluginInfo 显示插件信息（--info命令）
func showPluginInfo() {
	plugin := createPlugin()
	if err := plugin.StartWithInfo(); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}

// createPlugin 创建插件实例
func createPlugin() *wwplugin.Plugin {
	config := wwplugin.DefaultPluginConfig(
		PLUGIN_NAME,
		PLUGIN_VERSION,
		PLUGIN_DESCRIPTION,
	)

	plugin := wwplugin.NewPlugin(config)

	// === 在这里注册你的插件函数 ===
	plugin.RegisterFunction("Hello", helloFunction)
	plugin.RegisterFunction("GetConfig", getConfigFunction)
	// 添加更多函数...

	return plugin
}

// === 插件函数实现 ===

// helloFunction 示例插件函数
func helloFunction(ctx context.Context, params []*proto.Parameter) (*proto.Parameter, error) {
	name := "World"
	if len(params) > 0 {
		name = params[0].Value
	}

	return &proto.Parameter{
		Name:  "greeting",
		Type:  proto.ParameterType_STRING,
		Value: fmt.Sprintf("Hello, %s! 来自 %s", name, PLUGIN_NAME),
	}, nil
}

// getConfigFunction 获取配置函数
func getConfigFunction(ctx context.Context, params []*proto.Parameter) (*proto.Parameter, error) {
	configStr := fmt.Sprintf("配置项数量: %d", len(config))

	return &proto.Parameter{
		Name:  "config_info",
		Type:  proto.ParameterType_STRING,
		Value: configStr,
	}, nil
}

// waitForExit 等待退出信号
func waitForExit() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	log.Printf("🛑 收到退出信号: %v", sig)
	log.Println("👋 插件正在退出...")
}
