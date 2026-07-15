package grpc

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/StellarisJAY/cloudemu/internal/emurunner"
	workerpb "github.com/StellarisJAY/cloudemu/internal/proto/worker"
	"github.com/StellarisJAY/cloudemu/internal/worker"
)

// WorkerServer WorkerAgent gRPC 服务端实现
// 实现 proto/worker.proto 定义的 WorkerAgentServer 接口
// Control Plane 通过 gRPC 调用此服务来管理游戏会话
type WorkerServer struct {
	workerpb.UnimplementedWorkerAgentServer
	sessions *worker.SessionManager
	livekit  *worker.LiveKitManager
	hb       *worker.Heartbeat // 用于更新心跳中的 sessions 计数
}

// NewWorkerServer 创建 WorkerServer 实例
func NewWorkerServer(
	sessions *worker.SessionManager,
	livekit *worker.LiveKitManager,
	hb *worker.Heartbeat,
) *WorkerServer {
	return &WorkerServer{
		sessions: sessions,
		livekit:  livekit,
		hb:       hb,
	}
}

// StartGame 启动游戏会话
// 流程：创建 LiveKit 房间 → 生成 EmuRunner token → 返回 token 给 Control Plane → 启动 EmuRunner 子进程
func (s *WorkerServer) StartGame(ctx context.Context, req *workerpb.StartGameRequest) (*workerpb.StartGameResponse, error) {
	roomID := req.GetRoomId()
	romPath := req.GetRomPath()
	emulatorType := req.GetEmulatorType()

	slog.Info("StartGame received",
		"room_id", roomID,
		"emulator_type", emulatorType,
	)

	// 1. 创建 LiveKit 房间（设置空房间超时 60s）
	if err := s.livekit.CreateRoom(ctx, roomID); err != nil {
		slog.Error("failed to create livekit room", "room_id", roomID, "error", err)
		return nil, err
	}

	// 2. 生成 LiveKit token（EmuRunner 专属，identity="emurunner"，canPublish=true）
	emuToken, err := s.livekit.GenerateToken(roomID, "emurunner", true)
	if err != nil {
		slog.Error("failed to generate emurunner token", "room_id", roomID, "error", err)
		_ = s.livekit.DeleteRoom(context.Background(), roomID)
		return nil, err
	}

	// 3. 为房主生成专属 player token（identity="player:{host_id}"，canPublish=false）
	hostUserID := req.GetHostUserId()
	hostToken, err := s.livekit.GenerateToken(roomID, "player:"+hostUserID, false)
	if err != nil {
		slog.Error("failed to generate host token", "room_id", roomID, "error", err)
		_ = s.livekit.DeleteRoom(context.Background(), roomID)
		return nil, err
	}

	// 4. 启动 EmuRunner 子进程（传入 EmuRunner 专属 token + 房主用户 ID）
	session, err := s.sessions.Start(roomID, emuToken, romPath, req.GetRomUrl(), emulatorType, req.GetHostUserId())
	if err != nil {
		slog.Error("failed to start emurunner", "room_id", roomID, "error", err)
		_ = s.livekit.DeleteRoom(context.Background(), roomID)
		return nil, err
	}

	// 5. 更新心跳中的会话计数
	s.hb.UpdateSessions(1)

	slog.Info("game started",
		"room_id", roomID,
		"session_started_at", session.StartedAt,
	)

	return &workerpb.StartGameResponse{
		LivekitToken: emuToken,
		LivekitRoom:  roomID,
		LivekitUrl:   s.livekit.HostURL(),
		HostToken:    hostToken,
	}, nil
}

// StopGame 停止游戏会话
// 流程：停止 EmuRunner 子进程 → 清理 LiveKit 房间 → 更新心跳 sessions 计数
func (s *WorkerServer) StopGame(ctx context.Context, req *workerpb.StopGameRequest) (*workerpb.StopGameResponse, error) {
	roomID := req.GetRoomId()

	slog.Info("StopGame received", "room_id", roomID)

	// 1. 停止 EmuRunner 子进程
	if err := s.sessions.Stop(roomID); err != nil {
		slog.Warn("failed to stop emurunner", "room_id", roomID, "error", err)
	}

	// 2. 删除 LiveKit 房间
	if err := s.livekit.DeleteRoom(ctx, roomID); err != nil {
		slog.Warn("failed to delete livekit room", "room_id", roomID, "error", err)
	}

	// 3. 更新心跳中的会话计数
	s.hb.UpdateSessions(-1)

	slog.Info("game stopped", "room_id", roomID)

	return &workerpb.StopGameResponse{}, nil
}

// SessionStatus 查询游戏会话运行状态
func (s *WorkerServer) SessionStatus(ctx context.Context, req *workerpb.SessionStatusRequest) (*workerpb.SessionStatusResponse, error) {
	roomID := req.GetRoomId()
	status, uptime := s.sessions.Status(roomID)

	return &workerpb.SessionStatusResponse{
		RoomId:        roomID,
		Status:        status,
		UptimeSeconds: uptime,
	}, nil
}

