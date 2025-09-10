// Package wwplugin 提供插件实现
// 插件运行在独立进程中，通过gRPC与主机通信
package wwplugin

import (
	"context"       // 上下文控制，用于取消和超时管理
	"encoding/json" // JSON编解码，用于插件信息序列化
	"fmt"           // 格式化输出，用于错误信息和日志
	"log"           // 日志记录，用于运行时信息输出
	"net"           // 网络操作，用于创建gRPC服务器
	"os"            // 操作系统接口，环境变量和信号处理
	"os/signal"     // 系统信号处理，用于优雅关闭
	"strconv"       // 字符串转换，用于数据类型转换
	"syscall"       // 系统调用，用于信号处理
	"time"          // 时间处理，心跳和超时管理

	"github.com/wwwlkj/wwhyplugin/proto"          // gRPC协议定义
	"google.golang.org/grpc"                      // gRPC框架
	"google.golang.org/grpc/credentials/insecure" // gRPC安全凭据（不加密）
)

// Plugin 插件实例结构体
// 每个插件运行在独立进程中，提供特定功能服务
type Plugin struct {
	// === 基本配置 === //
	config    *PluginConfig             // 插件配置 - 包含名称、版本等信息
	ID        string                    // 插件唯一标识 - 由主机分配或自动生成
	Port      int32                     // 插件服务端口 - 主机用此端口连接插件
	functions map[string]PluginFunction // 插件函数映射 - 插件提供的可调用函数

	// === gRPC 相关 === //
	GrpcServer *grpc.Server            // gRPC服务器 - 提供插件服务接口
	HostConn   *grpc.ClientConn        // 主机连接 - 连接到主机的gRPC客户端
	HostClient proto.HostServiceClient // 主机客户端 - 用于调用主机服务

	// === 控制组件 === //
	ctx               context.Context    // 上下文控制 - 用于统一取消操作
	cancel            context.CancelFunc // 取消函数 - 用于停止所有子操作
	isShuttingDown    bool               // 关闭标志 - 标记插件是否正在关闭
	reconnectInterval time.Duration      // 重连间隔 - 连接断开后的重连等待时间
	maxReconnectTries int                // 最大重连次数 - 0表示无限重连

	// === 消息处理 === //
	messageHandler MessageHandler // 消息处理器 - 处理主机推送的消息
}

// NewPlugin 创建新的插件实例
func NewPlugin(config *PluginConfig) *Plugin {
	if config == nil {
		config = DefaultPluginConfig("UnnamedPlugin", "1.0.0", "A plugin created with WWPlugin")
	}

	ctx, cancel := context.WithCancel(context.Background())

	plugin := &Plugin{
		config:            config,
		functions:         make(map[string]PluginFunction),
		ctx:               ctx,
		cancel:            cancel,
		reconnectInterval: config.ReconnectInterval,
		maxReconnectTries: config.MaxReconnectTries,
	}

	// 生成插件ID
	if plugin.ID == "" {
		plugin.ID = fmt.Sprintf("%s-%d", config.Name, time.Now().Unix())
	}

	return plugin
}

// Start 启动插件
func (p *Plugin) Start() error {
	// 从环境变量获取主机地址
	if hostAddr := os.Getenv("HOST_GRPC_ADDRESS"); hostAddr != "" {
		p.config.HostAddress = hostAddr
	}

	log.Printf("启动插件: %s (ID: %s)", p.config.Name, p.ID)

	// 启动gRPC服务器
	if err := p.startGrpcServer(); err != nil {
		return fmt.Errorf("启动gRPC服务器失败: %v", err)
	}

	// 连接到主机
	if err := p.connectToHost(); err != nil {
		return fmt.Errorf("连接主机失败: %v", err)
	}

	// 注册到主机
	if err := p.registerToHost(); err != nil {
		return fmt.Errorf("注册到主机失败: %v", err)
	}

	// 启动心跳
	go p.startHeartbeat()

	// 启动连接监控
	go p.startConnectionMonitor()

	// 等待信号
	p.waitForSignal()

	return nil
}

// Stop 停止插件
func (p *Plugin) Stop() {
	log.Printf("停止插件: %s", p.config.Name)

	p.isShuttingDown = true

	// 取消上下文
	p.cancel()

	// 停止gRPC服务器
	if p.GrpcServer != nil {
		p.GrpcServer.GracefulStop()
	}

	// 关闭主机连接
	if p.HostConn != nil {
		p.HostConn.Close()
	}

	log.Printf("插件已停止: %s", p.config.Name)
}

