package schedule

import "errors"

// 业务错误，service 层返回，handler 层翻译为响应码。
var (
	// ErrScheduleBlockNotFound 时间块不存在。
	ErrScheduleBlockNotFound = errors.New("schedule: block not found")
	// ErrScheduleBlockAccessDenied 无权访问此时间块（非所属用户）。
	ErrScheduleBlockAccessDenied = errors.New("schedule: block access denied")
	// ErrScheduleTimeConflict 时间段冲突。
	ErrScheduleTimeConflict = errors.New("schedule: time conflict")
	// ErrScheduleInvalidTimeRange 时间范围无效（end <= start）。
	ErrScheduleInvalidTimeRange = errors.New("schedule: invalid time range")
	// ErrScheduleNoTasksToPlan 没有可排的任务。
	ErrScheduleNoTasksToPlan = errors.New("schedule: no active tasks to plan")
	// ErrScheduleInsufficientTime 可用时间不够塞下任何任务。
	ErrScheduleInsufficientTime = errors.New("schedule: insufficient time")
)
