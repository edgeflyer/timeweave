package goal

import "time"

// CreateGoalRequest POST /api/v1/goals 请求体。
type CreateGoalRequest struct {
	Title       string     `json:"title" binding:"required,min=1,max=200"`
	Description string     `json:"description" binding:"max=1000"`
	TargetDate  *time.Time `json:"target_date,omitempty"`
}

// UpdateGoalRequest PUT /api/v1/goals/:id 请求体。
// 指针字段表示「可选 + 不传 = 不更新」。
type UpdateGoalRequest struct {
	Title       *string    `json:"title" binding:"omitempty,min=1,max=200"`
	Description *string    `json:"description" binding:"omitempty,max=1000"`
	Status      *int8      `json:"status" binding:"omitempty,oneof=0 1 2 3"`
	TargetDate  *time.Time `json:"target_date"`
	Progress    *int       `json:"progress" binding:"omitempty,min=0,max=100"`
	// 显式清空 target_date 的标记
	ClearTargetDate bool `json:"clear_target_date"`
}

// GoalResponse 单个目标响应。
type GoalResponse struct {
	ID          uint64     `json:"id"`
	UserID      uint64     `json:"user_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      int8       `json:"status"`
	TargetDate  *time.Time `json:"target_date,omitempty"`
	Progress    int        `json:"progress"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ListGoalsResponse 列表响应（含分页）。
type ListGoalsResponse struct {
	List     []*GoalResponse `json:"list"`
	Total    int64           `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
}