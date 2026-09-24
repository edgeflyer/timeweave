// Package execution 是 QuestOS 闭环里的 Observe 层：
//
//   - 接收 Timer.complete / Timer.abandon 事件
//   - 把执行快照落库到 task_executions 表
//   - 触发 Learn 层（timeprofile.Service.UpdateProfile）更新 UserTimeProfile
//
// 与 timeprofile 的关系：execution 是「数据写入」，timeprofile 是「数据学习」。
// 两者通过 TimeProfileUpdater 接口解耦，main.go 装配时注入。
package execution

import "errors"

// 业务错误，service 层返回，handler 层翻译为响应码。
var (
	// ErrExecutionNotFound TaskExecution 不存在。
	ErrExecutionNotFound = errors.New("execution: not found")
	// ErrExecutionTaskNotFound 关联的 Task 不存在。
	ErrExecutionTaskNotFound = errors.New("execution: task not found")
	// ErrExecutionTaskAccessDenied 无权访问关联的 Task（非所属用户）。
	ErrExecutionTaskAccessDenied = errors.New("execution: task access denied")
	// ErrExecutionInvalidSession 无效的 TimerSession（重复落库或状态非法）。
	ErrExecutionInvalidSession = errors.New("execution: invalid timer session")
)