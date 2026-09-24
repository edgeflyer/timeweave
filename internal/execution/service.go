package execution

import (
	"context"
	"errors"
	"log/slog"

	"timeweave/internal/models"
	"timeweave/internal/task"
)

// TimeProfileUpdater 是 Learn 端（timeprofile.Service）的接口。
// execution.Service.RecordExecution 完成落库后调用此接口触发画像更新。
//
// 用接口注入的目的：让 execution 不直接依赖 timeprofile 包，保持
// execution → (interface) ← timeprofile 的单向边界，将来 timeprofile
// 重构不影响 execution。
type TimeProfileUpdater interface {
	UpdateProfile(ctx context.Context, userID uint64, taskType models.TaskType, actualMinutes, initialEstimateMinutes int) (*models.UserTimeProfile, error)
}

// TaskAccessor 是读取 Task 信息的接口（execution 只需要 task_type / estimated_minutes / user_id）。
// main.go 注入 *task.Repository 实现。
type TaskAccessor interface {
	GetByID(ctx context.Context, id uint64) (*models.Task, error)
}

// Service Observe 层业务逻辑：Timer 终态时落库 TaskExecution 并触发 Learn。
type Service struct {
	repo    *Repository
	task    TaskAccessor
	profile TimeProfileUpdater
}

// NewService 构造服务。
//
// 依赖注入说明：
//   - repo：task_executions 表的仓库
//   - task：用于读 Task 的 task_type / estimated_minutes（落库时快照）
//   - profile：Learn 端接口，写完执行后触发画像更新
func NewService(repo *Repository, task TaskAccessor, profile TimeProfileUpdater) *Service {
	return &Service{
		repo:    repo,
		task:    task,
		profile: profile,
	}
}

// RecordExecution 记录一次 Task 执行，触发 Learn 画像更新。
//
// 流程：
//  1. 读 Task（拿 task_type 和 estimated_minutes 快照）
//  2. 计算 planned / actual / overtime
//  3. 写 task_executions
//  4. 若 Result=Completed 且 actualMinutes > 0，调用 profile.UpdateProfile()
//     （abandoned 的执行不计入学习 —— 中途放弃不代表真实耗时节奏）
//
// 注意：actualMinutes = (EndAt - StartAt - PauseSeconds) / 60，向下取整。
// 分钟粒度足够 Learn 使用，更细粒度反而引入抖动。
func (s *Service) RecordExecution(ctx context.Context, req *RecordExecutionRequest) (*TaskExecutionResponse, error) {
	// 1. 读 Task
	t, err := s.task.GetByID(ctx, req.TaskID)
	if err != nil {
		if errors.Is(err, task.ErrTaskNotFound) {
			return nil, ErrExecutionTaskNotFound
		}
		if errors.Is(err, task.ErrTaskAccessDenied) {
			return nil, ErrExecutionTaskAccessDenied
		}
		return nil, err
	}
	if t.UserID != req.UserID {
		return nil, ErrExecutionTaskAccessDenied
	}

	// 2. 计算时长（分钟）
	elapsedSec := int(req.EndAt.Sub(req.StartAt).Seconds()) - req.PauseSeconds
	if elapsedSec < 0 {
		elapsedSec = 0
	}
	actualMinutes := elapsedSec / 60
	plannedMinutes := t.EstimatedMinutes
	overtimeSeconds := elapsedSec - plannedMinutes*60

	// 3. 落库
	exec := &models.TaskExecution{
		UserID:          req.UserID,
		TaskID:          req.TaskID,
		TimerSessionID:  req.TimerSessionID,
		TaskType:        t.TaskType,
		StartAt:         req.StartAt,
		EndAt:           req.EndAt,
		PlannedMinutes:  plannedMinutes,
		ActualMinutes:   actualMinutes,
		PauseSeconds:    req.PauseSeconds,
		OvertimeSeconds: overtimeSeconds,
		Result:          models.ExecutionResult(req.Result),
	}
	if err := s.repo.Create(ctx, exec); err != nil {
		return nil, err
	}

	// 4. 触发 Learn（仅 Completed 计入学习样本）
	if exec.Result == models.ExecutionResultCompleted && actualMinutes > 0 && s.profile != nil {
		if _, err := s.profile.UpdateProfile(ctx, req.UserID, t.TaskType, actualMinutes, plannedMinutes); err != nil {
			// Learn 失败不影响主流程，记 warn 即可（画像是优化项，不是核心数据）
			slog.Warn("execution: trigger update profile failed",
				"user_id", req.UserID, "task_type", t.TaskType, "err", err)
		}
	}

	return toResponse(exec), nil
}

// GetByID 查询单条执行记录（内部/debug 用，后续 dashboard 暴露 HTTP）。
func (s *Service) GetByID(ctx context.Context, id uint64) (*TaskExecutionResponse, error) {
	e, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toResponse(e), nil
}

// ListByUser 列某用户的执行历史（按 id DESC），支持按 task_type 过滤。
func (s *Service) ListByUser(ctx context.Context, userID uint64, taskType string, page, pageSize int) (*ListExecutionsResponse, error) {
	list, total, err := s.repo.ListByUserID(ctx, userID, taskType, page, pageSize)
	if err != nil {
		return nil, err
	}
	items := make([]*TaskExecutionResponse, 0, len(list))
	for i := range list {
		items = append(items, toResponse(&list[i]))
	}
	return &ListExecutionsResponse{
		List:     items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// toResponse 模型转 DTO。
func toResponse(e *models.TaskExecution) *TaskExecutionResponse {
	return &TaskExecutionResponse{
		ID:              e.ID,
		UserID:          e.UserID,
		TaskID:          e.TaskID,
		TimerSessionID:  e.TimerSessionID,
		TaskType:        string(e.TaskType),
		StartAt:         e.StartAt,
		EndAt:           e.EndAt,
		PlannedMinutes:  e.PlannedMinutes,
		ActualMinutes:   e.ActualMinutes,
		PauseSeconds:    e.PauseSeconds,
		OvertimeSeconds: e.OvertimeSeconds,
		Result:          int8(e.Result),
		CreatedAt:       e.CreatedAt,
	}
}