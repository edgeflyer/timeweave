package models

import "time"

// ExecutionResult 一次 Task 执行的结果标记。
//
// 用于聚合统计时区分（默认仅 Completed 计入 UserTimeProfile 的 rolling_avg，
// Abandoned 的执行虽然落库但不计学习样本，因为中途放弃不代表真实耗时节奏）。
type ExecutionResult int8

const (
	ExecutionResultCompleted ExecutionResult = 1 // 正常完成（Timer Complete）
	ExecutionResultAbandoned ExecutionResult = 2 // 中途放弃（Timer Abandon）
)

// TaskExecution 一次 Task 执行的完整快照（Observe 层数据源 → Learn 端输入）。
//
// 设计要点：
//   - 字段冗余 TaskType：避免 Learn 聚合时 JOIN tasks 表（高频读路径）
//   - 字段冗余 UserID：避免每次按 user 过滤都走 JOIN
//   - PlannedMinutes 是开始计时那一刻 Task.estimated_minutes 的快照，
//     用于事后分析「计划 vs 实际」偏差，不随 Task.estimated_minutes 后续被 Learn 修改而变化
//   - OvertimeSeconds = ActualMinutes*60 - PlannedMinutes*60（正数代表超时，负数代表提前）
type TaskExecution struct {
	BaseModel
	UserID          uint64          `gorm:"not null;index:idx_exec_user_type,priority:1" json:"user_id"`
	TaskID          uint64          `gorm:"not null;index" json:"task_id"`
	TimerSessionID  uint64          `gorm:"not null;index" json:"timer_session_id"`
	TaskType        TaskType        `gorm:"size:20;not null;index:idx_exec_user_type,priority:2" json:"task_type"`
	StartAt         time.Time       `gorm:"not null" json:"start_at"`
	EndAt           time.Time       `gorm:"not null" json:"end_at"`
	PlannedMinutes  int             `gorm:"not null;default:0" json:"planned_minutes"`
	ActualMinutes   int             `gorm:"not null;default:0" json:"actual_minutes"`
	PauseSeconds    int             `gorm:"not null;default:0" json:"pause_seconds"`
	OvertimeSeconds int             `gorm:"not null;default:0" json:"overtime_seconds"`
	Result          ExecutionResult `gorm:"not null;default:1" json:"result"`
}

// TableName 指定表名。
func (TaskExecution) TableName() string {
	return "task_executions"
}