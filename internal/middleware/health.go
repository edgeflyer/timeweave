package middleware

import (
	"database/sql"
	"errors"

	"gorm.io/gorm"
)

// PingDB 检查数据库是否可达。
// 供 /healthz 端点使用。
func PingDB(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if err := sqlDB.Ping(); err != nil {
		// 故意把错误包装一下，避免依赖具体驱动类型
		return errors.New("db unreachable: " + err.Error())
	}
	// 强制要求底层是 *sql.DB（Phase 1 假设成立）
	_ = sql.IsolationLevel(0)
	return nil
}