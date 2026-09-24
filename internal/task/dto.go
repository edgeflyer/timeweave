package task

import "time"

// CreateTaskRequest POST /api/v1/milestones/:milestoneID/tasks 请求体。
type CreateTaskRequest struct {
	Title            string     `json:"title" binding:"required,min=1,max=200"`
	Description      string     `json:"description" binding:"max=2000"`
	Priority         *int8      `json:"priority" binding:"omitempty,oneof=1 2 3 4"`
	TaskType         string     `json:"task_type" binding:"omitempty,oneof=coding reading writing exercise meeting study other"` // Sprint 5
	EstimatedMinutes int        `json:"estimated_minutes" binding:"omitempty,min=0,max=100000"`                                    // 0 表示未估计
	DueDate          *time.Time `json:"due_date,omitempty"`
	OrderIndex       int        `json:"order_index" binding:"omitempty,min=0"`
}

// UpdateTaskRequest PUT /api/v1/tasks/:id 请求体。
// 不包含 status —— 状态切换必须走 PATCH /tasks/:id/status 走状态机校验。
type UpdateTaskRequest struct {
	Title            *string    `json:"title" binding:"omitempty,min=1,max=200"`
	Description      *string    `json:"description" binding:"omitempty,max=2000"`
	Priority         *int8      `json:"priority" binding:"omitempty,oneof=1 2 3 4"`
	TaskType         *string    `json:"task_type" binding:"omitempty,oneof=coding reading writing exercise meeting study other"` // Sprint 5
	EstimatedMinutes *int       `json:"estimated_minutes" binding:"omitempty,min=0,max=100000"`
	ActualMinutes    *int       `json:"actual_minutes" binding:"omitempty,min=0,max=100000"`
	DueDate          *time.Time `json:"due_date"`
	ClearDueDate     bool       `json:"clear_due_date"`
	OrderIndex       *int       `json:"order_index" binding:"omitempty,min=0"`
}

// UpdateStatusRequest PATCH /api/v1/tasks/:id/status 请求体。
type UpdateStatusRequest struct {
	Status int8 `json:"status" binding:"required,oneof=0 1 2 3 4"`
}

// TaskResponse 单个 Task 响应。
type TaskResponse struct {
	ID               uint64     `json:"id"`
	UserID           uint64     `json:"user_id"`
	MilestoneID      uint64     `json:"milestone_id"`
	GoalID           uint64     `json:"goal_id"`
	Title            string     `json:"title"`
	Description      string     `json:"description"`
	Status           int8       `json:"status"`
	Priority         int8       `json:"priority"`
	TaskType         string     `json:"task_type"` // Sprint 5
	EstimatedMinutes int        `json:"estimated_minutes"`
	ActualMinutes    int        `json:"actual_minutes"`
	DueDate          *time.Time `json:"due_date,omitempty"`
	StartedAt        *time.Time `json:"started_at,omitempty"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
	OrderIndex       int        `json:"order_index"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// ListTasksResponse 列表响应（含分页）。
type ListTasksResponse struct {
	List     []*TaskResponse `json:"list"`
	Total    int64           `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
}