// GeneratePlayerToken 为指定玩家生成独立的 LiveKit token
// identity 格式："player:{user_id}"，canPublish=false（玩家只订阅视频+发送 DataChannel）
func (s *WorkerServer) GeneratePlayerToken(ctx context.Context, req *workerpb.GeneratePlayerTokenRequest) (*workerpb.GeneratePlayerTokenResponse, error) {
	roomID := req.GetRoomId()
	userID := req.GetUserId()
	identity := "player:" + userID

	token, err := s.livekit.GenerateToken(roomID, identity, false)
	if err != nil {
		slog.Error("failed to generate player token", "room_id", roomID, "user_id", userID, "error", err)
		return nil, err
	}

	return &workerpb.GeneratePlayerTokenResponse{Token: token}, nil
}

// UpdatePortMapping 将最新的 port → player identity 映射通过管道发送给 EmuRunner
// 替代原有的 DataChannel 二进制编码广播
func (s *WorkerServer) UpdatePortMapping(ctx context.Context, req *workerpb.UpdatePortMappingRequest) (*workerpb.UpdatePortMappingResponse, error) {
	roomID := req.GetRoomId()

	session, ok := s.sessions.Get(roomID)
	if !ok {
		return nil, fmt.Errorf("session not found: %s", roomID)
	}

	mapping := make(map[int]string, len(req.GetMapping()))
	for port, identity := range req.GetMapping() {
		mapping[int(port)] = identity
	}

	if _, err := session.SendCommand(emurunner.Cmd{Cmd: "port_map", Mapping: mapping}); err != nil {
		slog.Error("port_map via pipe failed", "room_id", roomID, "error", err)
		return nil, err
	}

	slog.Info("port mapping sent via pipe", "room_id", roomID, "entries", len(mapping))
	return &workerpb.UpdatePortMappingResponse{}, nil
}

// PauseGame 暂停游戏模拟器运行
// 通过 stdin 管道发送 pause 命令到 EmuRunner
func (s *WorkerServer) PauseGame(ctx context.Context, req *workerpb.PauseGameRequest) (*workerpb.PauseGameResponse, error) {
	roomID := req.GetRoomId()
	slog.Info("PauseGame received", "room_id", roomID)

	session, ok := s.sessions.Get(roomID)
	if !ok {
		return nil, fmt.Errorf("session not found: %s", roomID)
	}
	if _, err := session.SendCommand(emurunner.Cmd{Cmd: "pause"}); err != nil {
		slog.Error("pause via pipe failed", "room_id", roomID, "error", err)
		return nil, err
	}

	slog.Info("pause command sent via pipe", "room_id", roomID)
	return &workerpb.PauseGameResponse{}, nil
}

// ResumeGame 继续游戏模拟器运行
// 通过 stdin 管道发送 resume 命令到 EmuRunner
func (s *WorkerServer) ResumeGame(ctx context.Context, req *workerpb.ResumeGameRequest) (*workerpb.ResumeGameResponse, error) {
	roomID := req.GetRoomId()
	slog.Info("ResumeGame received", "room_id", roomID)

	session, ok := s.sessions.Get(roomID)
	if !ok {
		return nil, fmt.Errorf("session not found: %s", roomID)
	}
	if _, err := session.SendCommand(emurunner.Cmd{Cmd: "resume"}); err != nil {
		slog.Error("resume via pipe failed", "room_id", roomID, "error", err)
		return nil, err
	}

	slog.Info("resume command sent via pipe", "room_id", roomID)
	return &workerpb.ResumeGameResponse{}, nil
}

// SaveState 保存游戏存档
// 流程：通过管道命令 EmuRunner 序列化 → 读取 state.dat → PUT 到 MinIO
func (s *WorkerServer) SaveState(ctx context.Context, req *workerpb.SaveStateRequest) (*workerpb.SaveStateResponse, error) {
	roomID := req.GetRoomId()
	slog.Info("SaveState received", "room_id", roomID, "save_state_id", req.GetSaveStateId())

	size, err := s.sessions.SaveStateViaPipe(ctx, roomID, req.GetUploadUrl())
	if err != nil {
		slog.Error("save state via pipe failed", "room_id", roomID, "error", err)
		return nil, err
	}

	return &workerpb.SaveStateResponse{Size: size}, nil
}

// LoadState 读取游戏存档
// 流程：下载状态二进制到 workDir → 通过管道命令 EmuRunner 反序列化 → 等待确认
func (s *WorkerServer) LoadState(ctx context.Context, req *workerpb.LoadStateRequest) (*workerpb.LoadStateResponse, error) {
	roomID := req.GetRoomId()
	slog.Info("LoadState received", "room_id", roomID, "save_state_id", req.GetSaveStateId())

	if err := s.sessions.LoadStateViaPipe(ctx, roomID, req.GetDownloadUrl()); err != nil {
		slog.Error("load state via pipe failed", "room_id", roomID, "error", err)
		return nil, err
	}

	slog.Info("load state completed via pipe", "room_id", roomID)
	return &workerpb.LoadStateResponse{}, nil
}
