package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	heartbeatInterval = 15 * time.Second // 心跳间隔
	heartbeatTTL      = 30 * time.Second // Redis key TTL，超时未续期即视为 Worker 宕机
	workerKeyPrefix   = "worker:"        // Redis key 前缀
)

// HeartbeatData Worker 心跳上报数据（与 contract.WorkerInfo 字段一致，在此独立定义避免循环依赖）
type HeartbeatData struct {
	ID          string    `json:"id"`
	Addr        string    `json:"addr"`
	Weight      int       `json:"weight"`
	Sessions    int       `json:"sessions"`
	MaxSessions int       `json:"max_sessions"`
	CPUPercent  float64   `json:"cpu_percent"`
	MemPercent  float64   `json:"mem_percent"`
	StartedAt   time.Time `json:"started_at"`
}

// Heartbeat Worker 心跳管理器，负责向 Redis 注册自身并定期续期
// 生命周期：
//
//	Start() → 首次注册 + 启动心跳循环
//	Stop()  → 关闭循环 + 删除 Redis key
//	UpdateSessions() → 更新当前会话计数（Worker 启动/停止 EmuRunner 时调用）
type Heartbeat struct {
	mu   sync.Mutex
	data HeartbeatData
	done chan struct{}
}

// Start 启动心跳：先执行一次注册写入 Redis，然后启动后台循环每 15s 续期
func (h *Heartbeat) Start(ctx context.Context, rdb *redis.Client, data HeartbeatData) error {
	h.mu.Lock()
	h.data = data
	h.done = make(chan struct{})
	h.mu.Unlock()

	if err := h.register(ctx, rdb); err != nil {
		return fmt.Errorf("worker register: %w", err)
	}

	go h.loop(ctx, rdb)
	return nil
}

// Stop 停止心跳循环并从 Redis 删除注册 key
func (h *Heartbeat) Stop(ctx context.Context, rdb *redis.Client) {
	h.mu.Lock()
	if h.done == nil {
		h.mu.Unlock()
		return
	}
	close(h.done)
	id := h.data.ID
	h.mu.Unlock()

	if err := rdb.Del(ctx, workerKey(id)).Err(); err != nil {
		// 日志由调用方处理，此处不引入 slog 依赖
		_ = err
	}
}

// UpdateSessions 更新当前会话计数（delta 为正表示新增会话，负表示会话结束）
// 仅在内存中更新，下一次心跳时同步到 Redis
func (h *Heartbeat) UpdateSessions(delta int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.data.Sessions += delta
}

// Sessions 返回当前会话计数
func (h *Heartbeat) Sessions() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.data.Sessions
}

// loop 后台心跳循环，每 15s 执行一次 SET key EX 30
func (h *Heartbeat) loop(ctx context.Context, rdb *redis.Client) {
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := h.register(ctx, rdb); err != nil {
				// 心跳失败不退出循环，下个周期重试
				// 若连续多次失败（累计超过 TTL 30s），Worker 自然从 Redis 消失
			}
		case <-h.done:
			return
		case <-ctx.Done():
			return
		}
	}
}

// register 执行一次 Redis SET，将当前 HeartbeatData 序列化为 JSON 写入
func (h *Heartbeat) register(ctx context.Context, rdb *redis.Client) error {
	h.mu.Lock()
	data, err := json.Marshal(h.data)
	h.mu.Unlock()
	if err != nil {
		return fmt.Errorf("marshal heartbeat: %w", err)
	}

	return rdb.Set(ctx, workerKey(h.data.ID), string(data), heartbeatTTL).Err()
}

func workerKey(id string) string {
	return workerKeyPrefix + id
}
