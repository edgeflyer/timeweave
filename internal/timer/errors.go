package timer

import "errors"

// 业务错误。
var (
	// ErrTimerSessionNotFound 会话不存在。
	ErrTimerSessionNotFound = errors.New("timer: session not found")
	// ErrTimerSessionAccessDenied 无权访问此会话（非所属用户）。
	ErrTimerSessionAccessDenied = errors.New("timer: session access denied")
	// ErrTimerInvalidTransition 状态机非法跃迁。
	ErrTimerInvalidTransition = errors.New("timer: invalid status transition")
	// ErrTimerTaskNotFound 关联的 Task 不存在。
	ErrTimerTaskNotFound = errors.New("timer: task not found")
	// ErrTimerTaskAccessDenied 无权访问关联的 Task。
	ErrTimerTaskAccessDenied = errors.New("timer: task access denied")
	// ErrTimerTaskNotActive 关联的 Task 不是可计时状态（已 done/skipped）。
	ErrTimerTaskNotActive = errors.New("timer: task not active")
)
