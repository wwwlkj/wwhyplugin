// Package wwplugin 提供插件主机实现
// 负责插件的生命周期管理、通信协调和监控
package wwplugin

import (
	"context"       // 上下文控制，用于取消和超时管理
	"encoding/json" // JSON编解码，用于配置和数据交换
	"fmt"           // 格式化输出，用于错误信息和日志
	"log"           // 日志记录，用于运行时信息输出
	"net"           // 网络操作，gRPC服务器监听
	"os"            // 操作系统接口，环境变量和信号处理
	"os/exec"       // 进程执行，用于启动插件进程
	"os/signal"     // 系统信号处理，用于优雅关闭
	"sync"          // 同步原语，管理并发访问
	"syscall"       // 系统调用，用于信号处理
	"time"          // 时间处理，心跳和超时管理

	"github.com/wwwlkj/wwhyplugin/proto" // gRPC协议定义
	"google.golang.org/grpc"             // gRPC框架
)

// PluginHost 插件主机结构体 - 管理插件生命周期和通信
// 作为插件系统的中心控制器，负责协调所有插件的运行
type PluginHost struct {
	// === 核心组件 === //
	config        *HostConfig             // 主机配置 - 包含端口、日志等参数
	registry      *PluginRegistry         // 插件注册表 - 管理所有已加载的插件
	hostService   *hostService            // 主机服务实现 - 处理插件请求
	grpcServer    *grpc.Server            // gRPC服务器 - 提供插件调用接口
	listener      net.Listener            // 网络监听器 - 监听客户端连接
	actualPort    int                     // 实际使用端口 - 可能与配置不同（自动分配）
	hostFunctions map[string]HostFunction // 主机函数映射 - 插件可调用的函数

	// === 控制组件 === //
	ctx          context.Context    // 全局上下文 - 用于统一取消操作
	cancel       context.CancelFunc // 取消函数 - 用于停止所有子操作
	wg           sync.WaitGroup     // 等待组 - 等待所有goroutine结束
	shutdownChan chan bool          // 关闭信号通道 - 用于通知主动关闭

	// === 监控组件 === //
	heartbeatTicker *time.Ticker // 心跳计时器 - 定期检查插件健康状态
}

// NewPluginHost 创建新的插件主机实例
// 初始化主机所需的所有组件：注册表、服务、上下文等
// 参数:
//   - config: 主机配置，如为nil则使用默认配置
//
// 返回:
//   - *PluginHost: 初始化完成的主机实例
//   - error: 创建过程中的错误
func NewPluginHost(config *HostConfig) (*PluginHost, error) {
	// 如果没有提供配置，使用默认配置
	if config == nil {
		config = DefaultHostConfig()
	}

	// 创建可取消的上下文，用于统一控制所有子操作
	ctx, cancel := context.WithCancel(context.Background())

	// 初始化主机结构体
	host := &PluginHost{
		config:        config,                        // 保存配置信息
		registry:      NewPluginRegistry(),           // 创建插件注册表
		hostFunctions: make(map[string]HostFunction), // 初始化主机函数映射
		ctx:           ctx,                           // 设置上下文
		cancel:        cancel,                        // 设置取消函数
		shutdownChan:  make(chan bool, 1),            // 创建关闭信号通道
	}

	// 创建主机服务实例，用于处理插件请求
	host.hostService = newHostService(host)

	// 注册默认的主机函数（系统时间、系统信息等）
	host.registerDefaultFunctions()

	return host, nil // 返回初始化完成的主机
}

// Start 启动插件主机
func (ph *PluginHost) Start() error {
	log.Printf("🚀 启动插件主机...")

	// 启动gRPC服务器
	if err := ph.startGrpcServer(); err != nil {
		return fmt.Errorf("启动gRPC服务器失败: %v", err)
	}

	// 启动监控
	ph.startMonitoring()

	log.Printf("✅ 插件主机启动完成，监听端口: %d", ph.actualPort)
	return nil
}

