package contract

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// WorkerInfo Worker 心跳数据，由 WorkerAgent 每 15s 上报到 Redis DB 1
// JSON tag 与 Redis 中存储的字段名一致
type WorkerInfo struct {
	ID          string    `json:"id"`           // Worker UUIDv7 标识
	Addr        string    `json:"addr"`         // gRPC 地址 IP:Port，Control Plane 用于调用 Worker
	Weight      int       `json:"weight"`       // 调度权重 = CPU核心数 × 30
	Sessions    int       `json:"sessions"`     // 当前运行的 EmuRunner 会话数
	MaxSessions int       `json:"max_sessions"` // 最大会话数 = weight（硬上限）
	CPUPercent  float64   `json:"cpu_percent"`  // CPU 使用率百分比，仅监控告警，不参与调度评分
	MemPercent  float64   `json:"mem_percent"`  // 内存使用率百分比，仅监控告警，不参与调度评分
	StartedAt   time.Time `json:"started_at"`   // Worker 启动时间
}

// WorkerRegistry Worker 注册中心接口，基于 Redis 实现
// WorkerAgent 通过 Redis SET key TTL 注册自身，Control Plane 通过 SCAN 发现可用 Worker
type WorkerRegistry interface {
	// ListAlive 返回所有存活的 Worker 列表（从 Redis 读取未过期的 key）
	ListAlive(ctx context.Context) ([]WorkerInfo, error)
}

// Scheduler Worker 调度器接口，加权最低负载优先选择最优 Worker
type Scheduler interface {
	// SelectWorker 从注册中心中选择最优 Worker
	// 策略: score = sessions / weight，选 score 最小且 sessions < max_sessions 的 Worker
	SelectWorker(ctx context.Context, registry WorkerRegistry) (*WorkerInfo, error)
}

// StartGameRequest 启动游戏会话请求（contract 层纯 Go DTO，不依赖 proto 包）
type StartGameRequest struct {
	RoomID       uuid.UUID // 房间 ID
	RomPath      string    // MinIO 中的 ROM 路径（仅供日志/调试）
	RomURL       string    // MinIO 预签名下载 URL，Worker 用于下载 ROM 文件到本地临时目录
	HostUserID   uuid.UUID // 房主用户 ID，Worker 据此生成房主专属 player token
	EmulatorType string    // 模拟器类型："nes" | "gba" | "dos"
	MaxPorts     int32     // 最大手柄端口数
}

// StartGameResponse 启动游戏会话响应
type StartGameResponse struct {
	LivekitToken string `json:"livekit_token"` // EmuRunner token（identity="emurunner"）
	LivekitRoom  string `json:"livekit_room"`  // LiveKit 房间名（= room_id）
	LivekitURL   string `json:"livekit_url"`   // LiveKit 服务端地址，前端连接用
	HostToken    string `json:"host_token"`    // 房主专属 player token（identity="player:{host_id}"）
}

// WorkerClient gRPC 客户端接口，Control Plane 通过此接口调用 Worker Agent
type WorkerClient interface {
	// StartGame 调用 Worker 启动游戏会话
	StartGame(ctx context.Context, workerAddr string, req StartGameRequest) (*StartGameResponse, error)
	// StopGame 调用 Worker 停止游戏会话
	StopGame(ctx context.Context, workerAddr string, roomID uuid.UUID) error
	// GeneratePlayerToken 调用 Worker 为指定玩家生成独立的 LiveKit token
	GeneratePlayerToken(ctx context.Context, workerAddr string, roomID uuid.UUID, userID uuid.UUID) (string, error)
	// UpdatePortMapping 通知 Worker 广播最新的 port → player identity 映射
	// mapping: key=port 编号, value=LiveKit identity（格式 "player:{user_id}"）
	UpdatePortMapping(ctx context.Context, workerAddr string, roomID uuid.UUID, mapping map[int32]string) error
	// PauseGame 通知 Worker 暂停指定房间的模拟器运行
	PauseGame(ctx context.Context, workerAddr string, roomID uuid.UUID) error
	// ResumeGame 通知 Worker 继续指定房间的模拟器运行
	ResumeGame(ctx context.Context, workerAddr string, roomID uuid.UUID) error
}
