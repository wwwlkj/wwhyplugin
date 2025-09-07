// Package wwplugin  单实例功能辅助函数
// 提供简化的单实例管理接口，方便快速集成
package wwplugin

import (
	"log" // 日志记录，用于输出运行信息
	"net" // 网络接口，用于IPC通信
)

// SingletonManager 单实例管理器结构体
// 封装单实例管理的复杂逻辑，提供简化的接口
type SingletonManager struct {
	config   *SingletonConfig     // 单实例配置参数
	listener net.Listener         // IPC监听器
	isFirst  bool                 // 是否为首个实例
	cmdChan  chan *CommandMessage // 命令消息通道
}

// NewSingletonManager 创建单实例管理器
// appName: 应用程序名称，用于生成互斥体名称
// 返回值：管理器实例，错误信息
func NewSingletonManager(appName string) (*SingletonManager, error) {
	// 创建默认配置
	config := DefaultSingletonConfig(appName)

	// 检查单实例状态
	isFirst, listener, err := CheckSingleInstance(config)
	if err != nil {
		return nil, err
	}

	// 创建命令通道
	cmdChan := make(chan *CommandMessage, 10)

	// 创建管理器实例
	manager := &SingletonManager{
		config:   config,
		listener: listener,
		isFirst:  isFirst,
		cmdChan:  cmdChan,
	}

	// 如果是首个实例且有监听器，启动命令处理
	if isFirst && listener != nil {
		go manager.handleIPCMessages()
	}

	return manager, nil
}

// IsFirstInstance 检查是否为首个实例
// 返回值：true表示首个实例，false表示后续实例（但后续实例会自动退出）
func (sm *SingletonManager) IsFirstInstance() bool {
	return sm.isFirst
}

// GetCommandChannel 获取命令消息通道
// 返回值：只读的命令消息通道
func (sm *SingletonManager) GetCommandChannel() <-chan *CommandMessage {
	return sm.cmdChan
}

// GetListenerAddress 获取IPC监听地址
// 返回值：监听地址字符串，如果没有监听器则返回空字符串
func (sm *SingletonManager) GetListenerAddress() string {
	if sm.listener != nil {
		return sm.listener.Addr().String()
	}
	return ""
}

// Close 关闭单实例管理器
// 清理所有资源，包括监听器和通道
func (sm *SingletonManager) Close() error {
	// 清理资源
	CleanupSingleton()

	// 关闭监听器
	if sm.listener != nil {
		return sm.listener.Close()
	}

	// 关闭命令通道
	close(sm.cmdChan)

	return nil
}

// handleIPCMessages 处理IPC消息（内部方法）
// 在后台goroutine中运行，接收并处理来自其他实例的命令
func (sm *SingletonManager) handleIPCMessages() {
	log.Printf("🎯 单实例管理器开始监听IPC消息，地址: %s", sm.GetListenerAddress())

	for {
		// 接受连接
		conn, err := sm.listener.Accept()
		if err != nil {
			log.Printf("⚠️ 接受IPC连接失败: %v", err)
			break // 监听器关闭时退出循环
		}

		// 处理连接
		go func(conn net.Conn) {
			// 解析命令消息
			message, err := HandleIPCConnection(conn)
			if err != nil {
				log.Printf("⚠️ 处理IPC消息失败: %v", err)
				return
			}

			log.Printf("📨 收到来自进程 %d 的命令: %v", message.Pid, message.Args)

			// 发送到命令通道
			select {
			case sm.cmdChan <- message:
				// 成功发送到通道
			default:
				// 通道满了，丢弃消息
				log.Printf("⚠️ 命令通道已满，丢弃消息")
			}
		}(conn)
	}
}

// EnsureSingleInstance 确保单实例运行（简化版本）
// appName: 应用程序名称
// 返回值：命令消息通道（仅首个实例有效），错误信息
// 注意：如果不是首个实例，此函数不会返回（程序会退出）
func EnsureSingleInstance(appName string) (<-chan *CommandMessage, error) {
	// 创建管理器
	manager, err := NewSingletonManager(appName)
	if err != nil {
		return nil, err
	}

	// 如果不是首个实例，这里不会执行到
	// 因为CheckSingleInstance会让程序退出

	// 设置程序退出时的清理
	// 注意：这里使用了包级别的清理函数
	// 在实际使用中，建议在main函数中使用defer manager.Close()

	return manager.GetCommandChannel(), nil
}
