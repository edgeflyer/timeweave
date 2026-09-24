package schedule

import "time"

// PlanRequest POST /api/v1/schedule/plan 请求体。
// 全部字段可选；不传表示给"今日"做排程。
type PlanRequest struct {
	// Date 排程目标日期（YYYY-MM-DD），不传则为今天。
	Date string `json:"date"`
	// ReplaceExisting 是否清空该日已有时间块后重建。默认 false（在已有基础上补排）。
	ReplaceExisting bool `json:"replace_existing"`
}

// ReplanRequest POST /api/v1/schedule/replan 请求体。
type ReplanRequest struct {
	// Date 需要重排的日期，不传则为今天。
	Date string `json:"date"`
}

// ScheduleBlockResponse 单个时间块响应。
type ScheduleBlockResponse struct {
	ID            uint64    `json:"id"`
	UserID        uint64    `json:"user_id"`
	TaskID        uint64    `json:"task_id"`
	GoalID        uint64    `json:"goal_id"`
	StartAt       time.Time `json:"start_at"`
	EndAt         time.Time `json:"end_at"`
	PriorityScore float64   `json:"priority_score"`
	Reason        string    `json:"reason"`
	Status        int8      `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

// PlanResponse 排程结果响应。
type PlanResponse struct {
	Date         string                  `json:"date"`
	Blocks       []*ScheduleBlockResponse `json:"blocks"`
	PlannedCount int                     `json:"planned_count"`
	SkippedCount int                     `json:"skipped_count"`
}

// ListBlocksResponse 时间块列表响应。
type ListBlocksResponse struct {
	Date   string                   `json:"date"`
	Blocks []*ScheduleBlockResponse `json:"blocks"`
}
