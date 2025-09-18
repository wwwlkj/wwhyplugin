package wwplugin

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/wwwlkj/wwhyplugin/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// hostService 主机服务实现
type hostService struct {
	proto.UnimplementedHostServiceServer
	host *PluginHost
}

// newHostService 创建主机服务
func newHostService(host *PluginHost) *hostService {
	return &hostService{
		host: host,
	}
}

// RegisterPlugin 插件注册
func (hs *hostService) RegisterPlugin(ctx context.Context, req *proto.RegisterRequest) (*proto.RegisterResponse, error) {
	log.Printf("插件注册请求: %s (%s)", req.PluginName, req.PluginId)

	// 查找对应的插件（通过临时ID）
	plugins := hs.host.registry.List()
	var targetPlugin *PluginInfo
	for _, plugin := range plugins {
		// 匹配临时ID或者相同路径
		if plugin.ID == req.PluginId || plugin.Status == StatusStarting {
			targetPlugin = plugin
			break
		}
	}

	if targetPlugin == nil {
		return &proto.RegisterResponse{
			Success: false,
			Message: "未找到对应的插件",
		}, nil
	}

	// 更新插件信息
	oldID := targetPlugin.ID
	targetPlugin.ID = req.PluginId
	targetPlugin.Name = req.PluginName
	targetPlugin.Version = req.Version
	targetPlugin.Description = req.Description
	targetPlugin.Port = req.Port
	targetPlugin.Capabilities = req.Capabilities
	targetPlugin.Status = StatusStarting
	targetPlugin.LastHeartbeat = time.Now()

	// 如果ID发生变化，需要重新注册
	if oldID != req.PluginId {
		hs.host.registry.Unregister(oldID)
		hs.host.registry.Register(targetPlugin)
		log.Printf("🎆 插件注册: %s -> %s", oldID, req.PluginId)
	}

	// 建立到插件的gRPC连接
	go hs.connectToPlugin(targetPlugin)

	log.Printf("✅ 插件已注册: %s (localhost:%d)", req.PluginName, req.Port)

	return &proto.RegisterResponse{
		Success: true,
		Message: "注册成功",
		HostId:  fmt.Sprintf("host-%d", time.Now().Unix()),
	}, nil
}

// Heartbeat 插件心跳
func (hs *hostService) Heartbeat(ctx context.Context, req *proto.HeartbeatRequest) (*proto.HeartbeatResponse, error) {
	// 更新插件心跳时间
	plugin, exists := hs.host.registry.Get(req.PluginId)
	if exists {
		plugin.LastHeartbeat = time.Now()
	}

	return &proto.HeartbeatResponse{
		Success:         true,
		Message:         "心跳正常",
		ServerTimestamp: time.Now().Unix(),
	}, nil
}

// CallHostFunction 插件调用主机函数
func (hs *hostService) CallHostFunction(ctx context.Context, req *proto.CallRequest) (*proto.CallResponse, error) {
	// 检查是否是插件间调用请求
	if targetPluginID, exists := req.Metadata["target_plugin_id"]; exists {
		// 这是插件间调用请求，转发到目标插件
		return hs.callPluginFunction(ctx, req, targetPluginID)
	}

	// 正常的主机函数调用
	log.Printf("插件调用主机函数: %s (请求ID: %s)", req.FunctionName, req.RequestId)

	// 查找函数
	fn, exists := hs.host.hostFunctions[req.FunctionName]
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

// callPluginFunction 插件间调用函数（新增）
// 允许一个插件通过主机调用另一个插件的函数
func (hs *hostService) callPluginFunction(ctx context.Context, req *proto.CallRequest, targetPluginID string) (*proto.CallResponse, error) {
	// 获取调用者插件ID
	sourcePluginID := req.Metadata["plugin_id"]
	log.Printf("插件间调用: %s -> %s.%s", sourcePluginID, targetPluginID, req.FunctionName)

	// 获取目标插件信息
	targetPlugin, exists := hs.host.registry.Get(targetPluginID)
	if !exists {
		return &proto.CallResponse{
			Success:   false,
			Message:   fmt.Sprintf("目标插件 %s 不存在", targetPluginID),
			ErrorCode: "TARGET_PLUGIN_NOT_FOUND",
			RequestId: req.RequestId,
		}, nil
	}

	// 检查目标插件状态
	if targetPlugin.Status != StatusRunning {
		return &proto.CallResponse{
			Success:   false,
			Message:   fmt.Sprintf("目标插件 %s 状态异常: %s", targetPluginID, targetPlugin.Status),
			ErrorCode: "TARGET_PLUGIN_NOT_RUNNING",
			RequestId: req.RequestId,
		}, nil
	}

	// 调用目标插件函数
	callCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 更新元数据，标明这是插件间调用
	enhancedReq := &proto.CallRequest{
		FunctionName: req.FunctionName,
		Parameters:   req.Parameters,
		RequestId:    req.RequestId,
		Metadata: map[string]string{
			"source":        "inter_plugin",
			"source_plugin": sourcePluginID,
			"target_plugin": targetPluginID,
			"timestamp":     fmt.Sprintf("%d", time.Now().Unix()),
			"via_host":      "true",
		},
	}

	resp, err := targetPlugin.Client.CallPluginFunction(callCtx, enhancedReq)
	if err != nil {
		log.Printf("插件间调用失败: %v", err)
		return &proto.CallResponse{
			Success:   false,
			Message:   fmt.Sprintf("调用目标插件函数失败: %v", err),
			ErrorCode: "INTER_PLUGIN_CALL_ERROR",
			RequestId: req.RequestId,
		}, nil
	}

	log.Printf("插件间调用成功: %s -> %s.%s", sourcePluginID, targetPluginID, req.FunctionName)
	return resp, nil
}

// ReportLog 插件上报日志
func (hs *hostService) ReportLog(ctx context.Context, req *proto.LogRequest) (*proto.LogResponse, error) {
	// 格式化日志信息
	levelStr := req.Level.String()
	timestamp := time.Unix(req.Timestamp, 0).Format("2006-01-02 15:04:05")

	// 输出日志
	log.Printf("[%s] [%s] [%s] %s", timestamp, levelStr, req.PluginId, req.Message)

	return &proto.LogResponse{
		Success: true,
	}, nil
}

// connectToPlugin 连接到插件
func (hs *hostService) connectToPlugin(plugin *PluginInfo) {
	// 等待一段时间让插件启动gRPC服务
	time.Sleep(2 * time.Second)

	address := fmt.Sprintf("localhost:%d", plugin.Port)
	log.Printf("连接到插件: %s (%s)", plugin.ID, address)

	// 建立gRPC连接
	conn, err := grpc.Dial(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Printf("连接插件失败: %v", err)
		plugin.Status = StatusError
		return
	}

	plugin.Connection = conn
	plugin.Client = proto.NewPluginServiceClient(conn)

	log.Printf("✅ 已连接到插件: %s", plugin.ID)
	plugin.Status = StatusRunning
}
