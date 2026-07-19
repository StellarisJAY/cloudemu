package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/StellarisJAY/cloudemu/internal/control-plane/contract"
	"github.com/StellarisJAY/cloudemu/internal/proto/worker"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// WorkerClient gRPC 客户端实现，Control Plane 通过此客户端调用 Worker Agent
// 按 Worker 地址懒建立连接，复用 grpc.ClientConn
type WorkerClient struct {
	mu      sync.Mutex
	conns   map[string]*grpc.ClientConn // worker addr → gRPC 连接
	timeout time.Duration               // gRPC 调用超时
}

// NewWorkerClient 创建 WorkerClient 实例
// timeout: gRPC 调用超时时间
func NewWorkerClient(timeout time.Duration) *WorkerClient {
	return &WorkerClient{
		conns:   make(map[string]*grpc.ClientConn),
		timeout: timeout,
	}
}

// getConn 获取或创建到指定 Worker 的 gRPC 连接
// 懒建立连接，已连接则复用；连接失败则返回错误，下次调用时重建
func (c *WorkerClient) getConn(addr string) (*grpc.ClientConn, workerpb.WorkerAgentClient, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	conn, ok := c.conns[addr]
	if ok {
		state := conn.GetState()
		if state.String() == "SHUTDOWN" || state.String() == "TRANSIENT_FAILURE" {
			// 连接已关闭或失败，移除并重建
			conn.Close()
			delete(c.conns, addr)
		} else {
			return conn, workerpb.NewWorkerAgentClient(conn), nil
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("dial worker %s: %w", addr, err)
	}

	c.conns[addr] = conn
	return conn, workerpb.NewWorkerAgentClient(conn), nil
}

// StartGame 调用 Worker 启动游戏会话
// 将 contract DTO 转换为 proto 消息，调用 Worker gRPC 后转换回 DTO
func (c *WorkerClient) StartGame(ctx context.Context, workerAddr string, req contract.StartGameRequest) (*contract.StartGameResponse, error) {
	_, client, err := c.getConn(workerAddr)
	if err != nil {
		slog.Error("get grpc conn failed", "worker", workerAddr, "error", err)
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := client.StartGame(ctx, &workerpb.StartGameRequest{
		RoomId:       req.RoomID.String(),
		RomPath:      req.RomPath,
		RomUrl:       req.RomURL,
		HostUserId:   req.HostUserID.String(),
		EmulatorType: req.EmulatorType,
		MaxPorts:     req.MaxPorts,
	})
	if err != nil {
		slog.Error("worker StartGame gRPC failed", "worker", workerAddr, "error", err)
		return nil, fmt.Errorf("worker StartGame: %w", err)
	}

	return &contract.StartGameResponse{
		LivekitToken: resp.GetLivekitToken(),
		LivekitRoom:  resp.GetLivekitRoom(),
		LivekitURL:   resp.GetLivekitUrl(),
		HostToken:    resp.GetHostToken(),
	}, nil
}

// StopGame 调用 Worker 停止游戏会话
func (c *WorkerClient) StopGame(ctx context.Context, workerAddr string, roomID uuid.UUID) error {
	_, client, err := c.getConn(workerAddr)
	if err != nil {
		slog.Warn("get grpc conn failed for StopGame", "worker", workerAddr, "error", err)
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_, err = client.StopGame(ctx, &workerpb.StopGameRequest{
		RoomId: roomID.String(),
	})
	if err != nil {
		slog.Warn("worker StopGame gRPC failed", "worker", workerAddr, "room_id", roomID, "error", err)
		return fmt.Errorf("worker StopGame: %w", err)
	}

	return nil
}

// GeneratePlayerToken 调用 Worker 为指定玩家生成独立的 LiveKit token
func (c *WorkerClient) GeneratePlayerToken(ctx context.Context, workerAddr string, roomID uuid.UUID, userID uuid.UUID) (string, error) {
	_, client, err := c.getConn(workerAddr)
	if err != nil {
		slog.Error("get grpc conn failed for GeneratePlayerToken", "worker", workerAddr, "error", err)
		return "", err
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := client.GeneratePlayerToken(ctx, &workerpb.GeneratePlayerTokenRequest{
		RoomId: roomID.String(),
		UserId: userID.String(),
	})
	if err != nil {
		slog.Warn("worker GeneratePlayerToken gRPC failed", "worker", workerAddr, "room_id", roomID, "error", err)
		return "", fmt.Errorf("worker GeneratePlayerToken: %w", err)
	}

	return resp.GetToken(), nil
}

// UpdatePortMapping 通知 Worker 广播最新的 port → player identity 映射
// mapping 为当前房间所有 role=1 且 port!=null 的全量映射
func (c *WorkerClient) UpdatePortMapping(ctx context.Context, workerAddr string, roomID uuid.UUID, mapping map[int32]string) error {
	_, client, err := c.getConn(workerAddr)
	if err != nil {
		slog.Error("get grpc conn failed for UpdatePortMapping", "worker", workerAddr, "error", err)
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_, err = client.UpdatePortMapping(ctx, &workerpb.UpdatePortMappingRequest{
		RoomId:  roomID.String(),
		Mapping: mapping,
	})
	if err != nil {
		slog.Warn("worker UpdatePortMapping gRPC failed", "worker", workerAddr, "room_id", roomID, "error", err)
		return fmt.Errorf("worker UpdatePortMapping: %w", err)
	}

	return nil
}

// PauseGame 通知 Worker 暂停指定房间的模拟器运行
func (c *WorkerClient) PauseGame(ctx context.Context, workerAddr string, roomID uuid.UUID) error {
	_, client, err := c.getConn(workerAddr)
	if err != nil {
		slog.Error("get grpc conn failed for PauseGame", "worker", workerAddr, "error", err)
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_, err = client.PauseGame(ctx, &workerpb.PauseGameRequest{
		RoomId: roomID.String(),
	})
	if err != nil {
		slog.Warn("worker PauseGame gRPC failed", "worker", workerAddr, "room_id", roomID, "error", err)
		return fmt.Errorf("worker PauseGame: %w", err)
	}

	return nil
}

// ResumeGame 通知 Worker 继续指定房间的模拟器运行
func (c *WorkerClient) ResumeGame(ctx context.Context, workerAddr string, roomID uuid.UUID) error {
	_, client, err := c.getConn(workerAddr)
	if err != nil {
		slog.Error("get grpc conn failed for ResumeGame", "worker", workerAddr, "error", err)
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_, err = client.ResumeGame(ctx, &workerpb.ResumeGameRequest{
		RoomId: roomID.String(),
	})
	if err != nil {
		slog.Warn("worker ResumeGame gRPC failed", "worker", workerAddr, "room_id", roomID, "error", err)
		return fmt.Errorf("worker ResumeGame: %w", err)
	}

	return nil
}

// SaveState 通知 Worker 保存存档：令 EmuRunner 序列化并上传到 MinIO（uploadURL 为预签名 PUT URL）
func (c *WorkerClient) SaveState(ctx context.Context, workerAddr string, roomID, saveStateID uuid.UUID, uploadURL string) (int64, error) {
	_, client, err := c.getConn(workerAddr)
	if err != nil {
		slog.Error("get grpc conn failed for SaveState", "worker", workerAddr, "error", err)
		return 0, err
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := client.SaveState(ctx, &workerpb.SaveStateRequest{
		RoomId:      roomID.String(),
		SaveStateId: saveStateID.String(),
		UploadUrl:   uploadURL,
	})
	if err != nil {
		slog.Warn("worker SaveState gRPC failed", "worker", workerAddr, "room_id", roomID, "error", err)
		return 0, fmt.Errorf("worker SaveState: %w", err)
	}

	return resp.GetSize(), nil
}

// LoadState 通知 Worker 读取存档：下载状态二进制并令 EmuRunner 反序列化（downloadURL 为预签名 GET URL）
func (c *WorkerClient) LoadState(ctx context.Context, workerAddr string, roomID, saveStateID uuid.UUID, downloadURL string) error {
	_, client, err := c.getConn(workerAddr)
	if err != nil {
		slog.Error("get grpc conn failed for LoadState", "worker", workerAddr, "error", err)
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_, err = client.LoadState(ctx, &workerpb.LoadStateRequest{
		RoomId:      roomID.String(),
		SaveStateId: saveStateID.String(),
		DownloadUrl: downloadURL,
	})
	if err != nil {
		slog.Warn("worker LoadState gRPC failed", "worker", workerAddr, "room_id", roomID, "error", err)
		return fmt.Errorf("worker LoadState: %w", err)
	}

	return nil
}

// SwitchRom 通知 Worker 热切换 ROM：下载新 ROM 并令 EmuRunner 重新加载
func (c *WorkerClient) SwitchRom(ctx context.Context, workerAddr string, roomID, romID uuid.UUID, romURL, emulatorType string) error {
	_, client, err := c.getConn(workerAddr)
	if err != nil {
		slog.Error("get grpc conn failed for SwitchRom", "worker", workerAddr, "error", err)
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_, err = client.SwitchRom(ctx, &workerpb.SwitchRomRequest{
		RoomId:       roomID.String(),
		RomUrl:       romURL,
		RomId:        romID.String(),
		EmulatorType: emulatorType,
	})
	if err != nil {
		slog.Warn("worker SwitchRom gRPC failed", "worker", workerAddr, "room_id", roomID, "error", err)
		return fmt.Errorf("worker SwitchRom: %w", err)
	}

	return nil
}

// Close 关闭所有 gRPC 连接
func (c *WorkerClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for addr, conn := range c.conns {
		if err := conn.Close(); err != nil {
			slog.Warn("close grpc conn failed", "addr", addr, "error", err)
		}
	}
	c.conns = make(map[string]*grpc.ClientConn)
}
