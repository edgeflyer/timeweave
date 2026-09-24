package schedule

import (
	"context"
	"time"

	"timeweave/internal/models"
	"timeweave/internal/task"
)

// Service 编排层：排程 + 重排 + 查询的统一入口。
type Service struct {
	repo     *Repository
	taskRepo *task.Repository
}

// NewService 构造服务。
func NewService(repo *Repository, taskRepo *task.Repository) *Service {
	return &Service{repo: repo, taskRepo: taskRepo}
}

// Plan 给指定日期做排程。
//
// userID：用户 ID
// planDate：目标日期
// replaceExisting：是否清空已有时间块
//
// 返回：PlanResponse
func (s *Service) Plan(ctx context.Context, userID uint64, planDate time.Time, replaceExisting bool) (*PlanResponse, error) {
	now := time.Now()

	// 0. 可选：清空已有
	if replaceExisting {
		dayStart, dayEnd := dayBounds(planDate)
		if _, err := s.repo.DeleteByDateRange(ctx, userID, dayStart, dayEnd); err != nil {
			return nil, err
		}
	}

	// 1. 拉取 active Task
	tasks, err := s.taskRepo.ListActiveByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(tasks) == 0 {
		return nil, ErrScheduleNoTasksToPlan
	}

	// 2. 评分
	scored := scoreTasks(tasks, now)

	// 3. 计算可用 slot
	slots := defaultAvailabilitySlots(planDate, now)
	if len(slots) == 0 {
		return nil, ErrScheduleInsufficientTime
	}

	// 4. 取已有时间块（避免重复排）
	dayStart, dayEnd := dayBounds(planDate)
	existing, err := s.repo.ListByDateRange(ctx, userID, dayStart, dayEnd)
	if err != nil {
		return nil, err
	}

	// 5. 贪心填充
	blocks := buildPlan(scored, slots, existing, userID)
	if len(blocks) == 0 {
		return nil, ErrScheduleInsufficientTime
	}

	// 6. 写入
	if err := s.repo.CreateMany(ctx, blocks); err != nil {
		return nil, err
	}

	// 7. 响应
	skipped := len(tasks) - len(blocks)
	resp := &PlanResponse{
		Date:         planDate.Format("2006-01-02"),
		Blocks:       make([]*ScheduleBlockResponse, 0, len(blocks)),
		PlannedCount: len(blocks),
		SkippedCount: skipped,
	}
	for _, b := range blocks {
		resp.Blocks = append(resp.Blocks, toResponse(b))
	}
	return resp, nil
}

// Replan 重排指定日期。
func (s *Service) Replan(ctx context.Context, userID uint64, date time.Time) (*PlanResponse, error) {
	r := NewReplanner(s.repo, s.taskRepo)
	now := time.Now()
	blocks, err := r.Replan(ctx, userID, date, now)
	if err != nil {
		return nil, err
	}
	resp := &PlanResponse{
		Date:         date.Format("2006-01-02"),
		Blocks:       make([]*ScheduleBlockResponse, 0, len(blocks)),
		PlannedCount: len(blocks),
	}
	for _, b := range blocks {
		resp.Blocks = append(resp.Blocks, toResponse(b))
	}
	return resp, nil
}

// ListByDate 列出某日的时间块（按 start_at 排序）。
func (s *Service) ListByDate(ctx context.Context, userID uint64, date time.Time) (*ListBlocksResponse, error) {
	dayStart, dayEnd := dayBounds(date)
	list, err := s.repo.ListByDateRange(ctx, userID, dayStart, dayEnd)
	if err != nil {
		return nil, err
	}
	resp := &ListBlocksResponse{
		Date:   date.Format("2006-01-02"),
		Blocks: make([]*ScheduleBlockResponse, 0, len(list)),
	}
	for i := range list {
		resp.Blocks = append(resp.Blocks, toResponse(&list[i]))
	}
	return resp, nil
}

// Delete 删除单个时间块（带权限校验）。
func (s *Service) Delete(ctx context.Context, userID, blockID uint64) error {
	b, err := s.repo.GetByID(ctx, blockID)
	if err != nil {
		return err
	}
	if b.UserID != userID {
		return ErrScheduleBlockAccessDenied
	}
	return s.repo.SoftDelete(ctx, blockID)
}

// UpdateBlockStatus 更新单个时间块状态（带权限校验）。
// 给 timer 包用：start 时 active / complete 时 done / abandon 时 pending。
func (s *Service) UpdateBlockStatus(ctx context.Context, userID, blockID uint64, status models.ScheduleBlockStatus) error {
	return s.repo.UpdateStatus(ctx, blockID, userID, status)
}

// toResponse 模型转 DTO。
func toResponse(b *models.ScheduleBlock) *ScheduleBlockResponse {
	return &ScheduleBlockResponse{
		ID:            b.ID,
		UserID:        b.UserID,
		TaskID:        b.TaskID,
		GoalID:        b.GoalID,
		StartAt:       b.StartAt,
		EndAt:         b.EndAt,
		PriorityScore: b.PriorityScore,
		Reason:        b.Reason,
		Status:        int8(b.Status),
		CreatedAt:     b.CreatedAt,
	}
}
