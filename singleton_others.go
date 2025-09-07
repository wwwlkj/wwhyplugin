//go:build !windows
// +build !windows

// Package wwplugin 单实例管理模块 - 非Windows平台
// 在非Windows平台提供空实现，保持API兼容性
package wwplugin

import (
	"fmt" // 格式化输出，用于错误信息
	"net" // 网络接口，保持接口一致性
)

// SingletonConfig 单实例配置结构体（非Windows平台占位符）
type SingletonConfig struct {
	MutexName  string // 互斥体名称（在非Windows平台无效）
	IPCPort    int    // 进程间通信端口（在非Windows平台无效）
	Timeout    int    // 通信超时时间（在非Windows平台无效）
	RetryCount int    // 重试次数（在非Windows平台无效）
}

// CommandMessage 进程间通信消息结构体（非Windows平台占位符）
type CommandMessage struct {
	Args      []string `json:"args"`      // 命令行参数列表
	Pid       int      `json:"pid"`       // 发送进程的进程ID
	Timestamp int64    `json:"timestamp"` // 消息发送时间戳
	WorkDir   string   `json:"work_dir"`  // 工作目录路径
}

// DefaultSingletonConfig 返回默认的单实例配置（非Windows平台占位符）
// appName: 应用程序名称
// 返回值：配置结构体指针
func DefaultSingletonConfig(appName string) *SingletonConfig {
	return &SingletonConfig{
		MutexName:  appName, // 简单使用应用程序名称
		IPCPort:    0,       // 端口设置为0
		Timeout:    5,       // 默认超时时间
		RetryCount: 3,       // 默认重试次数
	}
}

// CheckSingleInstance 检查单实例（非Windows平台占位实现）
// config: 单实例配置参数
// 返回值：始终返回true（表示首个实例），nil监听器，不支持错误
func CheckSingleInstance(config *SingletonConfig) (isFirst bool, listener net.Listener, err error) {
	// 非Windows平台不支持单实例功能
	return true, nil, fmt.Errorf("单实例功能仅在Windows平台支持")
}

// HandleIPCConnection 处理IPC连接（非Windows平台占位实现）
// conn: 网络连接对象
// 返回值：nil消息，不支持错误
func HandleIPCConnection(conn net.Conn) (*CommandMessage, error) {
	// 非Windows平台不支持IPC功能
	return nil, fmt.Errorf("IPC功能仅在Windows平台支持")
}

// CleanupSingleton 清理单实例资源（非Windows平台占位实现）
// 在非Windows平台无需执行任何操作
func CleanupSingleton() {
	// 非Windows平台无需清理操作
}
