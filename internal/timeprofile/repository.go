package timeprofile

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"timeweave/internal/models"
)

// Repository 封装 user_time_profiles 表的数据库操作。
type Repository struct {
	db *gorm.DB
}

// NewRepository 构造仓库。
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// GetByUserAndType 取某 user × task_type 的画像，找不到返回 ErrTimeProfileNotFound。
func (r *Repository) GetByUserAndType(ctx context.Context, userID uint64, taskType models.TaskType) (*models.UserTimeProfile, error) {
	var p models.UserTimeProfile
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND task_type = ?", userID, taskType).
		First(&p).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTimeProfileNotFound
		}
		return nil, err
	}
	return &p, nil
}

// ListByUser 列某用户所有画像（按 task_type ASC）。
func (r *Repository) ListByUser(ctx context.Context, userID uint64) ([]models.UserTimeProfile, error) {
	var list []models.UserTimeProfile
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("task_type ASC").
		Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

// Upsert 创建或更新画像（按 user_id × task_type 唯一键）。
// 内部用 ON DUPLICATE KEY UPDATE 语义（GORM 的 clause.OnConflict）。
//
// 用 Upsert 的原因：
//   - 首次某 user × type 有完成样本时，需要创建画像（initial 来自 Task.estimated_minutes）
//   - 后续每次执行完成时，按 rolling 更新
//   - Upsert 把「不存在则插入、存在则更新」合并成一次原子操作，避免并发竞态
func (r *Repository) Upsert(ctx context.Context, p *models.UserTimeProfile) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}, {Name: "task_type"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"initial_estimate_minutes",
				"rolling_avg_minutes",
				"adjusted_estimate_minutes",
				"sample_count",
				"last_updated_at",
				"updated_at",
			}),
		}).
		Create(p).Error
}

// SetManualAdjustment 由 Review Agent 设置的强制覆盖 + 写入原因。
func (r *Repository) SetManualAdjustment(ctx context.Context, userID uint64, taskType models.TaskType, newValue int, reason string) error {
	res := r.db.WithContext(ctx).
		Model(&models.UserTimeProfile{}).
		Where("user_id = ? AND task_type = ?", userID, taskType).
		Updates(map[string]interface{}{
			"adjusted_estimate_minutes": newValue,
			"manual_adjustment_reason":  reason,
			"last_updated_at":           gorm.Expr("NOW()"),
			"updated_at":                gorm.Expr("NOW()"),
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrTimeProfileNotFound
	}
	return nil
}

// Delete 删除指定画像（手动 reset 用）。
func (r *Repository) Delete(ctx context.Context, userID uint64, taskType models.TaskType) error {
	res := r.db.WithContext(ctx).
		Where("user_id = ? AND task_type = ?", userID, taskType).
		Delete(&models.UserTimeProfile{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrTimeProfileNotFound
	}
	return nil
}