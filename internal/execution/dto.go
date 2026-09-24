package execution

import "time"

// RecordExecutionRequest 是 execution.Service.RecordExecution 的入参。
//
// 注意：这个结构不是 HTTP DTO，而是 Timer.Service.Complete() / Abandon()
// 完成时构造的内部结构体。execution 没有对外的 HTTP 接口（只读历史通过
// timeprofile 模块或者后续 Phase 的 dashboard 提供），写入完全由 Timer 触发。
type RecordExecutionRequest struct {
	TimerSessionID uint64
	UserID         uint64
	TaskID         uint64
	StartAt        time.Time
	EndAt          time.Time
	PauseSeconds   int
	Result         int8 // 1=completed 2=abandoned
}

// TaskExecutionResponse 单条 TaskExecution 的响应。
type TaskExecutionResponse struct {
	ID             uint64    `json:"id"`
	UserID         uint64    `json:"user_id"`
	TaskID         uint64    `json:"task_id"`
	TimerSessionID uint64    `json:"timer_session_id"`
	TaskType       string    `json:"task_type"`
	StartAt        time.Time `json:"start_at"`
	EndAt          time.Time `json:"end_at"`
	PlannedMinutes int       `json:"planned_minutes"`
	ActualMinutes  int       `json:"actual_minutes"`
	PauseSeconds   int       `json:"pause_seconds"`
	OvertimeSeconds int      `json:"overtime_seconds"`
	Result         int8      `json:"result"`
	CreatedAt      time.Time `json:"created_at"`
}

// ListExecutionsResponse 历史执行列表响应（含分页）。
type ListExecutionsResponse struct {
	List     []*TaskExecutionResponse `json:"list"`
	Total    int64                    `json:"total"`
	Page     int                      `json:"page"`
	PageSize int                      `json:"page_size"`
}