// Stop 停止插件主机
func (ph *PluginHost) Stop() {
	log.Printf("🛑 停止插件主机...")

	// 停止所有插件
	ph.StopAllPlugins()

	// 停止监控
	if ph.heartbeatTicker != nil {
		ph.heartbeatTicker.Stop()
	}

	// 停止gRPC服务器
	if ph.grpcServer != nil {
		ph.grpcServer.GracefulStop()
	}

	// 关闭监听器
	if ph.listener != nil {
		ph.listener.Close()
	}

	// 取消上下文
	ph.cancel()

	// 等待所有协程结束
	ph.wg.Wait()

	log.Printf("✅ 插件主机已安全停止")
}

// Wait 等待退出信号
func (ph *PluginHost) Wait() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigChan:
		log.Printf("📥 收到系统退出信号...")
	case <-ph.shutdownChan:
		log.Printf("📥 收到程序关闭信号...")
	}

	ph.Stop()
}

// LoadPlugin 加载插件
func (ph *PluginHost) LoadPlugin(executablePath string) (*PluginInfo, error) {
	log.Printf("📦 正在加载插件: %s", executablePath)

	// 获取插件信息
	pluginBasicInfo, err := ph.GetPluginInfo(executablePath)
	if err != nil {
		return nil, fmt.Errorf("获取插件信息失败: %v", err)
	}

	// 使用插件固定的ID，如果有的话
	pluginID := pluginBasicInfo.ID
	if pluginID == "" {
		// 如果插件没有固定ID，则生成一个
		pluginID = fmt.Sprintf("plugin-%d", time.Now().UnixNano())
	}

	pluginInfo := &PluginInfo{
		ID:             pluginID, // 使用插件固定的ID
		Name:           pluginBasicInfo.Name,
		Version:        pluginBasicInfo.Version,
		Description:    pluginBasicInfo.Description,
		Capabilities:   pluginBasicInfo.Capabilities,
		Functions:      pluginBasicInfo.Functions,
		ExecutablePath: executablePath,
		Status:         StatusStopped,
		AutoRestart:    ph.config.AutoRestartPlugin,
		MaxRestarts:    3,
		RestartCount:   0,
	}

	// 注册到注册表
	ph.registry.Register(pluginInfo)

	log.Printf("✅ 插件已加载（ID: %s）", pluginID)
	return pluginInfo, nil
}

// StartPlugin 启动插件
func (ph *PluginHost) StartPlugin(pluginID string) error {
	plugin, exists := ph.registry.Get(pluginID)
	if !exists {
		return fmt.Errorf("插件 %s 不存在", pluginID)
	}

	if plugin.Status == StatusRunning {
		return fmt.Errorf("插件 %s 已在运行中", pluginID)
	}

	log.Printf("🚀 正在启动插件: %s", plugin.ExecutablePath)
	return ph.startPluginProcess(plugin)
}

// StartPluginByPath 根据路径启动插件
func (ph *PluginHost) StartPluginByPath(executablePath string) (*PluginInfo, error) {
	// 查找对应的插件
	plugins := ph.registry.List()
	var targetPlugin *PluginInfo
	for _, plugin := range plugins {
		if plugin.ExecutablePath == executablePath {
			targetPlugin = plugin
			return plugin, nil
		}
	}

	if targetPlugin == nil {
		// 如果没有找到，先加载
		var err error
		targetPlugin, err = ph.LoadPlugin(executablePath)
		if err != nil {
			return nil, err
		}
	}

	err := ph.StartPlugin(targetPlugin.ID)
	return targetPlugin, err
}

// StopPlugin 停止插件
func (ph *PluginHost) StopPlugin(pluginID string) error {
	plugin, exists := ph.registry.Get(pluginID)
	if !exists {
		return fmt.Errorf("插件 %s 不存在", pluginID)
	}

	log.Printf("🛑 正在停止插件: %s", pluginID)
	return ph.stopPluginProcess(plugin)
}

