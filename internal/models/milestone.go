package models

import (
	"time"
)

// MilestoneStatus 阶段状态。
type MilestoneStatus int8

const (
	MilestoneStatusActive  MilestoneStatus = iota // 进行中
	MilestoneStatusDone                           // 已完成
	MilestoneStatusOverdue                        // 已逾期（Phase 3 Scheduler 标记）
)

// Milestone 阶段目标（隶属于 Goal + User）。
type Milestone struct {
	BaseModel
	UserID      uint64          `gorm:"not null;index" json:"user_id"`
	GoalID      uint64          `gorm:"not null;index" json:"goal_id"`
	Title       string          `gorm:"size:200;not null" json:"title"`
	Description string          `gorm:"size:1000" json:"description"`
	Status      MilestoneStatus `gorm:"not null;default:0" json:"status"`
	TargetDate  *time.Time      `json:"target_date,omitempty"`
	OrderIndex  int             `gorm:"not null;default:0" json:"order_index"`
}

// TableName 指定表名。
func (m *Milestone) TableName() string {
	return "milestones"
}