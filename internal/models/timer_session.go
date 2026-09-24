package models

import "time"

// TimerSessionStatus 计时会话状态。
type TimerSessionStatus int8

const (
	TimerSessionStatusRunning   TimerSessionStatus = iota // 进行中
	TimerSessionStatusPaused                             // 已暂停
	TimerSessionStatusCompleted                          // 已完成
	TimerSessionStatusAbandoned                          // 已放弃
)

// TimerSession 一次计时会话（= 用户对一个 Task 的一次完整执行记录）。
//
// 服务端权威：StartedAt、LastResumeAt、PausedAt 全部是服务端时间戳，
// 客户端只是"按钮"，杀 App 重启也能恢复。
//
// 状态约定：
//   - Running  ：LastResumeAt != nil, PausedAt == nil
//   - Paused   ：LastResumeAt == nil, PausedAt != nil
//   - Completed/Abandoned：终态，LastResumeAt / PausedAt 都为 nil
type TimerSession struct {
	BaseModel
	UserID             uint64             `gorm:"not null;index:idx_user_status,priority:1" json:"user_id"`
	TaskID             uint64             `gorm:"not null;index" json:"task_id"`
	ScheduleBlockID    *uint64            `gorm:"index" json:"schedule_block_id,omitempty"`
	StartedAt          time.Time          `gorm:"not null" json:"started_at"`
	LastResumeAt       *time.Time         `json:"last_resume_at,omitempty"`
	PausedAt           *time.Time         `json:"paused_at,omitempty"`
	AccumulatedSeconds int                `gorm:"not null;default:0" json:"accumulated_seconds"`
	Status             TimerSessionStatus `gorm:"not null;default:0;index:idx_user_status,priority:2" json:"status"`
	CompletedAt        *time.Time         `json:"completed_at,omitempty"`
	AbandonedAt        *time.Time         `json:"abandoned_at,omitempty"`
}

// TableName 指定表名。
func (t *TimerSession) TableName() string {
	return "timer_sessions"
}

// CurrentElapsedSeconds 返回"如果现在关闭会话，将累计多少秒"。
// 只在 running/paused 状态有意义；终态请直接读 AccumulatedSeconds。
func (t *TimerSession) CurrentElapsedSeconds(now time.Time) int {
	if t.Status == TimerSessionStatusRunning && t.LastResumeAt != nil {
		return t.AccumulatedSeconds + int(now.Sub(*t.LastResumeAt).Seconds())
	}
	return t.AccumulatedSeconds
}
