// Package main 单实例功能使用示例
// 演示如何在主程序中集成Windows互斥体单实例功能
package main

import (
	"fmt"       // 格式化输出，用于日志和错误信息
	"log"       // 日志记录，用于运行时信息输出
	"net"       // 网络操作，用于处理IPC连接
	"os"        // 操作系统接口，用于信号处理
	"os/signal" // 系统信号处理，用于优雅退出
	"syscall"   // 系统调用，用于信号处理

	wwplugin "github.com/wwwlkj/wwhyplugin" // WWPlugin插件框架核心库
)

// main 主函数 - 演示单实例功能集成
func main() {
	// 步骤1: 配置单实例管理器
	// 使用应用程序名称创建默认配置
	config := wwplugin.DefaultSingletonConfig("WWPluginHost")
	log.Printf("🔧 单实例配置: 互斥体名称=%s, 端口=%d", config.MutexName, config.IPCPort)

	// 步骤2: 检查单实例状态
	// 这个调用会检查是否已有实例运行
	isFirst, listener, err := wwplugin.CheckSingleInstance(config)
	if err != nil {
		log.Fatalf("❌ 单实例检查失败: %v", err)
	}

	// 步骤3: 根据检查结果执行相应逻辑
	if !isFirst {
		// 如果不是首个实例，程序会在CheckSingleInstance中自动退出
		// 这行代码实际上不会执行到
		log.Println("🔄 命令已发送到首个实例，程序退出")
		return
	}

	// 首个实例继续执行主程序逻辑
	log.Println("🚀 作为首个实例启动...")

	// 步骤4: 设置资源清理
	// 确保程序退出时清理资源
	defer wwplugin.CleanupSingleton()

	// 步骤5: 启动IPC消息处理
	// 在后台处理来自其他实例的命令
	if listener != nil {
		go handleIPCMessages(listener)
		log.Printf("📡 IPC服务已启动，监听地址: %s", listener.Addr().String())
	}

	// 步骤6: 启动主要的插件主机功能
	// 这里是你的原始主程序逻辑
	if err := startPluginHost(); err != nil {
		log.Fatalf("❌ 启动插件主机失败: %v", err)
	}

	// 步骤7: 等待退出信号
	// 设置优雅退出处理
	waitForExitSignal()
}

// handleIPCMessages 处理来自其他实例的IPC消息
// listener: IPC监听器
func handleIPCMessages(listener net.Listener) {
	log.Println("🎯 开始监听其他实例的命令...")

	for {
		// 接受新的连接
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("⚠️ 接受IPC连接失败: %v", err)
			continue
		}

		// 在新的goroutine中处理连接
		go func(conn net.Conn) {
			// 处理IPC连接，解析命令消息
			message, err := wwplugin.HandleIPCConnection(conn)
			if err != nil {
				log.Printf("⚠️ 处理IPC消息失败: %v", err)
				return
			}

			// 输出接收到的命令信息
			log.Printf("📨 收到来自进程 %d 的命令:", message.Pid)
			log.Printf("   📂 工作目录: %s", message.WorkDir)
			log.Printf("   📋 参数列表: %v", message.Args)
			log.Printf("   ⏰ 时间戳: %d", message.Timestamp)

			// 处理命令逻辑
			handleReceivedCommand(message)
		}(conn)
	}
}

// handleReceivedCommand 处理接收到的命令
// message: 从其他实例接收的命令消息
func handleReceivedCommand(message *wwplugin.CommandMessage) {
	// 根据命令行参数执行相应操作
	args := message.Args

	if len(args) > 1 {
		switch args[1] {
		case "--load-plugin":
			// 处理加载插件命令
			if len(args) > 2 {
				pluginPath := args[2]
				log.Printf("🔌 收到加载插件命令: %s", pluginPath)
				// 这里可以调用你的插件加载逻辑
				loadPluginFromCommand(pluginPath)
			}
		case "--list-plugins":
			// 处理列出插件命令
			log.Println("📝 收到列出插件命令")
			// 这里可以调用你的插件列表逻辑
			listPluginsFromCommand()
		case "--status":
			// 处理状态查询命令
			log.Println("📊 收到状态查询命令")
			// 这里可以调用你的状态查询逻辑
			showStatusFromCommand()
		default:
			log.Printf("❓ 未知命令: %s", args[1])
		}
	} else {
		// 无参数时显示已运行状态
		log.Println("💡 程序已在运行中，显示主窗口或执行默认操作")
		// 这里可以实现显示主界面的逻辑
		showMainWindow()
	}
}

// startPluginHost 启动插件主机
// 这里是你原有的主程序逻辑
func startPluginHost() error {
	log.Println("🏗️ 启动插件主机...")

	// 创建插件主机配置
	config := wwplugin.DefaultHostConfig()
	config.DebugMode = true

	// 创建插件主机实例
	host, err := wwplugin.NewPluginHost(config)
	if err != nil {
		return fmt.Errorf("创建插件主机失败: %v", err)
	}

	// 启动主机服务
	if err := host.Start(); err != nil {
		return fmt.Errorf("启动插件主机失败: %v", err)
	}

	log.Printf("✅ 插件主机已启动，监听端口: %d", host.GetActualPort())
	log.Println("📝 现在可以加载和管理插件了")

	// 这里可以添加更多的初始化逻辑
	// 比如自动加载配置文件中的插件等

	return nil
}

// loadPluginFromCommand 从命令加载插件
// pluginPath: 插件文件路径
func loadPluginFromCommand(pluginPath string) {
	log.Printf("🔌 正在加载插件: %s", pluginPath)
	// 这里实现你的插件加载逻辑
	// 可以调用host.StartPluginByPath(pluginPath)
}

// listPluginsFromCommand 从命令列出插件
func listPluginsFromCommand() {
	log.Println("📋 当前已加载的插件列表:")
	// 这里实现你的插件列表逻辑
	// 可以调用host.GetAllPlugins()
}

// showStatusFromCommand 从命令显示状态
func showStatusFromCommand() {
	log.Println("📊 系统运行状态:")
	// 这里实现你的状态显示逻辑
	log.Println("   - 插件主机: 运行中")
	log.Println("   - 已加载插件: 0")
}

// showMainWindow 显示主窗口
func showMainWindow() {
	log.Println("🖥️ 显示主界面（在GUI应用中可以将窗口置前）")
	// 在GUI应用程序中，这里可以实现将窗口置前的逻辑
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
	log.Println("👋 程序正在优雅退出...")
}
