// Package db 负责 GORM 的初始化、连接池配置、以及优雅关闭。
//
// 设计原则：
//   - 不在 db 包里写业务逻辑或 AutoMigrate，专注「连接 + 池」
//   - AutoMigrate 由 main.go（或专门的 migration 命令）调用
//   - 启动时立刻 Ping，fail fast：DB 连不上就别启动服务
//
// 用法：
//
//	db, err := db.Open(cfg)
//	if err != nil { ... }
//	defer db.Close(db)
package db

import (
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"timeweave/internal/config"
)

// Open 根据 Config 创建 GORM 连接，并设置合理的连接池参数。
//
// 返回的 *gorm.DB 是全局共享的，**不要**在多次调用中重复创建。
func Open(cfg *config.Config) (*gorm.DB, error) {
	gormCfg := &gorm.Config{
		// 默认只打 Warn 及以上，避免开发环境刷屏
		Logger: logger.Default.LogMode(logger.Warn),
	}

	db, err := gorm.Open(mysql.Open(cfg.DSN()), gormCfg)
	if err != nil {
		return nil, fmt.Errorf("connect mysql: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get underlying *sql.DB: %w", err)
	}

	// 连接池参数
	sqlDB.SetMaxIdleConns(10)           // 空闲连接池上限
	sqlDB.SetMaxOpenConns(100)          // 打开连接上限
	sqlDB.SetConnMaxLifetime(time.Hour) // 连接最长存活时间

	// 启动时 ping 一下，连不上就 fail fast
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping mysql: %w", err)
	}

	return db, nil
}

// Close 关闭底层 *sql.DB。main.go 里用 defer 调用。
func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}