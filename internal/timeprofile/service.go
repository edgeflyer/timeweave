package timeprofile

import (
	"context"
	"log/slog"
	"time"

	"timeweave/internal/models"
	"timeweave/internal/task"
)

// 关键参数 —— 改这里会影响 Learn 行为。集中定义便于调参。
const (
	// coldStartThreshold 冷启动样本阈值。低于此值时 adjusted 用加权混合（防异常值带偏）。
	coldStartThreshold = 10
	// coldStartWeightInitial 冷启动加权时 initial_estimate 的权重（0.7 = 70%）。
	coldStartWeightInitial = 0.7
	// coldStartWeightRolling 冷启动加权时 rolling_avg 的权重（0.3 = 30%）。
	coldStartWeightRolling = 0.3
	// propagationThreshold 回写到同类未完成 Task 的最小偏差比例（10%）。
	// 低于此值不写，避免抖动。
	propagationThreshold = 0.10
)

// TaskUpdater 抽象了 timeprofile 对 task 表的写权限（仅 estimated_minutes）。
// 由 main.go 注入 *task.Repository 实现。
//
// 单独抽象一个接口的原因：让 timeprofile 只依赖 task 的最小能力，
// 将来重构 task 不会牵连 timeprofile。
type TaskUpdater interface {
	ListActiveByUserAndType(ctx context.Context, userID uint64, taskType string) ([]models.Task, error)
	Update(ctx context.Context, id uint64, updates map[string]interface{}) error
}

// Service Learn 层业务逻辑：
//   - UpdateProfile：每次 TaskExecution 完成时由 execution.Service 回调
//   - Get / List / Reset：HTTP 暴露给用户查看 + 重置
//   - AdjustByAgent：内部方法，Phase 5 Review Agent 调用
type Service struct {
	repo  *Repository
	tasks TaskUpdater
}

// NewService 构造服务。
func NewService(repo *Repository, tasks TaskUpdater) *Service {
	return &Service{
		repo:  repo,
		tasks: tasks,
	}
}

// UpdateProfile Learn 端的核心算法入口。
//
// 入参：
//   - userID：当前用户
//   - taskType：从 Task 拿到的类型（已通过 IsValidTaskType 校验）
//   - actualMinutes：本轮 Timer 实际花费的分钟数
//   - initialEstimateMinutes：本次执行开始时 Task.estimated_minutes 的快照
//     （用作新画像的 baseline；后续不再变化，作为算法的「原点」）
//
// 算法步骤：
//  1. 读已有画像（不存在则当作空）
//  2. 计算新 rolling_avg（增量更新公式，无需重算历史）
//  3. 计算新 adjusted_estimate：
//     - 冷启动（count < threshold）：adjusted = w1*initial + w2*rolling
//     - 热态（count >= threshold）：adjusted = rolling（纯均值）
//  4. Upsert 画像
//  5. 若新 adjusted 与旧 adjusted 偏差 > 10%，把所有同类未完成 Task 的 estimated_minutes 同步过去
//
// 返回更新后的画像（方便调用方 / 调试打印）。
func (s *Service) UpdateProfile(
	ctx context.Context,
	userID uint64,
	taskType models.TaskType,
	actualMinutes int,
	initialEstimateMinutes int,
) (*models.UserTimeProfile, error) {
	if !models.IsValidTaskType(taskType) {
		return nil, ErrTimeProfileInvalidType
	}
	if actualMinutes <= 0 {
		// 没有有效样本，不更新（防御性，正常 Keep 时不用）
		return nil, nil
	}

	// 1. 读已有画像（找不到视为全新画像）
	old, err := s.repo.GetByUserAndType(ctx, userID, taskType)
	now := time.Now()
	if err != nil && err != ErrTimeProfileNotFound {
		return nil, err
	}

	var (
		newCount int
		newAvg   int
		initial  int
	)
	if err == ErrTimeProfileNotFound {
		// 新建画像 —— initial 来自首次执行的 Task 快照
		newCount = 1
		newAvg = actualMinutes
		initial = initialEstimateMinutes
	} else {
		newCount = old.SampleCount + 1
		// 增量滚动平均：(old_avg × old_count + actual) / new_count
		newAvg = (old.RollingAvgMinutes*old.SampleCount + actualMinutes) / newCount
		initial = old.InitialEstimateMinutes
	}

	// 3. 计算新 adjusted
	var newAdjusted int
	if newCount < coldStartThreshold {
		// 冷启动：加权混合（initial 占 70%，rolling 占 30%）
		// 注意：先转 float 再取整，避免 int 乘法溢出（虽然这里实际不会）
		newAdjusted = int(float64(initial)*coldStartWeightInitial + float64(newAvg)*coldStartWeightRolling + 0.5)
	} else {
		// 热态：纯滚动均值
		newAdjusted = newAvg
	}
	if newAdjusted < 1 {
		newAdjusted = 1 // 防御：至少 1 分钟
	}

	// 4. Upsert 画像
	profile := &models.UserTimeProfile{
		UserID:                  userID,
		TaskType:                taskType,
		InitialEstimateMinutes:  initial,
		RollingAvgMinutes:       newAvg,
		AdjustedEstimateMinutes: newAdjusted,
		SampleCount:             newCount,
		LastUpdatedAt:           now,
	}
	// 已存在时保留 manual_adjustment_reason（不让算法覆盖 Agent 设置的理由）
	if err == nil && old != nil {
		profile.ManualAdjustmentReason = old.ManualAdjustmentReason
	}
	if err := s.repo.Upsert(ctx, profile); err != nil {
		return nil, err
	}

	// 5. 回写到同类未完成 Task（仅当偏差 > 阈值）
	oldAdjusted := 0
	if err == nil && old != nil {
		oldAdjusted = old.AdjustedEstimateMinutes
	}
	if shouldPropagate(oldAdjusted, newAdjusted) {
		if err := s.propagateToActiveTasks(ctx, userID, taskType, newAdjusted); err != nil {
			// 回写失败不影响 Learn 主流程（画像已经更新了）
			slog.Warn("timeprofile: propagate to active tasks failed",
				"user_id", userID, "task_type", taskType, "err", err)
		}
	}

	return profile, nil
}

