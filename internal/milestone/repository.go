package milestone

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"timeweave/internal/models"
)

// Repository 封装 Milestone 表的数据库操作。
type Repository struct {
	db *gorm.DB
}

// NewRepository 构造仓库。
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Create 创建 Milestone。
func (r *Repository) Create(ctx context.Context, m *models.Milestone) error {
	return r.db.WithContext(ctx).Create(m).Error
}

// GetByID 按主键查询（含软删过滤）。
func (r *Repository) GetByID(ctx context.Context, id uint64) (*models.Milestone, error) {
	var m models.Milestone
	err := r.db.WithContext(ctx).First(&m, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMilestoneNotFound
		}
		return nil, err
	}
	return &m, nil
}

// ListByGoalID 列出某 Goal 下所有 Milestones（按 OrderIndex ASC, TargetDate ASC 排序）。
func (r *Repository) ListByGoalID(ctx context.Context, goalID uint64, page, pageSize int) ([]models.Milestone, int64, error) {
	var (
		list  []models.Milestone
		total int64
	)
	tx := r.db.WithContext(ctx).Model(&models.Milestone{}).Where("goal_id = ?", goalID)
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	if err := tx.
		Order("order_index ASC, target_date ASC, id ASC").
		Limit(pageSize).Offset(offset).
		Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// Update 按主键更新字段。
func (r *Repository) Update(ctx context.Context, id uint64, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}
	result := r.db.WithContext(ctx).
		Model(&models.Milestone{}).
		Where("id = ?", id).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		// 校验记录是否存在
		var exists int64
		if err := r.db.WithContext(ctx).Model(&models.Milestone{}).Where("id = ?", id).Count(&exists).Error; err != nil {
			return err
		}
		if exists == 0 {
			return ErrMilestoneNotFound
		}
	}
	return nil
}

// SoftDelete 软删除。
func (r *Repository) SoftDelete(ctx context.Context, id uint64) error {
	result := r.db.WithContext(ctx).Delete(&models.Milestone{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrMilestoneNotFound
	}
	return nil
}