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
	procReleaseMutex = kernel32.NewProc("ReleaseMutex")   // 释放互斥体API函数
	procCloseHandle  = kernel32.NewProc("CloseHandle")    // 关闭句柄API函数
	procGetLastError = kernel32.NewProc("GetLastError")   // 获取最后错误API函数
)

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
		// 首个实例：启动IPC服务器监听其他实例的连接
		listener, err := startIPCServer(config.IPCPort)
		if err != nil {
			// 如果启动服务器失败，释放互斥体
			releaseMutex(mutexHandle)
			return false, nil, fmt.Errorf("启动IPC服务器失败: %v", err)
		}
		return true, listener, nil
	} else {
		// 后续实例：发送命令参数到首个实例并退出
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

	// 调用Windows API创建互斥体
	ret, _, _ := procCreateMutex.Call(
		0,                                     // 安全属性，NULL表示使用默认安全描述符
		0,                                     // 初始所有者，FALSE表示调用线程不获得互斥体所有权
		uintptr(unsafe.Pointer(mutexNamePtr)), // 互斥体名称指针
	)

	// 检查API调用结果
	if ret == 0 {
		return 0, false, fmt.Errorf("CreateMutex API调用失败")
	}

	// 获取最后的错误码
	lastError, _, _ := procGetLastError.Call()

	// 转换句柄类型
	handle := syscall.Handle(ret)

	// 判断是否为首个实例
	isFirst := lastError != ERROR_ALREADY_EXISTS

	return handle, isFirst, nil
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
// 返回值：监听器对象，错误信息
func startIPCServer(port int) (net.Listener, error) {
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
	err = writePortToFile(actualPort)
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
	port, err := readPortFromFile()
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
func writePortToFile(port int) error {
	// 获取临时目录
	tempDir := os.TempDir()

	// 构建端口文件路径
	portFile := fmt.Sprintf("%s\\wwplugin_port_%d.tmp", tempDir, os.Getpid())

	// 写入端口号到文件
	return os.WriteFile(portFile, []byte(strconv.Itoa(port)), 0644)
}

// readPortFromFile 从临时文件读取端口号
// 返回值：端口号，错误信息
func readPortFromFile() (int, error) {
	// 获取临时目录
	tempDir := os.TempDir()

	// 查找端口文件（可能有多个进程）
	files, err := os.ReadDir(tempDir)
	if err != nil {
		return 0, fmt.Errorf("读取临时目录失败: %v", err)
	}

	// 遍历查找端口文件
	for _, file := range files {
		if len(file.Name()) > 14 && file.Name()[:14] == "wwplugin_port_" &&
			len(file.Name()) > 4 && file.Name()[len(file.Name())-4:] == ".tmp" {

			// 读取端口文件内容
			portFile := fmt.Sprintf("%s\\%s", tempDir, file.Name())
			data, err := os.ReadFile(portFile)
			if err != nil {
				continue // 跳过无法读取的文件
			}

			// 解析端口号
			port, err := strconv.Atoi(string(data))
			if err != nil {
				continue // 跳过无法解析的文件
			}

			return port, nil
		}
	}

	return 0, fmt.Errorf("未找到端口文件")
}

// CleanupSingleton 清理单实例相关资源
// 在程序退出时调用，清理临时文件等资源
func CleanupSingleton() {
	// 获取临时目录
	tempDir := os.TempDir()

	// 构建当前进程的端口文件路径
	portFile := fmt.Sprintf("%s\\wwplugin_port_%d.tmp", tempDir, os.Getpid())

	// 删除端口文件（忽略错误）
	os.Remove(portFile)
}
