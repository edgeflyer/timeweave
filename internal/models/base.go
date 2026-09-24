// Package models 集中放所有 GORM 数据模型。
//
// 设计原则：
//   - 所有业务 model 嵌入 BaseModel，自动获得主键 + 时间戳 + 软删除
//   - 敏感字段（如 password_hash）必须打 json:"-" 防止 API 泄露
//   - 每个 model 显式写 TableName()，不依赖 GORM 默认复数规则
package models

import (
	"time"

	"gorm.io/gorm"
)

// BaseModel 是所有业务 model 的基类，匿名嵌入即可继承全部通用字段。
//
// 使用：
//
//	type User struct {
//	    BaseModel
//	    Email string
//	}
//
// 字段说明：
//   - ID 用 uint64 而非 GORM 默认的 uint，避免跨平台（32/64 位）隐患
//   - DeletedAt 加索引，软删除查询更快；json:"-" 防止内部字段泄露到 API
type BaseModel struct {
	ID        uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}