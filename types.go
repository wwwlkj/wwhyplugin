// Package wwplugin 提供插件框架的核心类型定义
// 包含插件状态、配置、注册表等基础数据结构
package wwplugin

import (
	"context" // 用于上下文控制
	"os"      // 操作系统接口
	"os/exec" // 进程执行
	"sync"    // 同步原语
	"time"    // 时间处理

	"github.com/wwwlkj/wwhyplugin/proto" // gRPC协议定义
	"google.golang.org/grpc"             // gRPC框架
)

// PluginStatus 插件状态枚举类型
// 定义插件在生命周期中的各种状态
type PluginStatus string

// 插件状态常量定义
const (
	StatusStopped  PluginStatus = "stopped"  // 插件已停止 - 初始状态或正常停止
	StatusStarting PluginStatus = "starting" // 插件正在启动中 - 过渡状态
	StatusRunning  PluginStatus = "running"  // 插件正常运行中 - 可接收调用
	StatusStopping PluginStatus = "stopping" // 插件正在停止中 - 过渡状态
	StatusError    PluginStatus = "error"    // 插件出现错误 - 需要干预
	StatusCrashed  PluginStatus = "crashed"  // 插件崩溃 - 可能需要重启
)

// PluginInfo 插件信息结构体
// 包含插件的全部运行时信息和配置参数
type PluginInfo struct {
	// === 基本信息 === //
	ID             string   `json:"id"`              // 插件唯一标识符 - 用于区分不同插件实例
	Name           string   `json:"name"`            // 插件名称 - 用户友好的显示名称
	Version        string   `json:"version"`         // 插件版本号 - 遵循语义化版本规范
	Description    string   `json:"description"`     // 插件功能描述 - 详细说明插件作用
	Port           int32    `json:"port"`            // 插件gRPC服务监听端口 - 用于主机连接
	Capabilities   []string `json:"capabilities"`    // 插件能力列表 - 描述插件提供的功能
	Functions      []string `json:"functions"`       // 插件提供的函数列表 - 可调用的函数名
	ExecutablePath string   `json:"executable_path"` // 插件可执行文件路径 - 用于启动进程

	// === 运行时信息 === //
	Process       *os.Process               `json:"-"`              // 插件进程对象 - 用于进程控制
	Command       *exec.Cmd                 `json:"-"`              // 执行命令对象 - 保存启动参数
	Client        proto.PluginServiceClient `json:"-"`              // gRPC客户端 - 用于调用插件服务
	Connection    *grpc.ClientConn          `json:"-"`              // gRPC连接对象 - 管理网络连接
	Status        PluginStatus              `json:"status"`         // 当前插件运行状态 - 实时状态信息
	StartTime     time.Time                 `json:"start_time"`     // 插件启动时间 - 用于计算运行时长
	LastHeartbeat time.Time                 `json:"last_heartbeat"` // 最后一次心跳时间 - 用于健康检查

	// === 配置参数 === //
	AutoRestart  bool `json:"auto_restart"`  // 是否在插件崩溃时自动重启 - 容错配置
	MaxRestarts  int  `json:"max_restarts"`  // 最大重启次数 - 防止无限重启
	RestartCount int  `json:"restart_count"` // 当前已重启次数计数器 - 跟踪重启情况
}

// PluginBasicInfo 插件基础信息结构（用于信息查询）
// 不包含运行时信息，仅包含静态元数据，用于--info查询
type PluginBasicInfo struct {
	ID           string   `json:"id"`             // 插件ID - 唯一标识符
	Name         string   `json:"name"`           // 插件名称 - 用户友好名称
	Version      string   `json:"version"`        // 插件版本 - 语义化版本号
	Description  string   `json:"description"`    // 插件描述 - 功能说明
	Logo         string   `json:"logo,omitempty"` // 插件Logo - Base64编码的图片数据或图片路径
	Capabilities []string `json:"capabilities"`   // 插件能力 - 功能特性列表
	Functions    []string `json:"functions"`      // 插件函数列表 - 可调用的函数名
}

// HostConfig 主程序配置结构体
// 包含主机运行所需的所有配置参数
type HostConfig struct {
	// === 网络配置 === //
	Port      int   `json:"port"`       // gRPC服务端口（0表示自动分配）
	PortRange []int `json:"port_range"` // 端口范围 [start, end] - 自动分配时的范围

	// === 日志配置 === //
	DebugMode bool   `json:"debug_mode"` // 是否开启调试模式 - 输出详细日志
	LogLevel  string `json:"log_level"`  // 日志级别 - debug/info/warn/error
	LogDir    string `json:"log_dir"`    // 日志目录 - 日志文件存储位置

	// === 健康监控 === //
	HeartbeatInterval     time.Duration `json:"heartbeat_interval"`      // 心跳间隔 - 检查插件健康的时间间隔
	MaxHeartbeatMiss      int           `json:"max_heartbeat_miss"`      // 最大心跳丢失次数 - 超过后认为插件崩溃
	AutoRestartPlugin     bool          `json:"auto_restart_plugin"`     // 是否自动重启崩溃的插件
	EnablePluginReconnect bool          `json:"enable_plugin_reconnect"` // 是否允许插件断线重连
}

