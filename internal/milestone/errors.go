package milestone

import "errors"

// 业务错误，service 层返回，handler 层翻译为响应码。
var (
	// ErrMilestoneNotFound Milestone 不存在。
	ErrMilestoneNotFound = errors.New("milestone: not found")
	// ErrMilestoneAccessDenied 无权访问此 Milestone（非所属用户）。
	ErrMilestoneAccessDenied = errors.New("milestone: access denied")
	// ErrMilestoneGoalNotFound 隶属的 Goal 不存在。
	ErrMilestoneGoalNotFound = errors.New("milestone: goal not found")
	// ErrMilestoneGoalAccessDenied 无权访问隶属的 Goal（跨用户）。
	ErrMilestoneGoalAccessDenied = errors.New("milestone: goal access denied")
)