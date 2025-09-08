// Package main 长时间运行的调试程序
// 用于测试单实例功能的完整流程
package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	wwplugin "github.com/wwwlkj/wwhyplugin"
)

func main() {
	fmt.Printf("🧪 长时间运行调试程序启动 (PID: %d)\n", os.Getpid())

	// 使用与WWPlugin相同的互斥体名称格式
	appName := "LongRunningDebug"

	// 测试WWPlugin的CheckSingleInstance
	testLongRunningCheckSingleInstance(appName)
}

// testLongRunningCheckSingleInstance 长时间运行测试CheckSingleInstance
func testLongRunningCheckSingleInstance(appName string) {
	fmt.Println("\n=== 长时间运行测试CheckSingleInstance ===")

	// 创建配置
	config := wwplugin.DefaultSingletonConfig(appName)
	fmt.Printf("🔧 配置信息:\n")
	fmt.Printf("   互斥体名称: %s\n", config.MutexName)
	fmt.Printf("   IPC端口: %d\n", config.IPCPort)
	fmt.Printf("   超时时间: %d秒\n", config.Timeout)

	// 调用CheckSingleInstance
	fmt.Println("🔍 调用CheckSingleInstance...")
	isFirst, listener, err := wwplugin.CheckSingleInstance(config)

	if err != nil {
		fmt.Printf("❌ CheckSingleInstance失败: %v\n", err)
		return
	}

	if isFirst {
		fmt.Println("✅ 成功获取单实例锁，这是首个实例")
		if listener != nil {
			fmt.Printf("📡 IPC监听地址: %s\n", listener.Addr().String())
			fmt.Println("💡 现在启动第二个实例来测试")

			// 启动一个goroutine来处理连接
			go func() {
				for {
					conn, err := listener.Accept()
					if err != nil {
						fmt.Printf("⚠️  接受连接失败: %v\n", err)
						return
					}
					fmt.Printf("📨 收到连接请求\n")

					// 处理连接
					go func(conn net.Conn) {
						message, err := wwplugin.HandleIPCConnection(conn)
						if err != nil {
							fmt.Printf("⚠️  处理IPC连接失败: %v\n", err)
							return
						}
						fmt.Printf("📨 收到来自进程 %d 的命令: %v\n", message.Pid, message.Args)
						fmt.Printf("   工作目录: %s\n", message.WorkDir)
						fmt.Printf("   时间戳: %d\n", message.Timestamp)
					}(conn)
				}
			}()

			// 等待退出信号
			fmt.Println("⏰ 程序正在运行，按 Ctrl+C 退出...")
			waitForExitSignal(listener)
		}
	} else {
		// 这个分支永远不会执行到，因为CheckSingleInstance会让程序退出
		fmt.Println("⚠️  这是后续实例，程序应该已经退出")
	}
}

// waitForExitSignal 等待退出信号
func waitForExitSignal(listener net.Listener) {
	// 创建信号通道
	sigChan := make(chan os.Signal, 1)

	// 监听中断信号
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 等待信号
	select {
	case sig := <-sigChan:
		fmt.Printf("\n🛑 收到退出信号: %v\n", sig)
	case <-time.After(120 * time.Second):
		fmt.Printf("\n⏰ 超时自动退出\n")
	}

	fmt.Println("👋 程序正在优雅退出...")

	// 清理资源
	listener.Close()
	wwplugin.CleanupSingleton()
	fmt.Println("🧹 资源已清理")
}
