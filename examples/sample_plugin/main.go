// Package main 示例插件程序
// 演示如何创建一个功能完整的插件，包括文本处理、数学计算和插件间通信
package main

import (
	"context" // 上下文控制，用于函数调用的生命周期管理
	"fmt"     // 格式化输出，用于字符串处理和错误信息
	"log"     // 日志记录，用于输出运行信息和调试信息
	"os"      // 操作系统接口，用于命令行参数处理
	"strconv" // 字符串转换，用于数值类型转换
	"strings" // 字符串处理，用于文本操作函数

	wwplugin "github.com/wwwlkj/wwhyplugin" // WWPlugin插件框架核心库
	"github.com/wwwlkj/wwhyplugin/proto"    // gRPC协议定义，用于参数和返回值
)

// main 主函数 - 插件程序入口点
// 处理命令行参数，支持--info查询模式和正常运行模式
func main() {
	// 检查命令行参数，支持信息查询模式
	// --info 参数用于获取插件元数据，而不启动服务
	if len(os.Args) > 1 && os.Args[1] == "--info" {
		// 信息查询模式：不启动服务，只输出插件信息
		// 主机可以使用此功能在不加载插件的情况下获取插件信息
		plugin := createSamplePlugin() // 创建插件实例但不启动服务
		if err := plugin.StartWithInfo(); err != nil {
			os.Exit(1) // 信息查询失败则退出
		}
		os.Exit(0) // 正常退出信息查询模式
	}

	// 输出启动信息
	log.Println("启动示例插件...")

	// 创建插件实例
	// 这将配置插件的基本信息和能力
	plugin := createSamplePlugin()

	// 启动插件
	// 这将启动gRPC服务器、连接主机并注册服务
	if err := plugin.Start(); err != nil {
		log.Fatalf("启动插件失败: %v", err) // 启动失败则退出
	}
}

// createSamplePlugin 创建示例插件
func createSamplePlugin() *wwplugin.Plugin {
	// 插件配置
	config := wwplugin.DefaultPluginConfig(
		"SamplePlugin",
		"1.0.0",
		"这是一个示例插件，演示了插件系统的基本功能",
	)
	config.Capabilities = []string{
		"text_processing",
		"math_calculation",
		"inter_plugin_call",
	}

	// 创建插件
	plugin := wwplugin.NewPlugin(config)

	// 注册函数
	plugin.RegisterFunction("ReverseText", reverseText)
	plugin.RegisterFunction("UpperCase", upperCase)
	plugin.RegisterFunction("Add", add)
	plugin.RegisterFunction("TestHostCall", testHostCall(plugin))
	plugin.RegisterFunction("TestPluginCall", testPluginCall(plugin))

	// 设置消息处理器
	plugin.SetMessageHandler(messageHandler)

	return plugin
}

// 示例函数实现

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

// testHostCall 测试调用主机函数
func testHostCall(plugin *wwplugin.Plugin) wwplugin.PluginFunction {
	return func(ctx context.Context, params []*proto.Parameter) (*proto.Parameter, error) {
		// 调用主机的GetSystemTime函数
		resp, err := plugin.CallHostFunction("GetSystemTime", []*proto.Parameter{})
		if err != nil {
			return nil, fmt.Errorf("调用主机函数失败: %v", err)
		}

		if !resp.Success {
			return nil, fmt.Errorf("主机函数调用失败: %s", resp.Message)
		}

		result := fmt.Sprintf("主机时间: %s", resp.Result.Value)

		return &proto.Parameter{
			Name:  "host_call_result",
			Type:  proto.ParameterType_STRING,
			Value: result,
		}, nil
	}
}

// testPluginCall 测试插件间调用
func testPluginCall(plugin *wwplugin.Plugin) wwplugin.PluginFunction {
	return func(ctx context.Context, params []*proto.Parameter) (*proto.Parameter, error) {
		if len(params) < 2 {
			return nil, fmt.Errorf("需要参数: 目标插件ID和函数名")
		}

		targetPluginID := params[0].Value
		functionName := params[1].Value

		// 准备调用参数
		var callParams []*proto.Parameter
		if len(params) > 2 {
			callParams = params[2:]
		}

		// 调用其他插件的函数
		resp, err := plugin.CallOtherPlugin(targetPluginID, functionName, callParams)
		if err != nil {
			return nil, fmt.Errorf("调用插件函数失败: %v", err)
		}

		if !resp.Success {
			return nil, fmt.Errorf("插件函数调用失败: %s", resp.Message)
		}

		result := fmt.Sprintf("插件间调用成功\n目标插件: %s\n函数: %s\n结果: %s",
			targetPluginID, functionName, resp.Result.Value)

		return &proto.Parameter{
			Name:  "plugin_call_result",
			Type:  proto.ParameterType_STRING,
			Value: result,
		}, nil
	}
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
