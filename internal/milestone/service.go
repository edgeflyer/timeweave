package milestone

import (
	"context"
	"errors"
	"time"

	"timeweave/internal/goal"
	"timeweave/internal/models"
)

// Service Milestone 业务逻辑层。
type Service struct {
	repo     *Repository
	goalRepo *goal.Repository
}

// NewService 构造服务。
func NewService(repo *Repository, goalRepo *goal.Repository) *Service {
	return &Service{repo: repo, goalRepo: goalRepo}
}

// Create 在指定 Goal 下创建 Milestone。
// 校验 Goal 存在且属于当前用户。
func (s *Service) Create(ctx context.Context, userID, goalID uint64, req *CreateMilestoneRequest) (*MilestoneResponse, error) {
	goalRecord, err := s.goalRepo.GetByID(ctx, goalID)
	if err != nil {
		if errors.Is(err, goal.ErrGoalNotFound) {
			return nil, ErrMilestoneGoalNotFound
		}
		return nil, err
	}
	if goalRecord.UserID != userID {
		return nil, ErrMilestoneGoalAccessDenied
	}

	m := &models.Milestone{
		UserID:      userID,
		GoalID:      goalID,
		Title:       req.Title,
		Description: req.Description,
		Status:      models.MilestoneStatusActive,
		TargetDate:  req.TargetDate,
		OrderIndex:  req.OrderIndex,
	}
	if err := s.repo.Create(ctx, m); err != nil {
		return nil, err
	}
	return toResponse(m), nil
}

// Get 查询单个 Milestone（带权限校验）。
func (s *Service) Get(ctx context.Context, userID, milestoneID uint64) (*MilestoneResponse, error) {
	m, err := s.repo.GetByID(ctx, milestoneID)
	if err != nil {
		return nil, err
	}
	if m.UserID != userID {
		return nil, ErrMilestoneAccessDenied
	}
	return toResponse(m), nil
}

// List 列出某 Goal 下所有 Milestones。
func (s *Service) List(ctx context.Context, userID, goalID uint64, page, pageSize int) (*ListMilestonesResponse, error) {
	// 校验 Goal 归属
	goalRecord, err := s.goalRepo.GetByID(ctx, goalID)
	if err != nil {
		if errors.Is(err, goal.ErrGoalNotFound) {
			return nil, ErrMilestoneGoalNotFound
		}
		return nil, err
	}
	if goalRecord.UserID != userID {
		return nil, ErrMilestoneGoalAccessDenied
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	list, total, err := s.repo.ListByGoalID(ctx, goalID, page, pageSize)
	if err != nil {
		return nil, err
	}
	items := make([]*MilestoneResponse, 0, len(list))
	for i := range list {
		items = append(items, toResponse(&list[i]))
	}
	return &ListMilestonesResponse{
		List:     items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// Update 更新 Milestone（带权限校验）。
func (s *Service) Update(ctx context.Context, userID, milestoneID uint64, req *UpdateMilestoneRequest) (*MilestoneResponse, error) {
	m, err := s.repo.GetByID(ctx, milestoneID)
	if err != nil {
		return nil, err
	}
	if m.UserID != userID {
		return nil, ErrMilestoneAccessDenied
	}

	updates := map[string]interface{}{}
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.ClearTargetDate {
		updates["target_date"] = nil
	} else if req.TargetDate != nil {
		updates["target_date"] = *req.TargetDate
	}
	if req.OrderIndex != nil {
		updates["order_index"] = *req.OrderIndex
	}
	if len(updates) == 0 {
		return toResponse(m), nil
	}
	updates["updated_at"] = time.Now()

	if err := s.repo.Update(ctx, milestoneID, updates); err != nil {
		return nil, err
	}
	return s.Get(ctx, userID, milestoneID)
}

// Delete 软删除 Milestone（带权限校验）。
// Phase 1 不级联下属 Task，Task 由其所属 MilestoneID 字段继续指向已删除的 Milestone；
// Phase 2 再补级联逻辑。
func (s *Service) Delete(ctx context.Context, userID, milestoneID uint64) error {
	m, err := s.repo.GetByID(ctx, milestoneID)
	if err != nil {
		return err
	}
	if m.UserID != userID {
		return ErrMilestoneAccessDenied
	}
	return s.repo.SoftDelete(ctx, milestoneID)
}

// toResponse 模型转 DTO。
func toResponse(m *models.Milestone) *MilestoneResponse {
	return &MilestoneResponse{
		ID:          m.ID,
		GoalID:      m.GoalID,
		UserID:      m.UserID,
		Title:       m.Title,
		Description: m.Description,
		Status:      int8(m.Status),
		TargetDate:  m.TargetDate,
		OrderIndex:  m.OrderIndex,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}