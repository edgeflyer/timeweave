package timer

import "time"

// StartRequest POST /api/v1/timer/start 请求体。
type StartRequest struct {
	TaskID           uint64  `json:"task_id" binding:"required"`
	ScheduleBlockID  *uint64 `json:"schedule_block_id,omitempty"`
}

// SessionIDRequest 携带 session_id 的请求体（pause/resume/complete/abandon）。
type SessionIDRequest struct {
	SessionID uint64 `json:"session_id" binding:"required"`
}

// AbandonRequest 放弃会话的请求体（reason 可选）。
type AbandonRequest struct {
	SessionID uint64 `json:"session_id" binding:"required"`
	Reason    string `json:"reason" binding:"max=500"`
}

// TimerSessionResponse 单个会话响应。
type TimerSessionResponse struct {
	ID                 uint64     `json:"id"`
	UserID             uint64     `json:"user_id"`
	TaskID             uint64     `json:"task_id"`
	ScheduleBlockID    *uint64    `json:"schedule_block_id,omitempty"`
	StartedAt          time.Time  `json:"started_at"`
	LastResumeAt       *time.Time `json:"last_resume_at,omitempty"`
	PausedAt           *time.Time `json:"paused_at,omitempty"`
	AccumulatedSeconds int        `json:"accumulated_seconds"`
	CurrentElapsed     int        `json:"current_elapsed_seconds"` // 计算字段（running/paused 时）
	Status             int8       `json:"status"`
	CompletedAt        *time.Time `json:"completed_at,omitempty"`
	AbandonedAt        *time.Time `json:"abandoned_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
}
