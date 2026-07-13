package cache

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/StellarisJAY/cloudemu/internal/control-plane/contract"
	"github.com/redis/go-redis/v9"
)

const workerKeyPrefix = "worker:"

// WorkerRegistry Worker 注册中心，基于 Redis DB 1 实现 contract.WorkerRegistry 接口
// WorkerAgent 通过 SET worker:{id} {json} EX 30 注册心跳
// Control Plane 通过 SCAN + GET 发现所有存活 Worker
type WorkerRegistry struct {
	cli *redis.Client
}

// NewWorkerRegistry 创建 WorkerRegistry 实例
// cli 应当连接到 Redis DB 1（Worker 调度专用），由调用方保证
func NewWorkerRegistry(cli *redis.Client) *WorkerRegistry {
	return &WorkerRegistry{cli: cli}
}

// ListAlive 返回所有存活的 Worker 列表
// 使用 SCAN 遍历所有 worker:* key，逐条 GET 并反序列化 JSON
// 跳过已过期（GET 返回 nil）或解析失败的数据
func (r *WorkerRegistry) ListAlive(ctx context.Context) ([]contract.WorkerInfo, error) {
	var workers []contract.WorkerInfo
	var cursor uint64

	for {
		keys, nextCursor, err := r.cli.Scan(ctx, cursor, workerKeyPrefix+"*", 100).Result()
		if err != nil {
			return nil, fmt.Errorf("scan workers: %w", err)
		}

		for _, key := range keys {
			val, err := r.cli.Get(ctx, key).Result()
			if err != nil {
				continue // key 可能已过期，跳过
			}

			var info contract.WorkerInfo
			if err := json.Unmarshal([]byte(val), &info); err != nil {
				continue // 数据格式异常，跳过
			}
			workers = append(workers, info)
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return workers, nil
}
