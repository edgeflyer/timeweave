package auth

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"timeweave/internal/models"
)

// Repository 封装 User 表的数据库操作。
type Repository struct {
	db *gorm.DB
}

// NewRepository 构造仓库。
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Create 创建用户。
// 返回 ErrEmailExists 表示邮箱已存在（由唯一索引冲突翻译而来）。
func (r *Repository) Create(ctx context.Context, u *models.User) error {
	if err := r.db.WithContext(ctx).Create(u).Error; err != nil {
		// GORM/MySQL 唯一索引冲突错误码 1062
		if isDuplicateKey(err) {
			return ErrEmailExists
		}
		return err
	}
	return nil
}

// GetByEmail 按邮箱查询（不含已软删用户）。
func (r *Repository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var u models.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&u).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

// GetByID 按主键查询（不含已软删用户）。
func (r *Repository) GetByID(ctx context.Context, id uint64) (*models.User, error) {
	var u models.User
	err := r.db.WithContext(ctx).First(&u, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

// UpdateLastLogin 更新最近登录时间。
func (r *Repository) UpdateLastLogin(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).
		Model(&models.User{}).
		Where("id = ?", id).
		Update("last_login_at", gorm.Expr("now()")).Error
}

// isDuplicateKey 简单判断 MySQL 唯一索引冲突。
// 实际生产可换用 github.com/go-sql-driver/mysql 的 *mysql.MySQLError.Number。
func isDuplicateKey(err error) bool {
	if err == nil {
		return false
	}
	// MySQL 错误信息中包含 "Duplicate entry"
	return contains(err.Error(), "Duplicate entry")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr))))
}

// containsMiddle 简易子串扫描（避免引入 strings 而膨胀 import）。
func containsMiddle(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}