// RegisterFunction 注册插件函数
func (p *Plugin) RegisterFunction(name string, fn PluginFunction) {
	p.functions[name] = fn
	log.Printf("已注册插件函数: %s", name)
}

// SetMessageHandler 设置消息处理器
func (p *Plugin) SetMessageHandler(handler MessageHandler) {
	p.messageHandler = handler
}

// CallHostFunction 调用主机函数
func (p *Plugin) CallHostFunction(functionName string, params []*proto.Parameter) (*proto.CallResponse, error) {
	req := &proto.CallRequest{
		FunctionName: functionName,
		Parameters:   params,
		RequestId:    fmt.Sprintf("plugin-%s-%d", p.ID, time.Now().UnixNano()),
		Metadata: map[string]string{
			"source":    "plugin",
			"plugin_id": p.ID,
			"timestamp": strconv.FormatInt(time.Now().Unix(), 10),
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Printf("调用主机函数: %s", functionName)

	resp, err := p.HostClient.CallHostFunction(ctx, req)
	if err != nil {
		log.Printf("调用主机函数失败: %v", err)
		return nil, err
	}

	if resp.Success {
		log.Printf("主机函数调用成功: %s", functionName)
	} else {
		log.Printf("主机函数调用失败: %s", resp.Message)
	}

	return resp, nil
}

// CallOtherPlugin 调用其他插件函数
// 这是插件间调用的核心方法，通过主机作为中介来调用其他插件的函数
func (p *Plugin) CallOtherPlugin(targetPluginID string, functionName string, params []*proto.Parameter) (*proto.CallResponse, error) {
	req := &proto.CallRequest{
		FunctionName: functionName,
		Parameters:   params,
		RequestId:    fmt.Sprintf("inter-plugin-%s-%d", p.ID, time.Now().UnixNano()),
		Metadata: map[string]string{
			"source":           "plugin",
			"plugin_id":        p.ID,
			"target_plugin_id": targetPluginID,
			"call_type":        "inter_plugin",
			"timestamp":        strconv.FormatInt(time.Now().Unix(), 10),
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Printf("调用插件函数: %s -> %s.%s", p.ID, targetPluginID, functionName)

	// 通过主机的CallHostFunction接口转发请求
	resp, err := p.HostClient.CallHostFunction(ctx, req)
	if err != nil {
		log.Printf("调用插件函数失败: %v", err)
		return nil, err
	}

	if resp.Success {
		log.Printf("插件函数调用成功: %s -> %s.%s", p.ID, targetPluginID, functionName)
	} else {
		log.Printf("插件函数调用失败: %s", resp.Message)
	}

	return resp, nil
}

// GetConfig 获取插件配置
func (p *Plugin) GetConfig() *PluginConfig {
	return p.config
}

// GetPluginInfo 获取插件信息（不启动插件服务）
// 这个方法可以在不启动gRPC服务的情况下获取插件的基本信息
func (p *Plugin) GetPluginInfo() *PluginBasicInfo {
	// 如果还没有ID，先生成一个
	if p.ID == "" {
		p.ID = fmt.Sprintf("%s-%d", p.config.Name, time.Now().Unix())
	}

	return &PluginBasicInfo{
		ID:           p.ID,
		Name:         p.config.Name,
		Version:      p.config.Version,
		Description:  p.config.Description,
		Logo:         p.config.Logo,
		Capabilities: p.config.Capabilities,
		Functions:    p.getFunctionList(),
	}
}

// StartWithInfo 以信息查询模式启动
// 用于支持 --info 参数
func (p *Plugin) StartWithInfo() error {
	info := p.GetPluginInfo()

	// 输出JSON格式的插件信息
	jsonData, err := json.Marshal(info)
	if err != nil {
		fmt.Printf("{\"error\":\"序列化失败: %v\"}\n", err)
		return err
	}

	fmt.Println(string(jsonData))
	return nil
}

// PluginService接口实现

// CallPluginFunction 主机调用插件函数
func (p *Plugin) CallPluginFunction(ctx context.Context, req *proto.CallRequest) (*proto.CallResponse, error) {
	log.Printf("收到函数调用请求: %s (请求ID: %s)", req.FunctionName, req.RequestId)

	// 查找函数
	fn, exists := p.functions[req.FunctionName]
	if !exists {
		log.Printf("未找到函数: %s", req.FunctionName)
		return &proto.CallResponse{
			Success:   false,
			Message:   fmt.Sprintf("未找到函数: %s", req.FunctionName),
			ErrorCode: "FUNCTION_NOT_FOUND",
			RequestId: req.RequestId,
		}, nil
	}

	// 调用函数
	result, err := fn(ctx, req.Parameters)
	if err != nil {
		log.Printf("函数调用失败: %v", err)
		return &proto.CallResponse{
			Success:   false,
			Message:   err.Error(),
			ErrorCode: "FUNCTION_ERROR",
			RequestId: req.RequestId,
		}, nil
	}

	log.Printf("函数调用成功: %s", req.FunctionName)
	return &proto.CallResponse{
		Success:   true,
		Message:   "调用成功",
		Result:    result,
		RequestId: req.RequestId,
	}, nil
}

// ReceiveMessages 接收主机推送的消息
func (p *Plugin) ReceiveMessages(stream proto.PluginService_ReceiveMessagesServer) error {
	log.Println("开始接收消息流...")

	var messageCount int32 = 0

	for {
		msg, err := stream.Recv()
		if err != nil {
			break
		}

		messageCount++
		log.Printf("收到消息: %s - %s (ID: %s)", msg.MessageType, msg.Content, msg.MessageId)

		// 处理消息
		p.handleMessage(msg)
	}

	// 发送响应
	return stream.SendAndClose(&proto.MessageResponse{
		Success:        true,
		Message:        "消息处理完成",
		ProcessedCount: messageCount,
	})
}

// GetPluginStatus 获取插件状态
func (p *Plugin) GetPluginStatus(ctx context.Context, req *proto.StatusRequest) (*proto.StatusResponse, error) {
	uptime := time.Since(time.Unix(0, 0)).String() // 简化的运行时间计算

	resp := &proto.StatusResponse{
		Status:          "running",
		Uptime:          uptime,
		ActiveFunctions: make([]string, 0, len(p.functions)),
	}

	// 添加活跃函数列表
	for name := range p.functions {
		resp.ActiveFunctions = append(resp.ActiveFunctions, name)
	}

	// 添加指标信息
	if req.IncludeMetrics {
		resp.Metrics = map[string]string{
			"function_count": fmt.Sprintf("%d", len(p.functions)),
			"plugin_id":      p.ID,
			"port":           fmt.Sprintf("%d", p.Port),
		}
	}

	return resp, nil
}

// Shutdown 插件关闭通知
func (p *Plugin) Shutdown(ctx context.Context, req *proto.ShutdownRequest) (*proto.ShutdownResponse, error) {
	log.Printf("收到关闭请求: %s", req.Reason)

	// 标记正在关闭
	p.isShuttingDown = true

	// 延迟关闭，给当前请求时间完成
	go func() {
		time.Sleep(1 * time.Second)
		p.Stop()
	}()

	return &proto.ShutdownResponse{
		Success: true,
		Message: "插件正在关闭",
	}, nil
}

// 内部方法

// getFunctionList 获取插件注册的函数列表
func (p *Plugin) getFunctionList() []string {
	functions := make([]string, 0, len(p.functions))
	for name := range p.functions {
		functions = append(functions, name)
	}
	return functions
}

// startGrpcServer 启动gRPC服务器
func (p *Plugin) startGrpcServer() error {
	// 创建监听器，自动分配端口
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return err
	}

	// 获取分配的端口
	addr := listener.Addr().(*net.TCPAddr)
	p.Port = int32(addr.Port)

	// 创建gRPC服务器
	p.GrpcServer = grpc.NewServer()

	// 注册插件服务
	proto.RegisterPluginServiceServer(p.GrpcServer, p)

	// 启动服务器
	go func() {
		log.Printf("插件gRPC服务器启动在端口: %d", p.Port)
		if err := p.GrpcServer.Serve(listener); err != nil {
			log.Printf("gRPC服务器错误: %v", err)
		}
	}()

	return nil
}

// connectToHost 连接到主机
func (p *Plugin) connectToHost() error {
	log.Printf("连接到主机: %s", p.config.HostAddress)

	conn, err := grpc.Dial(
		p.config.HostAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return err
	}

	p.HostConn = conn
	p.HostClient = proto.NewHostServiceClient(conn)

	return nil
}

// registerToHost 注册到主机
func (p *Plugin) registerToHost() error {
	log.Printf("向主机注册插件: %s", p.config.Name)

	req := &proto.RegisterRequest{
		PluginId:     p.ID,
		PluginName:   p.config.Name,
		Version:      p.config.Version,
		Description:  p.config.Description,
		Port:         p.Port,
		Capabilities: p.config.Capabilities,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := p.HostClient.RegisterPlugin(ctx, req)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("注册失败: %s", resp.Message)
	}

	log.Printf("插件注册成功: %s", resp.Message)
	return nil
}

// startHeartbeat 启动心跳
func (p *Plugin) startHeartbeat() {
	ticker := time.NewTicker(p.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.sendHeartbeat()
		}
	}
}

// sendHeartbeat 发送心跳
func (p *Plugin) sendHeartbeat() {
	if p.isShuttingDown {
		return
	}

	req := &proto.HeartbeatRequest{
		PluginId:  p.ID,
		Timestamp: time.Now().Unix(),
		Status:    "running",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := p.HostClient.Heartbeat(ctx, req)
	if err != nil {
		log.Printf("⚠️ 发送心跳失败: %v (主机可能已断开连接)", err)
	}
}

// startConnectionMonitor 启动连接监控器
func (p *Plugin) startConnectionMonitor() {
	reconnectTries := 0
	lastHeartbeatSuccess := time.Now()
	checkInterval := 15 * time.Second

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	log.Println("🔍 启动连接监控器...")

	for {
		select {
		case <-p.ctx.Done():
			log.Println("🔍 连接监控器已停止")
			return
		case <-ticker.C:
			if p.isShuttingDown {
				return
			}

			// 检查连接状态
			if p.checkConnectionHealth() {
				lastHeartbeatSuccess = time.Now()
				reconnectTries = 0
			} else {
				if time.Since(lastHeartbeatSuccess) > 30*time.Second {
					log.Printf("⚠️ 检测到主机连接中断，尝试重连... (第 %d 次)", reconnectTries+1)

					if p.attemptReconnect() {
						log.Println("✅ 重连主机成功！")
						lastHeartbeatSuccess = time.Now()
						reconnectTries = 0
					} else {
						reconnectTries++
						log.Printf("❌ 重连失败，将在 %v 后重试", p.reconnectInterval)

						if p.maxReconnectTries > 0 && reconnectTries >= p.maxReconnectTries {
							log.Printf("❌ 超过最大重连次数 (%d)，插件将退出", p.maxReconnectTries)
							p.Stop()
							return
						}

						time.Sleep(p.reconnectInterval)
					}
				}
			}
		}
	}
}

// checkConnectionHealth 检查连接健康状态
func (p *Plugin) checkConnectionHealth() bool {
	if p.HostClient == nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req := &proto.HeartbeatRequest{
		PluginId:  p.ID,
		Timestamp: time.Now().Unix(),
		Status:    "running",
	}

	_, err := p.HostClient.Heartbeat(ctx, req)
	return err == nil
}

// attemptReconnect 尝试重新连接主机
func (p *Plugin) attemptReconnect() bool {
	// 关闭旧连接
	if p.HostConn != nil {
		p.HostConn.Close()
		p.HostConn = nil
		p.HostClient = nil
	}

	// 尝试重新连接
	if err := p.connectToHost(); err != nil {
		log.Printf("重连失败: %v", err)
		return false
	}

	// 尝试重新注册
	if err := p.registerToHost(); err != nil {
		log.Printf("重新注册失败: %v", err)
		return false
	}

	return true
}

// waitForSignal 等待退出信号
func (p *Plugin) waitForSignal() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Println("收到退出信号，开始关闭插件...")

	p.Stop()
}

// handleMessage 处理接收到的消息
func (p *Plugin) handleMessage(msg *proto.MessageRequest) {
	if p.messageHandler != nil {
		p.messageHandler(msg)
	} else {
		// 默认实现：只是记录日志
		log.Printf("处理消息: %s", msg.MessageType)
	}
}
