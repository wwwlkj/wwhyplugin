//go:build ignore
// +build ignore

package main

import (
	"log"
	"time"

	"github.com/wwplugin/wwplugin"
)

// 这个文件用于测试库的编译是否正常
func main() {
	log.Println("测试 WWPlugin 库编译...")

	// 测试主机创建
	config := wwplugin.DefaultHostConfig()
	host, err := wwplugin.NewPluginHost(config)
	if err != nil {
		log.Fatalf("创建主机失败: %v", err)
	}

	// 测试插件创建
	pluginConfig := wwplugin.DefaultPluginConfig("TestPlugin", "1.0.0", "测试插件")
	plugin := wwplugin.NewPlugin(pluginConfig)

	log.Printf("主机创建成功，插件创建成功")
	log.Printf("主机端口范围: %v", config.PortRange)
	log.Printf("插件名称: %s", plugin.GetPluginInfo().Name)

	// 简单的功能测试
	if err := host.Start(); err != nil {
		log.Fatalf("启动主机失败: %v", err)
	}

	log.Printf("主机启动成功，监听端口: %d", host.GetActualPort())

	// 短暂运行后停止
	time.Sleep(2 * time.Second)
	host.Stop()

	log.Println("✅ WWPlugin 库编译和基本功能测试通过！")
}
