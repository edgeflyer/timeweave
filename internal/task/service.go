package task

import (
	"context"
	"errors"
	"time"

	"timeweave/internal/milestone"
	"timeweave/internal/models"
)

// Service Task 业务逻辑层。
type Service struct {
	repo        *Repository
	milestoneRepo *milestone.Repository
}

// NewService 构造服务。
func NewService(repo *Repository, milestoneRepo *milestone.Repository) *Service {
	return &Service{repo: repo, milestoneRepo: milestoneRepo}
}

// Create 在指定 Milestone 下创建 Task。
// 校验 Milestone 存在且属于当前用户。
func (s *Service) Create(ctx context.Context, userID, milestoneID uint64, req *CreateTaskRequest) (*TaskResponse, error) {
	m, err := s.milestoneRepo.GetByID(ctx, milestoneID)
	if err != nil {
		if errors.Is(err, milestone.ErrMilestoneNotFound) {
			return nil, ErrTaskMilestoneNotFound
		}
		return nil, err
	}
	if m.UserID != userID {
		return nil, ErrTaskMilestoneAccessDenied
	}

	priority := models.TaskPriorityNormal
	if req.Priority != nil {
		priority = models.TaskPriority(*req.Priority)
	}

	taskType := models.TaskTypeOther
	if req.TaskType != "" {
		taskType = models.TaskType(req.TaskType)
	}

	t := &models.Task{
		UserID:           userID,
		MilestoneID:      milestoneID,
		GoalID:           m.GoalID, // 冗余存储，方便按用户聚合
		Title:            req.Title,
		Description:      req.Description,
		Status:           models.TaskStatusTodo,
		Priority:         priority,
		TaskType:         taskType, // Sprint 5: Learn 端的分桶维度
		EstimatedMinutes: req.EstimatedMinutes,
		DueDate:          req.DueDate,
		OrderIndex:       req.OrderIndex,
	}
	if err := s.repo.Create(ctx, t); err != nil {
		return nil, err
	}
	return toResponse(t), nil
}

// Get 查询单个 Task（带权限校验）。
func (s *Service) Get(ctx context.Context, userID, taskID uint64) (*TaskResponse, error) {
	t, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if t.UserID != userID {
		return nil, ErrTaskAccessDenied
	}
	return toResponse(t), nil
}

