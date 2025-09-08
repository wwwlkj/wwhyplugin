// Package main 单实例功能详细调试程序
// 用于深入调试Windows互斥体和单实例功能
package main

import (
	"fmt"
	"os"
	"time"

	wwplugin "github.com/wwwlkj/wwhyplugin"
)

func main() {
	fmt.Printf("🧪 单实例功能详细调试程序启动 (PID: %d)\n", os.Getpid())

	// 使用与WWPlugin相同的互斥体名称格式
	appName := "DebugSingleton"

	// 测试WWPlugin的CheckSingleInstance
	testWWPluginCheckSingleInstance(appName)
}

// testWWPluginCheckSingleInstance 测试WWPlugin的CheckSingleInstance
func testWWPluginCheckSingleInstance(appName string) {
	fmt.Println("\n=== 测试: 使用WWPlugin的CheckSingleInstance ===")

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

			// 等待一段时间让第二个实例可以连接
			fmt.Println("⏰ 程序将运行30秒...")
			time.Sleep(30 * time.Second)

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
