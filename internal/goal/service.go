package goal

import (
	"context"
	"time"

	"timeweave/internal/models"
)

// Service 业务逻辑层。
type Service struct {
	repo *Repository
}

// NewService 构造服务。
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// Create 创建目标。
func (s *Service) Create(ctx context.Context, userID uint64, req *CreateGoalRequest) (*GoalResponse, error) {
	g := &models.Goal{
		UserID:      userID,
		Title:       req.Title,
		Description: req.Description,
		Status:      models.GoalStatusActive,
		TargetDate:  req.TargetDate,
	}
	if err := s.repo.Create(ctx, g); err != nil {
		return nil, err
	}
	return toResponse(g), nil
}

// Get 查询单个目标（带权限校验）。
func (s *Service) Get(ctx context.Context, userID, goalID uint64) (*GoalResponse, error) {
	g, err := s.repo.GetByID(ctx, goalID)
	if err != nil {
		return nil, err
	}
	if g.UserID != userID {
		return nil, ErrGoalAccessDenied
	}
	return toResponse(g), nil
}

// List 列出用户的所有目标（分页）。
func (s *Service) List(ctx context.Context, userID uint64, page, pageSize int) (*ListGoalsResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	list, total, err := s.repo.ListByUserID(ctx, userID, page, pageSize)
	if err != nil {
		return nil, err
	}
	items := make([]*GoalResponse, 0, len(list))
	for i := range list {
		items = append(items, toResponse(&list[i]))
	}
	return &ListGoalsResponse{
		List:     items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// Update 更新目标（带权限校验）。
func (s *Service) Update(ctx context.Context, userID, goalID uint64, req *UpdateGoalRequest) (*GoalResponse, error) {
	// 1. 先校验权限
	g, err := s.repo.GetByID(ctx, goalID)
	if err != nil {
		return nil, err
	}
	if g.UserID != userID {
		return nil, ErrGoalAccessDenied
	}

	// 2. 构造 updates
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
	if req.Progress != nil {
		updates["progress"] = *req.Progress
	}
	if len(updates) == 0 {
		return toResponse(g), nil
	}
	updates["updated_at"] = time.Now()

	// 3. 写库
	if err := s.repo.Update(ctx, goalID, updates); err != nil {
		return nil, err
	}
	// 4. 返回最新值
	return s.Get(ctx, userID, goalID)
}

// Delete 软删除目标（带权限校验）。
func (s *Service) Delete(ctx context.Context, userID, goalID uint64) error {
	g, err := s.repo.GetByID(ctx, goalID)
	if err != nil {
		return err
	}
	if g.UserID != userID {
		return ErrGoalAccessDenied
	}
	return s.repo.SoftDelete(ctx, goalID)
}

// toResponse 模型转 DTO。
func toResponse(g *models.Goal) *GoalResponse {
	return &GoalResponse{
		ID:          g.ID,
		UserID:      g.UserID,
		Title:       g.Title,
		Description: g.Description,
		Status:      int8(g.Status),
		TargetDate:  g.TargetDate,
		Progress:    g.Progress,
		CreatedAt:   g.CreatedAt,
		UpdatedAt:   g.UpdatedAt,
	}
}