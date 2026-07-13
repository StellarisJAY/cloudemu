// CloudEmu Worker Agent 服务入口
// 负责：Redis 自注册 + 心跳续期、gRPC Server（供 Control Plane 调用）、EmuRunner 子进程管理
package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/StellarisJAY/cloudemu/internal/proto/worker"
	"github.com/StellarisJAY/cloudemu/internal/worker"
	workergrpc "github.com/StellarisJAY/cloudemu/internal/worker/grpc"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
)

func main() {
	cfg := worker.MustLoad()

	// 初始化基础日志（MVP 阶段使用 TextHandler 输出到 stderr）
	var level slog.Level
	switch cfg.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))

	// 检测硬件计算调度权重
	cpuCores := runtime.NumCPU()
	weight := cpuCores * 30

	// 生成 Worker 唯一标识（UUIDv7，按项目约定由应用层生成）
	workerID := uuid.Must(uuid.NewV7()).String()

	// 连接 Redis DB 1（Worker 注册与调度专用，与业务数据 DB 0 隔离）
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPass,
		DB:       cfg.RedisDB,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		slog.Error("redis connect failed", "addr", cfg.RedisAddr, "db", cfg.RedisDB, "error", err)
		os.Exit(1)
	}

	// 构造心跳数据
	heartbeatData := worker.HeartbeatData{
		ID:          workerID,
		Addr:        cfg.Addr,
		Weight:      weight,
		Sessions:    0,
		MaxSessions: weight,
		CPUPercent:  0, // MVP 阶段不采集，后续通过 gopsutil 补充
		MemPercent:  0,
		StartedAt:   time.Now(),
	}

	// 启动心跳注册
	hb := &worker.Heartbeat{}
	if err := hb.Start(context.Background(), rdb, heartbeatData); err != nil {
		slog.Error("heartbeat start failed", "error", err)
		os.Exit(1)
	}

	// 创建 LiveKit 管理器（用于创建房间 + 生成 token）
	livekitMgr := worker.NewLiveKitManager(cfg.LiveKitHost, cfg.LiveKitAPIKey, cfg.LiveKitAPISecret)

	// 创建会话管理器（管理 EmuRunner 子进程生命周期）
	sessionMgr := worker.NewSessionManager(cfg.EmuRunnerPath, cfg.LiveKitHost)

	// 创建 gRPC Server
	grpcSrv := grpc.NewServer()
	workerpb.RegisterWorkerAgentServer(grpcSrv, workergrpc.NewWorkerServer(sessionMgr, livekitMgr, hb))

	// 启动 gRPC 监听
	lis, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		slog.Error("gRPC listen failed", "addr", cfg.Addr, "error", err)
		os.Exit(1)
	}
	go func() {
		slog.Info("gRPC server starting", "addr", cfg.Addr)
		if err := grpcSrv.Serve(lis); err != nil {
			slog.Error("gRPC server error", "error", err)
		}
	}()

	slog.Info("worker agent started",
		"id", workerID,
		"addr", cfg.Addr,
		"cores", cpuCores,
		"weight", weight,
	)

	// 等待终止信号，优雅退出
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh

	slog.Info("worker shutting down", "signal", sig.String())

	// 停止所有 EmuRunner 子进程
	sessionMgr.StopAll(context.Background())

	// 优雅关闭 gRPC Server（等待活跃请求完成）
	grpcSrv.GracefulStop()

	hb.Stop(context.Background(), rdb)
	slog.Info("worker stopped")
}
