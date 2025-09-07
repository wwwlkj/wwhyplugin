//go:build windows
// +build windows

// Package wwplugin 单实例管理模块 - Windows专用
// 提供互斥体机制防止程序多开，支持命令参数转发到已运行实例
package wwplugin

import (
	"encoding/json" // JSON编解码，用于命令参数序列化传输
	"fmt"           // 格式化输出，用于错误信息和调试日志
	"net"           // 网络通信，用于进程间TCP通信
	"os"            // 操作系统接口，用于获取命令行参数和进程信息
	"strconv"       // 字符串转换，用于数字格式化
	"strings"       // 字符串操作，用于文件名处理
	"syscall"       // 系统调用，用于Windows API操作
	"time"          // 时间处理，用于超时控制和时间戳
	"unsafe"        // 不安全指针操作，用于Windows API参数传递
)

// Windows API 常量定义
const (
	MUTEX_ALL_ACCESS     = 0x1F0001 // 互斥体完全访问权限常量
	ERROR_ALREADY_EXISTS = 183      // 对象已存在错误码常量
	IPC_TIMEOUT          = 5        // 进程间通信超时时间（秒）
)

// Windows API 函数声明
var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll") // 加载内核动态链接库
	procCreateMutex  = kernel32.NewProc("CreateMutexW")   // 创建互斥体API函数
	procOpenMutex    = kernel32.NewProc("OpenMutexW")     // 打开互斥体API函数
	procReleaseMutex = kernel32.NewProc("ReleaseMutex")   // 释放互斥体API函数
	procCloseHandle  = kernel32.NewProc("CloseHandle")    // 关闭句柄API函数
)

// windowsSingletonManager Windows下的单实例管理器内部结构体
// 用于保持互斥体句柄和相关资源
type windowsSingletonManager struct {
	mutexHandle syscall.Handle // 互斥体句柄，必须持续持有
	mutexName   string         // 互斥体名称
}

// 全局变量，用于保持Windows互斥体管理器
var globalMutexManager *windowsSingletonManager

// CommandMessage 进程间通信消息结构体
// 用于在不同进程实例间传递命令行参数
type CommandMessage struct {
	Args      []string `json:"args"`      // 命令行参数列表
	Pid       int      `json:"pid"`       // 发送进程的进程ID
	Timestamp int64    `json:"timestamp"` // 消息发送时间戳
	WorkDir   string   `json:"work_dir"`  // 工作目录路径
}

// SingletonConfig 单实例配置结构体
// 用于配置单实例管理器的行为参数
type SingletonConfig struct {
	MutexName  string // 互斥体名称，建议使用应用程序唯一标识
	IPCPort    int    // 进程间通信端口，0表示自动分配
	Timeout    int    // 通信超时时间（秒）
	RetryCount int    // 重试次数
}

// DefaultSingletonConfig 返回默认的单实例配置
// appName: 应用程序名称，用于生成互斥体名称
func DefaultSingletonConfig(appName string) *SingletonConfig {
	return &SingletonConfig{
		MutexName:  fmt.Sprintf("Global\\%s_Mutex", appName), // 全局互斥体名称
		IPCPort:    0,                                        // 自动分配端口
		Timeout:    IPC_TIMEOUT,                              // 默认超时时间
		RetryCount: 3,                                        // 默认重试次数
	}
}

