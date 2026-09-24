package schedule

import (
	"context"
	"time"

	"timeweave/internal/models"
)

// TaskAccessor 是 task.Repository 的最小接口，避免 schedule → task 循环依赖。
type TaskAccessor interface {
	ListActiveByUserID(ctx context.Context, userID uint64) ([]models.Task, error)
}

// Replanner 重排指定日期的时间表。
//
// 触发场景：
//   - Task 完成：释放时间块，把后续待排 Task 补进来
//   - Task 超时：把后续时间顺延
//   - 手动触发：用户点"重排今日"
//
// 算法：
//   1. 删除该日所有未执行的时间块（status=pending/skipped/missed）
//   2. 重新拉取 active Task 列表
//   3. 重新评分 + 贪心填充
type Replanner struct {
	repo     *Repository
	taskRepo TaskAccessor
}

// NewReplanner 构造重排器。
func NewReplanner(repo *Repository, taskRepo TaskAccessor) *Replanner {
	return &Replanner{repo: repo, taskRepo: taskRepo}
}

// Replan 执行重排，返回新生成的时间块列表。
func (r *Replanner) Replan(ctx context.Context, userID uint64, date time.Time, now time.Time) ([]*models.ScheduleBlock, error) {
	// 1. 删除该日所有时间块（避免保留旧的 pending/skipped）
	dayStart, dayEnd := dayBounds(date)
	if _, err := r.repo.DeleteByDateRange(ctx, userID, dayStart, dayEnd); err != nil {
		return nil, err
	}

	// 2. 重新拉取 active Task
	tasks, err := r.taskRepo.ListActiveByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(tasks) == 0 {
		return nil, ErrScheduleNoTasksToPlan
	}

	// 3. 评分 + 贪心填充
	scored := scoreTasks(tasks, now)
	slots := defaultAvailabilitySlots(date, now)
	if len(slots) == 0 {
		return nil, ErrScheduleInsufficientTime
	}

	blocks := buildPlan(scored, slots, nil, userID)
	if len(blocks) == 0 {
		return nil, ErrScheduleInsufficientTime
	}

	if err := r.repo.CreateMany(ctx, blocks); err != nil {
		return nil, err
	}
	return blocks, nil
}

// dayBounds 返回某日的 [start, end) 区间。
func dayBounds(date time.Time) (time.Time, time.Time) {
	y, m, d := date.Date()
	start := time.Date(y, m, d, 0, 0, 0, 0, date.Location())
	end := start.Add(24 * time.Hour)
	return start, end
}
