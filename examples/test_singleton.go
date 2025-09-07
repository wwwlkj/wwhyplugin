// Package main 单实例功能测试程序
// 专门用于测试Windows互斥体单实例功能，不依赖插件主机
package main

import (
	"fmt"       // 格式化输出，用于信息显示
	"log"       // 日志记录，用于输出运行信息
	"os"        // 操作系统接口，用于信号处理
	"os/signal" // 系统信号处理，用于优雅退出
	"syscall"   // 系统调用，用于信号处理
	"time"      // 时间处理，用于延时

	wwplugin "github.com/wwwlkj/wwhyplugin" // WWPlugin插件框架核心库
)

func main() {
	fmt.Println("🧪 单实例功能测试程序启动...")
	fmt.Printf("⏰ 启动时间: %s\n", time.Now().Format("15:04:05"))
	fmt.Printf("🆔 进程ID: %d\n", os.Getpid())

	// 创建单实例管理器
	appName := "TestSingleton"
	manager, err := wwplugin.NewSingletonManager(appName)
	if err != nil {
		log.Fatalf("❌ 创建单实例管理器失败: %v", err)
	}
	defer manager.Close()

	// 检查是否为首个实例
	if !manager.IsFirstInstance() {
		// 这行代码不会执行，因为非首个实例会自动退出
		fmt.Println("🔄 命令已发送到首个实例")
		return
	}

	fmt.Println("✅ 成功获取单实例锁，作为首个实例运行")
	fmt.Printf("📡 IPC监听地址: %s\n", manager.GetListenerAddress())
	fmt.Println("📝 现在可以尝试启动第二个实例来测试互斥体功能")
	fmt.Println("💡 使用以下命令测试:")
	fmt.Println("   1. 打开新的命令行窗口")
	fmt.Println("   2. 运行: test_singleton.exe")
	fmt.Println("   3. 观察第二个实例是否被阻止")

	// 处理来自其他实例的命令
	go handleCommands(manager.GetCommandChannel())

	// 模拟主程序运行
	fmt.Println("🔄 程序运行中，按 Ctrl+C 退出...")

	// 定期输出状态，证明程序在运行
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				fmt.Printf("💓 程序运行中... %s (PID: %d)\n",
					time.Now().Format("15:04:05"), os.Getpid())
			}
		}
	}()

	// 等待退出信号
	waitForExit()
}

// handleCommands 处理来自其他实例的命令
func handleCommands(cmdChan <-chan *wwplugin.CommandMessage) {
	for message := range cmdChan {
		fmt.Printf("\n📨 收到来自进程 %d 的命令:\n", message.Pid)
		fmt.Printf("   📂 工作目录: %s\n", message.WorkDir)
		fmt.Printf("   📋 参数列表: %v\n", message.Args)
		fmt.Printf("   ⏰ 时间戳: %s\n", time.Unix(message.Timestamp, 0).Format("15:04:05"))

		// 根据命令参数执行不同操作
		if len(message.Args) > 1 {
			switch message.Args[1] {
			case "--status":
				fmt.Println("📊 显示当前状态:")
				fmt.Printf("   - 程序状态: 运行中\n")
				fmt.Printf("   - 运行时间: %s\n", time.Now().Format("15:04:05"))
				fmt.Printf("   - 进程ID: %d\n", os.Getpid())
			case "--hello":
				fmt.Println("👋 收到问候命令: Hello from another instance!")
			default:
				fmt.Printf("❓ 未知命令: %s\n", message.Args[1])
			}
		} else {
			fmt.Println("💡 收到无参数命令，显示程序正在运行")
		}
		fmt.Println()
	}
}

// waitForExit 等待退出信号
func waitForExit() {
	// 创建信号通道
	sigChan := make(chan os.Signal, 1)

	// 监听中断信号
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 等待信号
	sig := <-sigChan
	fmt.Printf("\n🛑 收到退出信号: %v\n", sig)
	fmt.Println("👋 程序正在优雅退出...")
	fmt.Println("🧹 清理单实例资源...")
}