// StopAllPlugins 停止所有插件
func (ph *PluginHost) StopAllPlugins() {
	plugins := ph.registry.List()
	for _, plugin := range plugins {
		if plugin.Status == StatusRunning {
			ph.stopPluginProcess(plugin)
		}
	}
}

// GetPlugin 获取插件信息
func (ph *PluginHost) GetPlugin(pluginID string) (*PluginInfo, bool) {
	return ph.registry.Get(pluginID)
}

// GetAllPlugins 获取所有插件
func (ph *PluginHost) GetAllPlugins() []*PluginInfo {
	return ph.registry.List()
}

// CallPluginFunction 调用插件函数
func (ph *PluginHost) CallPluginFunction(pluginID string, functionName string, params []*proto.Parameter) (*proto.CallResponse, error) {
	plugin, exists := ph.registry.Get(pluginID)
	if !exists {
		return nil, fmt.Errorf("插件 %s 不存在", pluginID)
	}

	if plugin.Status != StatusRunning {
		return nil, fmt.Errorf("插件 %s 状态异常: %s", pluginID, plugin.Status)
	}

	if plugin.Client == nil {
		return nil, fmt.Errorf("插件 %s gRPC客户端未连接", pluginID)
	}

	// 创建请求
	req := &proto.CallRequest{
		FunctionName: functionName,
		Parameters:   params,
		RequestId:    fmt.Sprintf("host-%d", time.Now().UnixNano()),
		Metadata: map[string]string{
			"source":    "host",
			"timestamp": fmt.Sprintf("%d", time.Now().Unix()),
		},
	}

	// 调用插件函数
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return plugin.Client.CallPluginFunction(ctx, req)
}

// SendMessageToPlugin 向插件发送消息
func (ph *PluginHost) SendMessageToPlugin(pluginID string, messageType string, content string, metadata map[string]string) (*proto.MessageResponse, error) {
	plugin, exists := ph.registry.Get(pluginID)
	if !exists {
		return nil, fmt.Errorf("插件 %s 不存在", pluginID)
	}

	if plugin.Status != StatusRunning {
		return nil, fmt.Errorf("插件 %s 状态异常: %s", pluginID, plugin.Status)
	}

	message := &proto.MessageRequest{
		MessageId:   fmt.Sprintf("msg-%d", time.Now().UnixNano()),
		MessageType: messageType,
		Content:     content,
		Timestamp:   time.Now().Unix(),
		Metadata:    metadata,
	}

	// 创建流式连接
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	stream, err := plugin.Client.ReceiveMessages(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建消息流失败: %v", err)
	}

	// 发送消息
	if err := stream.Send(message); err != nil {
		return nil, fmt.Errorf("发送消息失败: %v", err)
	}

	// 关闭发送并接收响应
	return stream.CloseAndRecv()
}

// BroadcastMessage 广播消息到所有插件
func (ph *PluginHost) BroadcastMessage(messageType string, content string, metadata map[string]string) map[string]*proto.MessageResponse {
	plugins := ph.registry.List()
	results := make(map[string]*proto.MessageResponse)

	for _, plugin := range plugins {
		if plugin.Status == StatusRunning {
			resp, err := ph.SendMessageToPlugin(plugin.ID, messageType, content, metadata)
			if err != nil {
				log.Printf("向插件 %s 广播消息失败: %v", plugin.ID, err)
				continue
			}
			results[plugin.ID] = resp
		}
	}

	return results
}

// RegisterHostFunction 注册主机函数
func (ph *PluginHost) RegisterHostFunction(name string, fn HostFunction) {
	ph.hostFunctions[name] = fn
	log.Printf("已注册主机函数: %s", name)
}

