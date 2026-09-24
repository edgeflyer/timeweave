package schedule

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"timeweave/internal/models"
)

// Repository 封装 ScheduleBlock 表的数据库操作。
type Repository struct {
	db *gorm.DB
}

// NewRepository 构造仓库。
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// CreateMany 批量创建时间块（事务）。
//
// 冲突检测：先查 [start_at, end_at) 内是否有同用户已存在的 active 时间块，
// 有则整个事务回滚（保证一致性）。
func (r *Repository) CreateMany(ctx context.Context, blocks []*models.ScheduleBlock) error {
	if len(blocks) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 收集所有时间区间，一次性查冲突
		userID := blocks[0].UserID
		var minStart, maxEnd time.Time
		for i, b := range blocks {
			if i == 0 || b.StartAt.Before(minStart) {
				minStart = b.StartAt
			}
			if i == 0 || b.EndAt.After(maxEnd) {
				maxEnd = b.EndAt
			}
		}

		var conflictCount int64
		if err := tx.Model(&models.ScheduleBlock{}).
			Where("user_id = ? AND status != ? AND start_at < ? AND end_at > ?",
				userID, models.ScheduleBlockStatusMissed, maxEnd, minStart).
			Count(&conflictCount).Error; err != nil {
			return err
		}
		if conflictCount > 0 {
			return ErrScheduleTimeConflict
		}

		// 批量插入
		if err := tx.Create(blocks).Error; err != nil {
			return err
		}
		return nil
	})
}

// GetByID 按主键查询。
func (r *Repository) GetByID(ctx context.Context, id uint64) (*models.ScheduleBlock, error) {
	var b models.ScheduleBlock
	err := r.db.WithContext(ctx).First(&b, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrScheduleBlockNotFound
		}
		return nil, err
	}
	return &b, nil
}

// ListByDateRange 列出某用户在指定时间区间内的时间块。
func (r *Repository) ListByDateRange(ctx context.Context, userID uint64, start, end time.Time) ([]models.ScheduleBlock, error) {
	var list []models.ScheduleBlock
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND start_at < ? AND end_at > ?", userID, end, start).
		Order("start_at ASC").
		Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

// DeleteByDateRange 删除某用户在指定日期的所有时间块。
// 用于重排前清空。
func (r *Repository) DeleteByDateRange(ctx context.Context, userID uint64, start, end time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("user_id = ? AND start_at >= ? AND start_at < ?", userID, start, end).
		Delete(&models.ScheduleBlock{})
	return result.RowsAffected, result.Error
}

// SoftDelete 软删除单个时间块。
func (r *Repository) SoftDelete(ctx context.Context, id uint64) error {
	result := r.db.WithContext(ctx).Delete(&models.ScheduleBlock{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrScheduleBlockNotFound
	}
	return nil
}

// UpdateStatus 更新单个时间块的状态字段（带归属校验）。
func (r *Repository) UpdateStatus(ctx context.Context, id, userID uint64, status models.ScheduleBlockStatus) error {
	result := r.db.WithContext(ctx).
		Model(&models.ScheduleBlock{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("status", status)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrScheduleBlockNotFound
	}
	return nil
}
