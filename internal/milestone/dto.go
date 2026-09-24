package milestone

import "time"

// CreateMilestoneRequest POST /api/v1/goals/:goalID/milestones 请求体。
type CreateMilestoneRequest struct {
	Title       string     `json:"title" binding:"required,min=1,max=200"`
	Description string     `json:"description" binding:"max=1000"`
	TargetDate  *time.Time `json:"target_date,omitempty"`
	OrderIndex  int        `json:"order_index" binding:"omitempty,min=0"`
}

// UpdateMilestoneRequest PUT /api/v1/milestones/:id 请求体。
type UpdateMilestoneRequest struct {
	Title       *string    `json:"title" binding:"omitempty,min=1,max=200"`
	Description *string    `json:"description" binding:"omitempty,max=1000"`
	Status      *int8      `json:"status" binding:"omitempty,oneof=0 1 2"`
	TargetDate  *time.Time `json:"target_date"`
	OrderIndex  *int       `json:"order_index" binding:"omitempty,min=0"`
	ClearTargetDate bool   `json:"clear_target_date"`
}

// MilestoneResponse 单个 Milestone 响应。
type MilestoneResponse struct {
	ID          uint64     `json:"id"`
	GoalID      uint64     `json:"goal_id"`
	UserID      uint64     `json:"user_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      int8       `json:"status"`
	TargetDate  *time.Time `json:"target_date,omitempty"`
	OrderIndex  int        `json:"order_index"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ListMilestonesResponse 列表响应（含分页）。
type ListMilestonesResponse struct {
	List     []*MilestoneResponse `json:"list"`
	Total    int64                `json:"total"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"page_size"`
}