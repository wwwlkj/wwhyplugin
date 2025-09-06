// Package wwplugin 提供了一个基于 gRPC 的高性能、跨平台插件框架
//
// WWPlugin 支持多进程架构和双向通信，允许主程序和插件之间相互调用函数，
// 以及插件之间通过主程序中介进行调用。
//
// 基本用法:
//
// 创建插件主机:
//
//	config := wwplugin.DefaultHostConfig()
//	host, err := wwplugin.NewPluginHost(config)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// 启动主机
//	if err := host.Start(); err != nil {
//		log.Fatal(err)
//	}
//
//	// 加载并启动插件
//	plugin, err := host.StartPluginByPath("./myplugin.exe")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// 调用插件函数
//	result, err := host.CallPluginFunction(plugin.ID, "MyFunction", params)
//
// 创建插件:
//
//	config := wwplugin.DefaultPluginConfig("MyPlugin", "1.0.0", "示例插件")
//	plugin := wwplugin.NewPlugin(config)
//
//	// 注册函数
//	plugin.RegisterFunction("MyFunction", myFunction)
//
//	// 启动插件
//	if err := plugin.Start(); err != nil {
//		log.Fatal(err)
//	}
//
// 插件间调用:
//
//	// 在插件中调用其他插件的函数
//	resp, err := plugin.CallOtherPlugin("targetPluginID", "FunctionName", params)
package wwplugin

// Version 库版本号
// 遵循语义化版本规范 (Semantic Versioning)
const Version = "1.0.0"

// 导出主要类型和函数，方便使用者导入
// 注意：这些是类型别名，指向当前包中定义的实际类型
// 所有公共类型、常量和函数已在相应的源文件中定义，这里不需要重复导出
// 用户可以直接使用 wwplugin.PluginHost、wwplugin.NewPlugin() 等
