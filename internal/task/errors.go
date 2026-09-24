package task

import "errors"

// 业务错误，service 层返回，handler 层翻译为响应码。
var (
	// ErrTaskNotFound Task 不存在。
	ErrTaskNotFound = errors.New("task: not found")
	// ErrTaskAccessDenied 无权访问此 Task（非所属用户）。
	ErrTaskAccessDenied = errors.New("task: access denied")
	// ErrTaskMilestoneNotFound 隶属的 Milestone 不存在。
	ErrTaskMilestoneNotFound = errors.New("task: milestone not found")
	// ErrTaskMilestoneAccessDenied 无权访问隶属的 Milestone（跨用户）。
	ErrTaskMilestoneAccessDenied = errors.New("task: milestone access denied")
	// ErrTaskInvalidStatusTransition 状态机非法跃迁。
	ErrTaskInvalidStatusTransition = errors.New("task: invalid status transition")
	// ErrTaskAlreadyDone 已 done 的任务不能再改状态（Phase 1 锁死）。
	ErrTaskAlreadyDone = errors.New("task: already done")
)
