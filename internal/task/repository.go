package task

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"timeweave/internal/models"
)

// Repository 封装 Task 表的数据库操作。
type Repository struct {
	db *gorm.DB
}

// NewRepository 构造仓库。
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Create 创建 Task。
func (r *Repository) Create(ctx context.Context, t *models.Task) error {
	return r.db.WithContext(ctx).Create(t).Error
}

// GetByID 按主键查询（含软删过滤）。
func (r *Repository) GetByID(ctx context.Context, id uint64) (*models.Task, error) {
	var t models.Task
	err := r.db.WithContext(ctx).First(&t, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}
	return &t, nil
}

// ListByMilestoneID 列出某 Milestone 下所有 Tasks（按 OrderIndex ASC, ID ASC 排序）。
func (r *Repository) ListByMilestoneID(ctx context.Context, milestoneID uint64, page, pageSize int) ([]models.Task, int64, error) {
	var (
		list  []models.Task
		total int64
	)
	tx := r.db.WithContext(ctx).Model(&models.Task{}).Where("milestone_id = ?", milestoneID)
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	if err := tx.
		Order("order_index ASC, id ASC").
		Limit(pageSize).Offset(offset).
		Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// ListByUserID 列当前用户的所有 Task，支持按 status 过滤（Phase 2 Dashboard 用）。
func (r *Repository) ListByUserID(ctx context.Context, userID uint64, status *int8, page, pageSize int) ([]models.Task, int64, error) {
	var (
		list  []models.Task
		total int64
	)
	tx := r.db.WithContext(ctx).Model(&models.Task{}).Where("user_id = ?", userID)
	if status != nil {
		tx = tx.Where("status = ?", *status)
	}
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	if err := tx.
		Order("due_date IS NULL, due_date ASC, priority DESC, id ASC").
		Limit(pageSize).Offset(offset).
		Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// ListActiveByUserID 列出当前用户所有未完成任务（status IN todo/doing/overdue）。
// 给 Scheduler 用：取出待排 Task 池。
//
// 排序：优先级降序 → 截止时间升序（NULL 排最后，MySQL 写法）→ ID 兜底稳定序。
// 注：MySQL 不支持 NULLS LAST，用 "due_date IS NULL"（false=0 排前）等价实现。
func (r *Repository) ListActiveByUserID(ctx context.Context, userID uint64) ([]models.Task, error) {
	var list []models.Task
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND status IN ?", userID, []int8{0, 1, 4}). // 0=todo 1=doing 4=overdue
		Order("priority DESC, due_date IS NULL, due_date ASC, id ASC").
		Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

// ListActiveByUserAndType 列出某用户某 task_type 下所有未完成任务。
// 给 timeprofile.Service 用：Learn 端回写 adjusted_estimate 时找出同类待办 Task。
func (r *Repository) ListActiveByUserAndType(ctx context.Context, userID uint64, taskType string) ([]models.Task, error) {
	var list []models.Task
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND task_type = ? AND status IN ?",
			userID, taskType, []int8{0, 1, 4}). // todo / doing / overdue
		Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

// Update 按主键更新字段。
func (r *Repository) Update(ctx context.Context, id uint64, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}
	result := r.db.WithContext(ctx).
		Model(&models.Task{}).
		Where("id = ?", id).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		// 校验记录是否存在
		var exists int64
		if err := r.db.WithContext(ctx).Model(&models.Task{}).Where("id = ?", id).Count(&exists).Error; err != nil {
			return err
		}
		if exists == 0 {
			return ErrTaskNotFound
		}
	}
	return nil
}

// SoftDelete 软删除。
func (r *Repository) SoftDelete(ctx context.Context, id uint64) error {
	result := r.db.WithContext(ctx).Delete(&models.Task{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrTaskNotFound
	}
	return nil
}
