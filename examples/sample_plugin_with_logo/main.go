// Package main 带Logo的示例插件程序
// 演示如何在插件信息中添加Logo
package main

import (
	"context"
	"log"
	"os"

	wwplugin "github.com/wwwlkj/wwhyplugin"
	"github.com/wwwlkj/wwhyplugin/proto"
)

// main 主函数 - 插件程序入口点
func main() {
	// 检查命令行参数，支持信息查询模式
	if len(os.Args) > 1 && os.Args[1] == "--info" {
		// 信息查询模式：不启动服务，只输出插件信息
		plugin := createSamplePluginWithLogo()
		if err := plugin.StartWithInfo(); err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}

	// 输出启动信息
	log.Println("启动带Logo的示例插件...")

	// 创建插件实例
	plugin := createSamplePluginWithLogo()

	// 启动插件
	if err := plugin.Start(); err != nil {
		log.Fatalf("启动插件失败: %v", err)
	}
}

// createSamplePluginWithLogo 创建带Logo的示例插件
func createSamplePluginWithLogo() *wwplugin.Plugin {
	// 插件配置，包含Logo信息
	config := wwplugin.DefaultPluginConfig(
		"SamplePluginWithLogo",
		"1.0.0",
		"这是一个带Logo的示例插件，演示了如何在插件信息中添加Logo",
	)

	// 设置Logo（可以是Base64编码的图片数据或图片路径）
	config.Logo = getPluginLogo()

	// 设置插件能力
	config.Capabilities = []string{
		"text_processing",
		"logo_demo",
	}

	// 创建插件
	plugin := wwplugin.NewPlugin(config)

	// 注册函数
	plugin.RegisterFunction("GetPluginLogoInfo", getPluginLogoInfo)

	// 设置消息处理器
	plugin.SetMessageHandler(messageHandler)

	return plugin
}

// getPluginLogoInfo 获取插件Logo信息
func getPluginLogoInfo(ctx context.Context, params []*proto.Parameter) (*proto.Parameter, error) {
	return &proto.Parameter{
		Name:  "logo_info",
		Type:  proto.ParameterType_STRING,
		Value: "这个插件包含Logo信息，可以在--info模式下查看",
	}, nil
}

// messageHandler 消息处理器
func messageHandler(msg *proto.MessageRequest) {
	switch msg.MessageType {
	case "notification":
		log.Printf("📢 收到通知: %s", msg.Content)
	case "command":
		log.Printf("⚡ 收到命令: %s", msg.Content)
	case "data":
		log.Printf("📊 收到数据: %s", msg.Content)
	default:
		log.Printf("❓ 收到未知类型消息: %s - %s", msg.MessageType, msg.Content)
	}
}

// getPluginLogo 获取插件Logo数据
// 这里返回一个简单的Base64编码示例
// 在实际应用中，你可以：
// 1. 嵌入真实的Base64图片数据
// 2. 返回图片文件路径
// 3. 从网络URL获取图片
func getPluginLogo() string {
	// 这是一个1x1像素的透明PNG图片的Base64编码示例
	// 在实际使用中，你应该替换为真实的Logo图片数据
	return "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg=="
}