// GetPluginInfo 获取插件信息（不加载插件）
func (ph *PluginHost) GetPluginInfo(executablePath string) (*PluginBasicInfo, error) {
	cmd := exec.Command(executablePath, "--info")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("获取插件信息失败: %v", err)
	}

	var info PluginBasicInfo
	err = json.Unmarshal(output, &info)
	if err != nil {
		return nil, fmt.Errorf("解析插件信息失败: %v", err)
	}

	return &info, nil
}

// GetActualPort 获取实际使用的端口
func (ph *PluginHost) GetActualPort() int {
	return ph.actualPort
}

// 内部方法

// startGrpcServer 启动gRPC服务器（自适应端口）
func (ph *PluginHost) startGrpcServer() error {
	startPort := ph.config.PortRange[0]
	maxPort := ph.config.PortRange[1]

	if ph.config.Port > 0 {
		startPort = ph.config.Port
		maxPort = ph.config.Port
	}

	var listener net.Listener
	var err error
	var actualPort int

	// 自动寻找可用端口
	for port := startPort; port <= maxPort; port++ {
		address := fmt.Sprintf(":%d", port)
		listener, err = net.Listen("tcp", address)
		if err == nil {
			actualPort = port
			log.Printf("🎯 找到可用端口: %d", actualPort)
			break
		}
		log.Printf("端口 %d 被占用，尝试下一个...", port)
	}

	if listener == nil {
		return fmt.Errorf("无法找到可用端口 (尝试范围: %d-%d)", startPort, maxPort)
	}

	ph.listener = listener
	ph.actualPort = actualPort
	ph.grpcServer = grpc.NewServer()

	// 注册gRPC服务
	proto.RegisterHostServiceServer(ph.grpcServer, ph.hostService)

	// 启动服务器
	ph.wg.Add(1)
	go func() {
		defer ph.wg.Done()
		log.Printf("🌐 gRPC服务器启动中，监听端口: %d", actualPort)
		if err := ph.grpcServer.Serve(listener); err != nil {
			log.Printf("gRPC服务器错误: %v", err)
		}
	}()

	return nil
}

// startPluginProcess 启动插件进程
func (ph *PluginHost) startPluginProcess(plugin *PluginInfo) error {
	plugin.Status = StatusStarting

	// 设置环境变量
	cmd := exec.Command(plugin.ExecutablePath)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("PLUGIN_ID=%s", plugin.ID),
		fmt.Sprintf("HOST_GRPC_ADDRESS=localhost:%d", ph.actualPort),
	)

	// 启动进程
	err := cmd.Start()
	if err != nil {
		plugin.Status = StatusError
		return fmt.Errorf("启动插件进程失败: %v", err)
	}

	plugin.Process = cmd.Process
	plugin.Command = cmd
	plugin.StartTime = time.Now()

	log.Printf("插件进程已启动: %s, PID: %d", plugin.ExecutablePath, plugin.Process.Pid)

	// 启动进程监控
	ph.wg.Add(1)
	go ph.monitorPluginProcess(plugin)

	return nil
}

// stopPluginProcess 停止插件进程
func (ph *PluginHost) stopPluginProcess(plugin *PluginInfo) error {
	plugin.Status = StatusStopping

	// 关闭gRPC连接
	if plugin.Connection != nil {
		plugin.Connection.Close()
		plugin.Connection = nil
		plugin.Client = nil
	}

	// 终止进程
	if plugin.Process != nil {
		err := plugin.Process.Kill()
		plugin.Process = nil
		if err != nil {
			log.Printf("终止插件进程失败: %v", err)
		}
	}

	plugin.Status = StatusStopped
	log.Printf("插件已停止: %s", plugin.ID)

	return nil
}

