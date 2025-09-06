// Package main 插件主机示例程序
// 演示如何创建和管理插件主机，加载插件并调用插件功能
package main

import (
	"log"  // 日志记录，用于输出运行信息和错误
	"time" // 时间处理，用于延时和定时操作

	"github.com/wwplugin/wwplugin"       // WWPlugin插件框架核心库
	"github.com/wwplugin/wwplugin/proto" // gRPC协议定义，用于参数传递
)

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

	// 启动主机服务
	// 这将启动gRPC服务器并开始监听插件连接
	if err := host.Start(); err != nil {
		log.Fatal(err) // 启动失败则退出程序
	}

	// 输出启动成功信息，显示实际监听端口
	log.Printf("🚀 插件主机已启动，监听端口: %d", host.GetActualPort())
	log.Printf("📝 现在可以加载和管理插件了")

	// 示例：自动加载插件（如果存在）
	go func() {
		time.Sleep(2 * time.Second)

		// 尝试加载示例插件
		pluginPath := `D:\GoSrc\wwplugin\examples\sample_plugin\plugin.exe`
		plugin, err := host.StartPluginByPath(pluginPath)
		if err != nil {
			log.Printf("自动加载插件失败: %v", err)
			return
		}

		log.Printf("✅ 自动加载插件成功: %s", plugin.ID)

		// 等待插件注册
		time.Sleep(3 * time.Second)

		// 测试插件调用
		log.Printf("🔧 测试插件功能...")
		testPluginFunctions(host, plugin.ID)
	}()

	// 等待退出信号
	host.Wait()
}

// testPluginFunctions 测试插件功能
func testPluginFunctions(host *wwplugin.PluginHost, pluginID string) {
	// 测试文本反转
	resp, err := host.CallPluginFunction(pluginID, "ReverseText", []*proto.Parameter{
		{Name: "text", Type: proto.ParameterType_STRING, Value: "Hello World"},
	})
	if err != nil {
		log.Printf("❌ 调用ReverseText失败: %v", err)
	} else if resp.Success {
		log.Printf("✅ ReverseText: %s", resp.Result.Value)
	}

	// 测试加法
	resp, err = host.CallPluginFunction(pluginID, "Add", []*proto.Parameter{
		{Name: "num1", Type: proto.ParameterType_FLOAT, Value: "10.5"},
		{Name: "num2", Type: proto.ParameterType_FLOAT, Value: "20.3"},
	})
	if err != nil {
		log.Printf("❌ 调用Add失败: %v", err)
	} else if resp.Success {
		log.Printf("✅ Add: %s", resp.Result.Value)
	}

	// 测试发送消息
	resp2, err := host.SendMessageToPlugin(pluginID, "notification", "来自主机的问候消息", nil)
	if err != nil {
		log.Printf("❌ 发送消息失败: %v", err)
	} else {
		log.Printf("✅ 消息已发送，处理了 %d 条消息", resp2.ProcessedCount)
	}
}
