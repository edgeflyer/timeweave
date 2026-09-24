package models

import "time"

// GoalStatus 长期目标状态。
type GoalStatus int8

const (
	GoalStatusActive   GoalStatus = iota // 进行中
	GoalStatusPaused                     // 暂停
	GoalStatusDone                       // 已完成
	GoalStatusArchived                   // 归档
)

// IsValid 校验状态枚举合法性。
func (s GoalStatus) IsValid() bool {
	return s >= GoalStatusActive && s <= GoalStatusArchived
}

// Goal 长期目标，隶属于 User。
type Goal struct {
	BaseModel
	UserID      uint64     `gorm:"not null;index" json:"user_id"`
	Title       string     `gorm:"size:200;not null" json:"title"`
	Description string     `gorm:"size:1000" json:"description"`
	Status      GoalStatus `gorm:"not null;default:0" json:"status"`
	TargetDate  *time.Time `json:"target_date,omitempty"`
	Progress    int        `gorm:"not null;default:0" json:"progress"`
}

// TableName 显式指定表名。
func (g *Goal) TableName() string {
	return "goals"
}