// PluginConfig 插件配置结构体
// 包含插件运行所需的所有配置参数
type PluginConfig struct {
	// === 基本信息 === //
	Name         string   `json:"name"`           // 插件名称 - 显示名称
	Version      string   `json:"version"`        // 插件版本 - 语义化版本号
	Description  string   `json:"description"`    // 插件描述 - 功能说明
	Logo         string   `json:"logo,omitempty"` // 插件Logo - Base64编码的图片数据或图片路径
	Capabilities []string `json:"capabilities"`   // 插件能力列表 - 描述插件功能特性

	// === 网络配置 === //
	HostAddress string `json:"host_address"` // 主程序地址 - 插件连接的主机地址

	// === 健康监控 === //
	HeartbeatInterval     time.Duration `json:"heartbeat_interval"`       // 心跳间隔 - 发送心跳的时间间隔
	ReconnectInterval     time.Duration `json:"reconnect_interval"`       // 重连间隔 - 连接断开后的重连等待时间
	MaxReconnectTries     int           `json:"max_reconnect_tries"`      // 最大重连次数（0表示无限重连）
	CloseOnHostDisconnect bool          `json:"close_on_host_disconnect"` // 主机断开连接后是否关闭插件
}

// PluginFunction 插件函数类型定义
type PluginFunction func(ctx context.Context, params []*proto.Parameter) (*proto.Parameter, error)

// HostFunction 主程序函数类型定义
type HostFunction func(ctx context.Context, params []*proto.Parameter) (*proto.Parameter, error)

// MessageHandler 消息处理器类型定义
type MessageHandler func(msg *proto.MessageRequest)

// LogLevel 日志级别
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

// String returns the string representation of LogLevel
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "debug"
	case INFO:
		return "info"
	case WARN:
		return "warn"
	case ERROR:
		return "error"
	default:
		return "unknown"
	}
}

// LogConfig 日志配置
type LogConfig struct {
	DebugMode   bool     `json:"debug_mode"`   // 是否开启Debug模式
	LogLevel    LogLevel `json:"log_level"`    // 日志级别
	LogDir      string   `json:"log_dir"`      // 日志文件目录
	ServiceName string   `json:"service_name"` // 服务名称
}

// DefaultHostConfig 返回默认的主程序配置
func DefaultHostConfig() *HostConfig {
	return &HostConfig{
		Port:                  0, // 自动分配端口
		PortRange:             []int{50051, 50100},
		DebugMode:             true,
		LogLevel:              "info",
		LogDir:                "./logs",
		HeartbeatInterval:     10 * time.Second,
		MaxHeartbeatMiss:      3,
		AutoRestartPlugin:     true,
		EnablePluginReconnect: true, // 默认允许插件断线重连
	}
}

// DefaultPluginConfig 返回默认的插件配置
func DefaultPluginConfig(name, version, description string) *PluginConfig {
	return &PluginConfig{
		Name:                  name,
		Version:               version,
		Description:           description,
		Logo:                  "", // 默认为空Logo
		Capabilities:          []string{},
		HostAddress:           "localhost:50051",
		HeartbeatInterval:     10 * time.Second,
		ReconnectInterval:     5 * time.Second,
		MaxReconnectTries:     0,    // 无限重连
		CloseOnHostDisconnect: true, // 默认主机断开连接后关闭插件
	}
}

// PluginRegistry 插件注册表
type PluginRegistry struct {
	plugins map[string]*PluginInfo
	mutex   sync.RWMutex
}

// NewPluginRegistry 创建新的插件注册表
func NewPluginRegistry() *PluginRegistry {
	return &PluginRegistry{
		plugins: make(map[string]*PluginInfo),
	}
}

// Register 注册插件
func (pr *PluginRegistry) Register(plugin *PluginInfo) {
	pr.mutex.Lock()
	defer pr.mutex.Unlock()
	pr.plugins[plugin.ID] = plugin
}

// Unregister 注销插件
func (pr *PluginRegistry) Unregister(pluginID string) {
	pr.mutex.Lock()
	defer pr.mutex.Unlock()
	delete(pr.plugins, pluginID)
}

// Get 获取插件信息
func (pr *PluginRegistry) Get(pluginID string) (*PluginInfo, bool) {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()
	plugin, exists := pr.plugins[pluginID]
	return plugin, exists
}

// List 获取所有插件列表
func (pr *PluginRegistry) List() []*PluginInfo {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	plugins := make([]*PluginInfo, 0, len(pr.plugins))
	for _, plugin := range pr.plugins {
		plugins = append(plugins, plugin)
	}
	return plugins
}

// Count 获取插件数量
func (pr *PluginRegistry) Count() int {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()
	return len(pr.plugins)
}