// shouldPropagate 判定偏差是否超过阈值需要回写。
//
// 边界：
//   - 首次创建画像（oldAdjusted == 0）：newAdjusted 一定 > 0，直接视为大幅偏差 → 回写
//   - 旧值 > 0：相对偏差 = |new - old| / old，超过阈值才回写
func shouldPropagate(oldAdjusted, newAdjusted int) bool {
	if oldAdjusted == 0 {
		return newAdjusted > 0
	}
	diff := newAdjusted - oldAdjusted
	if diff < 0 {
		diff = -diff
	}
	return float64(diff)/float64(oldAdjusted) > propagationThreshold
}

// propagateToActiveTasks 把 adjusted 回写到某 user × task_type 下所有未完成 Task。
func (s *Service) propagateToActiveTasks(ctx context.Context, userID uint64, taskType models.TaskType, newAdjusted int) error {
	list, err := s.tasks.ListActiveByUserAndType(ctx, userID, string(taskType))
	if err != nil {
		return err
	}
	for i := range list {
		if err := s.tasks.Update(ctx, list[i].ID, map[string]interface{}{
			"estimated_minutes": newAdjusted,
			"updated_at":        time.Now(),
		}); err != nil {
			slog.Warn("timeprofile: update task estimated_minutes failed",
				"task_id", list[i].ID, "err", err)
		}
	}
	return nil
}

// Get 查询某 task_type 的画像。
func (s *Service) Get(ctx context.Context, userID uint64, taskType models.TaskType) (*UserTimeProfileResponse, error) {
	if !models.IsValidTaskType(taskType) {
		return nil, ErrTimeProfileInvalidType
	}
	p, err := s.repo.GetByUserAndType(ctx, userID, taskType)
	if err != nil {
		return nil, err
	}
	return toResponse(p), nil
}

// List 列当前用户的所有画像（按 task_type ASC）。
func (s *Service) List(ctx context.Context, userID uint64) (*ListProfilesResponse, error) {
	list, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	items := make([]*UserTimeProfileResponse, 0, len(list))
	for i := range list {
		items = append(items, toResponse(&list[i]))
	}
	return &ListProfilesResponse{List: items, Total: len(items)}, nil
}

// Reset 删除指定 task_type 的画像（让系统"重新教我"）。
// 注意：删除后下次执行完成时会按全新画像重建。
func (s *Service) Reset(ctx context.Context, userID uint64, taskType models.TaskType) error {
	if !models.IsValidTaskType(taskType) {
		return ErrTimeProfileInvalidType
	}
	return s.repo.Delete(ctx, userID, taskType)
}

// AdjustByAgent Review Agent 强制覆盖 adjusted_estimate + 写入归因。
// 本期不暴露 HTTP，仅由 service 层直接调用（Phase 5 Agent 工具层接入）。
func (s *Service) AdjustByAgent(ctx context.Context, userID uint64, req *AdjustByAgentRequest) (*UserTimeProfileResponse, error) {
	taskType := models.TaskType(req.TaskType)
	if !models.IsValidTaskType(taskType) {
		return nil, ErrTimeProfileInvalidType
	}
	if req.NewValue <= 0 {
		return nil, ErrTimeProfileInvalidAgentValue
	}
	if err := s.repo.SetManualAdjustment(ctx, userID, taskType, req.NewValue, req.Reason); err != nil {
		return nil, err
	}
	// Agent 调整后也回写到同类未完成 Task（Agent 的判断通常有充分理由）
	if err := s.propagateToActiveTasks(ctx, userID, taskType, req.NewValue); err != nil {
		slog.Warn("timeprofile: agent adjustment propagate failed",
			"user_id", userID, "task_type", taskType, "err", err)
	}
	p, err := s.repo.GetByUserAndType(ctx, userID, taskType)
	if err != nil {
		return nil, err
	}
	return toResponse(p), nil
}

// toResponse 模型转 DTO。
func toResponse(p *models.UserTimeProfile) *UserTimeProfileResponse {
	return &UserTimeProfileResponse{
		ID:                      p.ID,
		UserID:                  p.UserID,
		TaskType:                string(p.TaskType),
		InitialEstimateMinutes:  p.InitialEstimateMinutes,
		RollingAvgMinutes:       p.RollingAvgMinutes,
		AdjustedEstimateMinutes: p.AdjustedEstimateMinutes,
		SampleCount:             p.SampleCount,
		ManualAdjustmentReason:  p.ManualAdjustmentReason,
		LastUpdatedAt:           p.LastUpdatedAt,
		CreatedAt:               p.CreatedAt,
	}
}

// 编译期检查：*task.Repository 必须实现 TaskUpdater 接口。
// 若接口不兼容，下面这行编译失败 —— 比运行时报错更友好。
var _ TaskUpdater = (*task.Repository)(nil)