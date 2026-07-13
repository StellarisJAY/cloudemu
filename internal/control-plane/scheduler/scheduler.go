package scheduler

import (
	"context"
	"sort"

	"github.com/StellarisJAY/cloudemu/internal/control-plane/contract"
	"github.com/StellarisJAY/cloudemu/internal/pkg/apperror"
)

// Scheduler Worker 调度器实现，加权最低负载优先选择最优 Worker
// 评分公式: score = sessions / weight，选 score 最小且 sessions < max_sessions 的 Worker
type Scheduler struct{}

// New 创建 Scheduler 实例
func New() *Scheduler {
	return &Scheduler{}
}

// SelectWorker 从注册中心中选择最优 Worker
// 过滤条件：sessions < max_sessions
// 排序规则：score = sessions / weight 升序
func (s *Scheduler) SelectWorker(ctx context.Context, registry contract.WorkerRegistry) (*contract.WorkerInfo, error) {
	workers, err := registry.ListAlive(ctx)
	if err != nil {
		return nil, err
	}

	// 过滤：已满的 Worker 不参与调度
	candidates := make([]contract.WorkerInfo, 0, len(workers))
	for _, w := range workers {
		if w.Sessions < w.MaxSessions && w.Addr != "" {
			candidates = append(candidates, w)
		}
	}

	if len(candidates) == 0 {
		return nil, apperror.ErrNoAvailableWorker
	}

	// 排序：得分越低越优先（负载越低）
	sort.Slice(candidates, func(i, j int) bool {
		scoreI := float64(candidates[i].Sessions) / float64(candidates[i].Weight)
		scoreJ := float64(candidates[j].Sessions) / float64(candidates[j].Weight)
		return scoreI < scoreJ
	})

	return &candidates[0], nil
}