// CheckSingleInstance 检查单实例并处理多开情况
// config: 单实例配置参数
// 返回值：isFirst表示是否为首个实例，listener用于接收其他实例的命令，error表示错误信息
func CheckSingleInstance(config *SingletonConfig) (isFirst bool, listener net.Listener, err error) {
	// 参数验证
	if config == nil {
		return false, nil, fmt.Errorf("配置参数不能为空")
	}
	if config.MutexName == "" {
		return false, nil, fmt.Errorf("互斥体名称不能为空")
	}

	// 尝试创建互斥体
	mutexHandle, isFirst, err := createMutex(config.MutexName)
	if err != nil {
		return false, nil, fmt.Errorf("创建互斥体失败: %v", err)
	}

	if isFirst {
		// 首个实例：保存互斥体句柄并启动IPC服务器
		globalMutexManager = &windowsSingletonManager{
			mutexHandle: mutexHandle,
			mutexName:   config.MutexName,
		}

		listener, err := startIPCServer(config.IPCPort, config.MutexName)
		if err != nil {
			// 如果启动服务器失败，释放互斥体
			releaseMutex(mutexHandle)
			globalMutexManager = nil
			return false, nil, fmt.Errorf("启动IPC服务器失败: %v", err)
		}
		return true, listener, nil
	} else {
		// 后续实例：发送命令参数到首个实例并退出
		// 先关闭当前实例的互斥体句柄
		procCloseHandle.Call(uintptr(mutexHandle))

		err := sendCommandToFirstInstance(config)
		if err != nil {
			return false, nil, fmt.Errorf("发送命令到首个实例失败: %v", err)
		}
		// 发送成功后退出程序
		os.Exit(0)
		return false, nil, nil // 永远不会执行到这里
	}
}

// createMutex 创建Windows互斥体
// mutexName: 互斥体名称
// 返回值：互斥体句柄，是否为首个实例，错误信息
func createMutex(mutexName string) (syscall.Handle, bool, error) {
	// 将Go字符串转换为Windows宽字符串指针
	mutexNamePtr, err := syscall.UTF16PtrFromString(mutexName)
	if err != nil {
		return 0, false, fmt.Errorf("转换互斥体名称失败: %v", err)
	}

	// 首先尝试打开现有的互斥体
	openHandle, _, _ := procOpenMutex.Call(
		MUTEX_ALL_ACCESS,                      // 访问权限
		0,                                     // 不继承句柄
		uintptr(unsafe.Pointer(mutexNamePtr)), // 互斥体名称
	)

	if openHandle != 0 {
		// 互斥体已存在，这是后续实例
		procCloseHandle.Call(uintptr(openHandle))
		return 0, false, nil // 返回无效句柄，表示不是首个实例
	}

	// 互斥体不存在，创建新的互斥体
	createHandle, _, _ := procCreateMutex.Call(
		0,                                     // 安全属性，NULL
		1,                                     // 初始拥有者，立即获得所有权
		uintptr(unsafe.Pointer(mutexNamePtr)), // 互斥体名称
	)

	// 检查创建是否成功
	if createHandle == 0 {
		return 0, false, fmt.Errorf("CreateMutex失败")
	}

	handle := syscall.Handle(createHandle)
	return handle, true, nil // 返回有效句柄，表示是首个实例
}

// releaseMutex 释放互斥体资源
// handle: 互斥体句柄
func releaseMutex(handle syscall.Handle) error {
	if handle == 0 {
		return nil // 句柄为空，直接返回
	}

	// 释放互斥体
	ret, _, _ := procReleaseMutex.Call(uintptr(handle))
	if ret == 0 {
		return fmt.Errorf("释放互斥体失败")
	}

	// 关闭句柄
	ret, _, _ = procCloseHandle.Call(uintptr(handle))
	if ret == 0 {
		return fmt.Errorf("关闭互斥体句柄失败")
	}

	return nil
}

// startIPCServer 启动进程间通信服务器
// port: 监听端口，0表示自动分配
// mutexName: 互斥体名称，用于生成端口文件名
// 返回值：监听器对象，错误信息
func startIPCServer(port int, mutexName string) (net.Listener, error) {
	// 构建监听地址
	address := "127.0.0.1:" + strconv.Itoa(port)
	if port == 0 {
		address = "127.0.0.1:0" // 自动分配可用端口
	}

	// 创建TCP监听器
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, fmt.Errorf("创建TCP监听器失败: %v", err)
	}

	// 将实际监听端口写入临时文件供其他实例读取
	actualPort := listener.Addr().(*net.TCPAddr).Port
	err = writePortToFile(actualPort, mutexName)
	if err != nil {
		listener.Close() // 关闭监听器
		return nil, fmt.Errorf("写入端口文件失败: %v", err)
	}

	return listener, nil
}

