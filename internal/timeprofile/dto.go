package timeprofile

import "time"

// UserTimeProfileResponse 单条画像的响应。
type UserTimeProfileResponse struct {
	ID                       uint64    `json:"id"`
	UserID                   uint64    `json:"user_id"`
	TaskType                 string    `json:"task_type"`
	InitialEstimateMinutes   int       `json:"initial_estimate_minutes"`
	RollingAvgMinutes        int       `json:"rolling_avg_minutes"`
	AdjustedEstimateMinutes  int       `json:"adjusted_estimate_minutes"`
	SampleCount              int       `json:"sample_count"`
	ManualAdjustmentReason   string    `json:"manual_adjustment_reason,omitempty"`
	LastUpdatedAt            time.Time `json:"last_updated_at"`
	CreatedAt                time.Time `json:"created_at"`
}

// ListProfilesResponse 画像列表响应。
type ListProfilesResponse struct {
	List  []*UserTimeProfileResponse `json:"list"`
	Total int                        `json:"total"`
}

// AdjustByAgentRequest Review Agent 调用的入参（Phase 5 启用，本期不暴露 HTTP）。
type AdjustByAgentRequest struct {
	TaskType   string `json:"task_type" binding:"required,oneof=coding reading writing exercise meeting study other"`
	NewValue   int    `json:"new_value" binding:"required,min=1,max=100000"`
	Reason     string `json:"reason" binding:"required,min=1,max=500"`
}