package goal

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"timeweave/internal/models"
)

// Repository 封装 Goal 表的数据库操作。
type Repository struct {
	db *gorm.DB
}

// NewRepository 构造仓库。
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Create 创建目标。
func (r *Repository) Create(ctx context.Context, g *models.Goal) error {
	return r.db.WithContext(ctx).Create(g).Error
}

// GetByID 按主键查询（含软删过滤）。
func (r *Repository) GetByID(ctx context.Context, id uint64) (*models.Goal, error) {
	var g models.Goal
	err := r.db.WithContext(ctx).First(&g, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrGoalNotFound
		}
		return nil, err
	}
	return &g, nil
}

// ListByUserID 分页查询用户的所有目标。
func (r *Repository) ListByUserID(ctx context.Context, userID uint64, page, pageSize int) ([]models.Goal, int64, error) {
	var (
		list  []models.Goal
		total int64
	)
	tx := r.db.WithContext(ctx).Model(&models.Goal{}).Where("user_id = ?", userID)
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	if err := tx.Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// Update 按主键更新字段。
// updates 是 map[string]interface{}，使用 GORM 的 Updates 方法（只更新非零字段由 map 控制）。
func (r *Repository) Update(ctx context.Context, id uint64, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}
	result := r.db.WithContext(ctx).
		Model(&models.Goal{}).
		Where("id = ?", id).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		// 0 行受影响：可能是 id 不存在，或字段值没变化。
		// 为存在性不检查再查一次
		var exists int64
		if err := r.db.WithContext(ctx).Model(&models.Goal{}).Where("id = ?", id).Count(&exists).Error; err != nil {
			return err
		}
		if exists == 0 {
			return ErrGoalNotFound
		}
	}
	return nil
}

// SoftDelete 软删除目标。
func (r *Repository) SoftDelete(ctx context.Context, id uint64) error {
	result := r.db.WithContext(ctx).Delete(&models.Goal{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrGoalNotFound
	}
	return nil
}