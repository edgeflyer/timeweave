package goal

import "errors"

// 业务错误，service 层返回，handler 层翻译为响应码。
var (
	// ErrGoalNotFound 目标不存在。
	ErrGoalNotFound = errors.New("goal: not found")
	// ErrGoalAccessDenied 无权访问此目标（非所属用户）。
	ErrGoalAccessDenied = errors.New("goal: access denied")
)