package timer

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"timeweave/internal/models"
)

// Repository 封装 TimerSession 表的数据库操作。
type Repository struct {
	db *gorm.DB
}

// NewRepository 构造仓库。
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Create 创建会话。
func (r *Repository) Create(ctx context.Context, s *models.TimerSession) error {
	return r.db.WithContext(ctx).Create(s).Error
}

// GetByID 按主键查询。
func (r *Repository) GetByID(ctx context.Context, id uint64) (*models.TimerSession, error) {
	var s models.TimerSession
	err := r.db.WithContext(ctx).First(&s, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTimerSessionNotFound
		}
		return nil, err
	}
	return &s, nil
}

// GetActiveByUserID 取当前用户的 running session（最多一个）。
func (r *Repository) GetActiveByUserID(ctx context.Context, userID uint64) (*models.TimerSession, error) {
	var s models.TimerSession
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND status = ?", userID, models.TimerSessionStatusRunning).
		First(&s).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // 没找到不算错
		}
		return nil, err
	}
	return &s, nil
}

// ForcePauseAllRunning 把某用户所有 running session 强制置为 paused（start 前的并发保护）。
// 返回被修改的 session 列表（用于回写 paused_at）。
func (r *Repository) ForcePauseAllRunning(ctx context.Context, userID uint64, now time.Time) ([]models.TimerSession, error) {
	var sessions []models.TimerSession
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. 查所有 running
		if err := tx.Where("user_id = ? AND status = ?", userID, models.TimerSessionStatusRunning).
			Find(&sessions).Error; err != nil {
			return err
		}
		if len(sessions) == 0 {
			return nil
		}
		// 2. 逐个累计 + 置 paused
		for i := range sessions {
			s := &sessions[i]
			if s.LastResumeAt != nil {
				s.AccumulatedSeconds += int(now.Sub(*s.LastResumeAt).Seconds())
			}
			s.LastResumeAt = nil
			s.PausedAt = &now
			s.Status = models.TimerSessionStatusPaused
			if err := tx.Save(s).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return sessions, err
}

// Update 按主键更新字段。
func (r *Repository) Update(ctx context.Context, id uint64, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}
	result := r.db.WithContext(ctx).
		Model(&models.TimerSession{}).
		Where("id = ?", id).
		Updates(updates)
	return result.Error
}

// ListByUserID 列出某用户最近的 N 个会话（按 id DESC）。
func (r *Repository) ListByUserID(ctx context.Context, userID uint64, limit int) ([]models.TimerSession, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var list []models.TimerSession
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("id DESC").
		Limit(limit).
		Find(&list).Error
	return list, err
}
