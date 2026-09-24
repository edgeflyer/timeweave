package models

import "time"

// ScheduleBlockStatus 时间块状态。
type ScheduleBlockStatus int8

const (
	ScheduleBlockStatusPending ScheduleBlockStatus = iota // 待执行
	ScheduleBlockStatusActive                             // 进行中（Timer 已启动，Phase 2）
	ScheduleBlockStatusDone                              // 已完成（关联 Task 也 done）
	ScheduleBlockStatusSkipped                           // 已跳过
	ScheduleBlockStatusMissed                            // 已错过（时间过了还没做）
)

// ScheduleBlock 一个时间段内的具体 Task 安排。
//
// 与 Task 的关系：Task 是"想做的事"，ScheduleBlock 是"日程表上的格子"。
// 一个 Task 可能产生 0~N 个 ScheduleBlock（取决于 Scheduler 重排次数）。
type ScheduleBlock struct {
	BaseModel
	UserID        uint64              `gorm:"not null;index:idx_user_date,priority:1" json:"user_id"`
	TaskID        uint64              `gorm:"not null;index" json:"task_id"`
	GoalID        uint64              `gorm:"not null;index" json:"goal_id"`
	StartAt       time.Time           `gorm:"not null;index:idx_user_date,priority:2" json:"start_at"`
	EndAt         time.Time           `gorm:"not null;index" json:"end_at"`
	PriorityScore float64             `gorm:"not null;default:0" json:"priority_score"`
	Reason        string              `gorm:"size:500" json:"reason"`
	Status        ScheduleBlockStatus `gorm:"not null;default:0" json:"status"`
}

// TableName 指定表名。
func (s *ScheduleBlock) TableName() string {
	return "schedule_blocks"
}