// monitorPluginProcess 监控插件进程
func (ph *PluginHost) monitorPluginProcess(plugin *PluginInfo) {
	defer ph.wg.Done()

	if plugin.Command != nil {
		// 等待进程结束
		err := plugin.Command.Wait()
		if err != nil && plugin.Status != StatusStopping {
			log.Printf("插件进程异常退出: %s, 错误: %v", plugin.ID, err)
			plugin.Status = StatusCrashed
		} else {
			log.Printf("插件进程正常退出: %s", plugin.ID)
			plugin.Status = StatusStopped
		}

		// 检查是否需要自动重启
		if plugin.AutoRestart && plugin.Status == StatusCrashed && plugin.RestartCount < plugin.MaxRestarts {
			plugin.RestartCount++
			log.Printf("自动重启插件: %s (第 %d 次)", plugin.ID, plugin.RestartCount)
			time.Sleep(5 * time.Second) // 等待一段时间再重启
			ph.startPluginProcess(plugin)
		}
	}
}

// startMonitoring 启动监控
func (ph *PluginHost) startMonitoring() {
	ph.heartbeatTicker = time.NewTicker(ph.config.HeartbeatInterval)

	ph.wg.Add(1)
	go func() {
		defer ph.wg.Done()
		defer ph.heartbeatTicker.Stop()

		for {
			select {
			case <-ph.ctx.Done():
				return
			case <-ph.heartbeatTicker.C:
				ph.checkPluginsHealth()
			}
		}
	}()
}

// checkPluginsHealth 检查插件健康状态
func (ph *PluginHost) checkPluginsHealth() {
	now := time.Now()
	plugins := ph.registry.List()

	for _, plugin := range plugins {
		if plugin.Status == StatusRunning {
			// 检查心跳超时
			if now.Sub(plugin.LastHeartbeat) > ph.config.HeartbeatInterval*time.Duration(ph.config.MaxHeartbeatMiss) {
				log.Printf("插件 %s 心跳超时，标记为崩溃", plugin.ID)
				plugin.Status = StatusCrashed

				// 检查是否需要自动重启
				if plugin.AutoRestart && plugin.RestartCount < plugin.MaxRestarts {
					plugin.RestartCount++
					log.Printf("自动重启心跳超时的插件: %s (第 %d 次)", plugin.ID, plugin.RestartCount)
					ph.startPluginProcess(plugin)
				}
			}
		}
	}
}

// registerDefaultFunctions 注册默认主机函数
func (ph *PluginHost) registerDefaultFunctions() {
	ph.RegisterHostFunction("GetSystemTime", ph.getSystemTime)
	ph.RegisterHostFunction("GetSystemInfo", ph.getSystemInfo)
	ph.RegisterHostFunction("GetPluginList", ph.getPluginList)
}

// 默认主机函数实现

func (ph *PluginHost) getSystemTime(ctx context.Context, params []*proto.Parameter) (*proto.Parameter, error) {
	return &proto.Parameter{
		Name:  "system_time",
		Type:  proto.ParameterType_STRING,
		Value: time.Now().Format("2006-01-02 15:04:05"),
	}, nil
}

func (ph *PluginHost) getSystemInfo(ctx context.Context, params []*proto.Parameter) (*proto.Parameter, error) {
	info := map[string]interface{}{
		"os":        "windows",
		"plugins":   ph.registry.Count(),
		"uptime":    time.Now().Format("2006-01-02 15:04:05"),
		"grpc_port": ph.actualPort,
	}

	jsonData, _ := json.Marshal(info)
	return &proto.Parameter{
		Name:  "system_info",
		Type:  proto.ParameterType_JSON,
		Value: string(jsonData),
	}, nil
}

func (ph *PluginHost) getPluginList(ctx context.Context, params []*proto.Parameter) (*proto.Parameter, error) {
	plugins := ph.registry.List()
	pluginData := make([]map[string]interface{}, len(plugins))

	for i, plugin := range plugins {
		pluginData[i] = map[string]interface{}{
			"id":     plugin.ID,
			"name":   plugin.Name,
			"status": string(plugin.Status),
			"port":   plugin.Port,
		}
	}

	jsonData, _ := json.Marshal(pluginData)
	return &proto.Parameter{
		Name:  "plugin_list",
		Type:  proto.ParameterType_JSON,
		Value: string(jsonData),
	}, nil
}