// List 列出某 Milestone 下所有 Tasks。
func (s *Service) List(ctx context.Context, userID, milestoneID uint64, page, pageSize int) (*ListTasksResponse, error) {
	m, err := s.milestoneRepo.GetByID(ctx, milestoneID)
	if err != nil {
		if errors.Is(err, milestone.ErrMilestoneNotFound) {
			return nil, ErrTaskMilestoneNotFound
		}
		return nil, err
	}
	if m.UserID != userID {
		return nil, ErrTaskMilestoneAccessDenied
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	list, total, err := s.repo.ListByMilestoneID(ctx, milestoneID, page, pageSize)
	if err != nil {
		return nil, err
	}
	items := make([]*TaskResponse, 0, len(list))
	for i := range list {
		items = append(items, toResponse(&list[i]))
	}
	return &ListTasksResponse{
		List:     items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// ListByUser 列当前用户所有 Task，支持按 status 过滤。
func (s *Service) ListByUser(ctx context.Context, userID uint64, status *int8, page, pageSize int) (*ListTasksResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	list, total, err := s.repo.ListByUserID(ctx, userID, status, page, pageSize)
	if err != nil {
		return nil, err
	}
	items := make([]*TaskResponse, 0, len(list))
	for i := range list {
		items = append(items, toResponse(&list[i]))
	}
	return &ListTasksResponse{
		List:     items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// Update 更新 Task（带权限校验，不允许改 status）。
func (s *Service) Update(ctx context.Context, userID, taskID uint64, req *UpdateTaskRequest) (*TaskResponse, error) {
	t, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if t.UserID != userID {
		return nil, ErrTaskAccessDenied
	}

	updates := map[string]interface{}{}
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Priority != nil {
		updates["priority"] = *req.Priority
	}
	if req.TaskType != nil {
		updates["task_type"] = *req.TaskType
	}
	if req.EstimatedMinutes != nil {
		updates["estimated_minutes"] = *req.EstimatedMinutes
	}
	if req.ActualMinutes != nil {
		updates["actual_minutes"] = *req.ActualMinutes
	}
	if req.ClearDueDate {
		updates["due_date"] = nil
	} else if req.DueDate != nil {
		updates["due_date"] = *req.DueDate
	}
	if req.OrderIndex != nil {
		updates["order_index"] = *req.OrderIndex
	}
	if len(updates) == 0 {
		return toResponse(t), nil
	}
	updates["updated_at"] = time.Now()

	if err := s.repo.Update(ctx, taskID, updates); err != nil {
		return nil, err
	}
	return s.Get(ctx, userID, taskID)
}

// UpdateStatus 切换任务状态（走状态机校验）。
func (s *Service) UpdateStatus(ctx context.Context, userID, taskID uint64, newStatus models.TaskStatus) (*TaskResponse, error) {
	t, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if t.UserID != userID {
		return nil, ErrTaskAccessDenied
	}

	// 已 done 的任务不允许再切换（Phase 1 锁死，Phase 2 可放开"重做"）
	if t.Status == models.TaskStatusDone && newStatus != models.TaskStatusDone {
		return nil, ErrTaskAlreadyDone
	}

	// 状态机校验
	if !isValidTransition(t.Status, newStatus) {
		return nil, ErrTaskInvalidStatusTransition
	}

	updates := map[string]interface{}{
		"status":     newStatus,
		"updated_at": time.Now(),
	}

	now := time.Now()
	switch newStatus {
	case models.TaskStatusDoing:
		// 第一次进入 doing 时记录 StartedAt（如果还没记过）
		if t.StartedAt == nil {
			updates["started_at"] = now
		}
	case models.TaskStatusDone:
		updates["completed_at"] = now
		// 如果有 started_at，累计 actual_minutes
		if t.StartedAt != nil {
			elapsed := int(now.Sub(*t.StartedAt).Minutes())
			if elapsed > 0 {
				updates["actual_minutes"] = t.ActualMinutes + elapsed
			}
		}
	}

	if err := s.repo.Update(ctx, taskID, updates); err != nil {
		return nil, err
	}
	return s.Get(ctx, userID, taskID)
}

// Delete 软删除 Task（带权限校验）。
func (s *Service) Delete(ctx context.Context, userID, taskID uint64) error {
	t, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return err
	}
	if t.UserID != userID {
		return ErrTaskAccessDenied
	}
	return s.repo.SoftDelete(ctx, taskID)
}

// isValidTransition 状态机合法跃迁校验。
func isValidTransition(from, to models.TaskStatus) bool {
	if from == to {
		return true // 幂等
	}
	switch from {
	case models.TaskStatusTodo:
		return to == models.TaskStatusDoing || to == models.TaskStatusSkipped
	case models.TaskStatusDoing:
		return to == models.TaskStatusTodo || to == models.TaskStatusDone || to == models.TaskStatusSkipped
	case models.TaskStatusSkipped:
		return to == models.TaskStatusTodo
	case models.TaskStatusOverdue:
		return to == models.TaskStatusTodo || to == models.TaskStatusDoing || to == models.TaskStatusSkipped
	case models.TaskStatusDone:
		return false // 终态
	default:
		return false
	}
}

// toResponse 模型转 DTO。
func toResponse(t *models.Task) *TaskResponse {
	return &TaskResponse{
		ID:               t.ID,
		UserID:           t.UserID,
		MilestoneID:      t.MilestoneID,
		GoalID:           t.GoalID,
		Title:            t.Title,
		Description:      t.Description,
		Status:           int8(t.Status),
		Priority:         int8(t.Priority),
		TaskType:         string(t.TaskType), // Sprint 5
		EstimatedMinutes: t.EstimatedMinutes,
		ActualMinutes:    t.ActualMinutes,
		DueDate:          t.DueDate,
		StartedAt:        t.StartedAt,
		CompletedAt:      t.CompletedAt,
		OrderIndex:       t.OrderIndex,
		CreatedAt:        t.CreatedAt,
		UpdatedAt:        t.UpdatedAt,
	}
}