// sendCommandToFirstInstance 发送命令参数到首个实例
// config: 单实例配置参数
func sendCommandToFirstInstance(config *SingletonConfig) error {
	// 从临时文件读取首个实例的监听端口
	port, err := readPortFromFile(config.MutexName)
	if err != nil {
		return fmt.Errorf("读取端口文件失败: %v", err)
	}

	// 获取当前工作目录
	workDir, _ := os.Getwd()

	// 构建命令消息
	message := CommandMessage{
		Args:      os.Args,           // 当前进程的命令行参数
		Pid:       os.Getpid(),       // 当前进程ID
		Timestamp: time.Now().Unix(), // 当前时间戳
		WorkDir:   workDir,           // 当前工作目录
	}

	// 序列化消息为JSON
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("序列化命令消息失败: %v", err)
	}

	// 连接到首个实例
	address := fmt.Sprintf("127.0.0.1:%d", port)
	conn, err := net.DialTimeout("tcp", address, time.Duration(config.Timeout)*time.Second)
	if err != nil {
		return fmt.Errorf("连接到首个实例失败: %v", err)
	}
	defer conn.Close() // 确保连接关闭

	// 设置写入超时
	conn.SetWriteDeadline(time.Now().Add(time.Duration(config.Timeout) * time.Second))

	// 发送消息长度（4字节）
	length := len(data)
	lengthBytes := []byte{
		byte(length >> 24), // 高位字节
		byte(length >> 16),
		byte(length >> 8),
		byte(length), // 低位字节
	}

	_, err = conn.Write(lengthBytes)
	if err != nil {
		return fmt.Errorf("发送消息长度失败: %v", err)
	}

	// 发送消息内容
	_, err = conn.Write(data)
	if err != nil {
		return fmt.Errorf("发送消息内容失败: %v", err)
	}

	return nil
}

// HandleIPCConnection 处理来自其他实例的IPC连接
// conn: 网络连接对象
// 返回值：解析出的命令消息，错误信息
func HandleIPCConnection(conn net.Conn) (*CommandMessage, error) {
	defer conn.Close() // 确保连接关闭

	// 设置读取超时
	conn.SetReadDeadline(time.Now().Add(IPC_TIMEOUT * time.Second))

	// 读取消息长度（4字节）
	lengthBytes := make([]byte, 4)
	_, err := conn.Read(lengthBytes)
	if err != nil {
		return nil, fmt.Errorf("读取消息长度失败: %v", err)
	}

	// 解析消息长度
	length := int(lengthBytes[0])<<24 | int(lengthBytes[1])<<16 | int(lengthBytes[2])<<8 | int(lengthBytes[3])

	// 验证消息长度合理性
	if length <= 0 || length > 1024*1024 { // 限制最大1MB
		return nil, fmt.Errorf("消息长度异常: %d", length)
	}

	// 读取消息内容
	data := make([]byte, length)
	_, err = conn.Read(data)
	if err != nil {
		return nil, fmt.Errorf("读取消息内容失败: %v", err)
	}

	// 反序列化JSON消息
	var message CommandMessage
	err = json.Unmarshal(data, &message)
	if err != nil {
		return nil, fmt.Errorf("反序列化消息失败: %v", err)
	}

	return &message, nil
}

