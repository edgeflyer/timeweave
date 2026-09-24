package execution

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"timeweave/internal/models"
)

// Repository 封装 task_executions 表的数据库操作。
type Repository struct {
	db *gorm.DB
}

// NewRepository 构造仓库。
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Create 写入一条 TaskExecution。
func (r *Repository) Create(ctx context.Context, e *models.TaskExecution) error {
	return r.db.WithContext(ctx).Create(e).Error
}

// GetByID 按主键查询。
func (r *Repository) GetByID(ctx context.Context, id uint64) (*models.TaskExecution, error) {
	var e models.TaskExecution
	err := r.db.WithContext(ctx).First(&e, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrExecutionNotFound
		}
		return nil, err
	}
	return &e, nil
}

// ListByUserID 列某用户的执行历史（按 id DESC），支持按 task_type 过滤。
func (r *Repository) ListByUserID(ctx context.Context, userID uint64, taskType string, page, pageSize int) ([]models.TaskExecution, int64, error) {
	var (
		list  []models.TaskExecution
		total int64
	)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	tx := r.db.WithContext(ctx).Model(&models.TaskExecution{}).Where("user_id = ?", userID)
	if taskType != "" {
		tx = tx.Where("task_type = ?", taskType)
	}
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	if err := tx.Order("id DESC").Limit(pageSize).Offset(offset).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// ListByTaskID 列某 Task 的全部执行历史（用于任务详情页）。
func (r *Repository) ListByTaskID(ctx context.Context, taskID uint64) ([]models.TaskExecution, error) {
	var list []models.TaskExecution
	err := r.db.WithContext(ctx).
		Where("task_id = ?", taskID).
		Order("id ASC").
		Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

// CountCompletedByUserAndType 统计某 user × task_type 已完成的执行次数。
// 给 timeprofile.Service 用：首次创建画像时若已存在完成样本，直接拿来算初始 rolling_avg
// （避免冷启动样本丢失，历史上完成过的同类任务也参与统计）。
func (r *Repository) CountCompletedByUserAndType(ctx context.Context, userID uint64, taskType string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.TaskExecution{}).
		Where("user_id = ? AND task_type = ? AND result = ?",
			userID, taskType, models.ExecutionResultCompleted).
		Count(&count).Error
	return count, err
}

// ListCompletedByUserAndType 拉取某 user × task_type 的全部已完成执行（按时间正序）。
// 给 timeprofile.Service 用：补建画像时计算历史 rolling_avg。
func (r *Repository) ListCompletedByUserAndType(ctx context.Context, userID uint64, taskType string) ([]models.TaskExecution, error) {
	var list []models.TaskExecution
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND task_type = ? AND result = ?",
			userID, taskType, models.ExecutionResultCompleted).
		Order("id ASC").
		Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}