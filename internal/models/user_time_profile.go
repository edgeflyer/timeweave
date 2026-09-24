package models

import "time"

// UserTimeProfile 用户对某类任务的耗时画像（Learn 端产物）。
//
// 数据生命周期：
//   - 首次某 user × task_type 有 TaskExecution 完成时创建（initial 来自 Task.estimated_minutes）
//   - 每次 TaskExecution 完成 → rolling_avg 更新 → adjusted 重新计算 → 回写同类未完成 Task
//   - Phase 5 Review Agent 可通过 AdjustByAgent 强制覆盖 adjusted_estimate（写 manual_adjustment_reason）
//
// 算法（详见 timeprofile/service.go）：
//   - rolling_avg = (old_avg * old_count + actual) / (old_count + 1)  简单滚动平均
//   - sample_count < 10：adjusted = 0.7 * initial + 0.3 * rolling（冷启动加权，防异常值带偏）
//   - sample_count >= 10：adjusted = rolling（热态纯均值）
type UserTimeProfile struct {
	BaseModel
	UserID                  uint64    `gorm:"not null;uniqueIndex:idx_utp_user_type,priority:1" json:"user_id"`
	TaskType                TaskType  `gorm:"size:20;not null;uniqueIndex:idx_utp_user_type,priority:2" json:"task_type"`
	InitialEstimateMinutes  int       `gorm:"not null;default:0" json:"initial_estimate_minutes"`
	RollingAvgMinutes       int       `gorm:"not null;default:0" json:"rolling_avg_minutes"`
	AdjustedEstimateMinutes int       `gorm:"not null;default:0" json:"adjusted_estimate_minutes"`
	SampleCount             int       `gorm:"not null;default:0" json:"sample_count"`
	ManualAdjustmentReason  string    `gorm:"size:500" json:"manual_adjustment_reason,omitempty"`
	LastUpdatedAt           time.Time `gorm:"not null" json:"last_updated_at"`
}

// TableName 指定表名。
func (UserTimeProfile) TableName() string {
	return "user_time_profiles"
}