// writePortToFile 将端口号写入临时文件
// port: 要写入的端口号
// mutexName: 互斥体名称，用于生成文件名
func writePortToFile(port int, mutexName string) error {
	// 获取临时目录
	tempDir := os.TempDir()

	// 使用互斥体名称的哈希值生成唯一但固定的文件名
	// 替换路径分隔符和特殊字符，确保文件名有效
	safeName := strings.ReplaceAll(mutexName, "Global\\", "")
	safeName = strings.ReplaceAll(safeName, "\\", "_")
	safeName = strings.ReplaceAll(safeName, ":", "_")
	safeName = strings.ReplaceAll(safeName, "*", "_")
	safeName = strings.ReplaceAll(safeName, "?", "_")
	safeName = strings.ReplaceAll(safeName, "<", "_")
	safeName = strings.ReplaceAll(safeName, ">", "_")
	safeName = strings.ReplaceAll(safeName, "|", "_")

	// 构建端口文件路径，使用互斥体名称而不是进程ID
	portFile := fmt.Sprintf("%s\\wwplugin_port_%s.tmp", tempDir, safeName)

	// 写入端口号到文件
	return os.WriteFile(portFile, []byte(strconv.Itoa(port)), 0644)
}

// readPortFromFile 从临时文件读取端口号
// mutexName: 互斥体名称，用于定位对应的端口文件
// 返回值：端口号，错误信息
func readPortFromFile(mutexName string) (int, error) {
	// 获取临时目录
	tempDir := os.TempDir()

	// 使用与writePortToFile相同的逻辑生成文件名
	safeName := strings.ReplaceAll(mutexName, "Global\\", "")
	safeName = strings.ReplaceAll(safeName, "\\", "_")
	safeName = strings.ReplaceAll(safeName, ":", "_")
	safeName = strings.ReplaceAll(safeName, "*", "_")
	safeName = strings.ReplaceAll(safeName, "?", "_")
	safeName = strings.ReplaceAll(safeName, "<", "_")
	safeName = strings.ReplaceAll(safeName, ">", "_")
	safeName = strings.ReplaceAll(safeName, "|", "_")

	// 构建端口文件路径
	portFile := fmt.Sprintf("%s\\wwplugin_port_%s.tmp", tempDir, safeName)

	// 读取端口文件内容
	data, err := os.ReadFile(portFile)
	if err != nil {
		return 0, fmt.Errorf("读取端口文件失败: %v", err)
	}

	// 解析端口号
	port, err := strconv.Atoi(string(data))
	if err != nil {
		return 0, fmt.Errorf("解析端口号失败: %v", err)
	}

	return port, nil
}

// CleanupSingleton 清理单实例相关资源
// 在程序退出时调用，清理互斥体和临时文件等资源
func CleanupSingleton() {
	// 释放互斥体资源
	if globalMutexManager != nil {
		if globalMutexManager.mutexHandle != 0 {
			// 释放互斥体
			procReleaseMutex.Call(uintptr(globalMutexManager.mutexHandle))
			// 关闭句柄
			procCloseHandle.Call(uintptr(globalMutexManager.mutexHandle))
		}

		// 清理对应的端口文件
		if globalMutexManager.mutexName != "" {
			cleanupPortFile(globalMutexManager.mutexName)
		}

		globalMutexManager = nil
	}
}

// cleanupPortFile 清理端口文件
// mutexName: 互斥体名称
func cleanupPortFile(mutexName string) {
	// 获取临时目录
	tempDir := os.TempDir()

	// 使用与writePortToFile相同的逻辑生成文件名
	safeName := strings.ReplaceAll(mutexName, "Global\\", "")
	safeName = strings.ReplaceAll(safeName, "\\", "_")
	safeName = strings.ReplaceAll(safeName, ":", "_")
	safeName = strings.ReplaceAll(safeName, "*", "_")
	safeName = strings.ReplaceAll(safeName, "?", "_")
	safeName = strings.ReplaceAll(safeName, "<", "_")
	safeName = strings.ReplaceAll(safeName, ">", "_")
	safeName = strings.ReplaceAll(safeName, "|", "_")

	// 构建端口文件路径
	portFile := fmt.Sprintf("%s\\wwplugin_port_%s.tmp", tempDir, safeName)

	// 删除端口文件（忽略错误）
	os.Remove(portFile)
}
