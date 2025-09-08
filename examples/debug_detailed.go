// Package main 详细调试程序
// 用于深入调试单实例功能的每个步骤
package main

import (
	"fmt"
	"net"
	"os"
	"time"

	wwplugin "github.com/wwwlkj/wwhyplugin"
)

func main() {
	fmt.Printf("🧪 详细调试程序启动 (PID: %d)\n", os.Getpid())

	// 使用与WWPlugin相同的互斥体名称格式
	appName := "DetailedDebug"

	// 测试WWPlugin的CheckSingleInstance
	testDetailedCheckSingleInstance(appName)
}

// testDetailedCheckSingleInstance 详细测试CheckSingleInstance
func testDetailedCheckSingleInstance(appName string) {
	fmt.Println("\n=== 详细测试CheckSingleInstance ===")

	// 创建配置
	config := wwplugin.DefaultSingletonConfig(appName)
	fmt.Printf("🔧 配置信息:\n")
	fmt.Printf("   互斥体名称: %s\n", config.MutexName)
	fmt.Printf("   IPC端口: %d\n", config.IPCPort)
	fmt.Printf("   超时时间: %d秒\n", config.Timeout)

	// 调用CheckSingleInstance
	fmt.Println("🔍 调用CheckSingleInstance...")
	isFirst, listener, err := wwplugin.CheckSingleInstance(config)

	fmt.Printf("📝 CheckSingleInstance返回值:\n")
	fmt.Printf("   isFirst: %t\n", isFirst)
	fmt.Printf("   listener: %v\n", listener)
	fmt.Printf("   err: %v\n", err)

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
					}(conn)
				}
			}()

			// 等待一段时间让第二个实例可以连接
			fmt.Println("⏰ 程序将运行60秒...")
			time.Sleep(60 * time.Second)

			// 清理资源
			listener.Close()
			wwplugin.CleanupSingleton()
			fmt.Println("🔚 首个实例退出")
		}
	} else {
		// 这个分支永远不会执行到，因为CheckSingleInstance会让程序退出
		fmt.Println("⚠️  这是后续实例，程序应该已经退出")
	}
}
