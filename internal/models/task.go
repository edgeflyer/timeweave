package models

import "time"

// TaskStatus 任务状态机。
//
// 合法跃迁（详见 task/service.go 的 isValidTransition）：
//   todo    -> doing, skipped
//   doing   -> todo, done, skipped
//   skipped -> todo
//   done    -> 终态（不允许再改，Phase 2 可放开"重做"）
//   overdue -> doing, todo, skipped
type TaskStatus int8

const (
	TaskStatusTodo    TaskStatus = iota // 0 待办
	TaskStatusDoing                     // 1 进行中
	TaskStatusDone                      // 2 已完成
	TaskStatusSkipped                   // 3 已跳过
	TaskStatusOverdue                   // 4 已逾期（Phase 3 Scheduler 自动标记）
)

// TaskPriority 任务优先级。
type TaskPriority int8

const (
	TaskPriorityLow       TaskPriority = 1 // 低
	TaskPriorityNormal    TaskPriority = 2 // 中（默认）
	TaskPriorityImportant TaskPriority = 3 // 高
	TaskPriorityUrgent    TaskPriority = 4 // 紧急
)

// TaskType 任务类型（Learn 端 UserTimeProfile 的分桶维度）。
//
// 选这个维度的原因：同类任务（如 coding / reading）的真实耗时节奏有强相关性，
// 不同类之间（如 coding vs meeting）几乎没有可比性。
// 类型用 string 而非 int 是为了让 DB 里可读、加新类型不影响老数据迁移。
type TaskType string

const (
	TaskTypeCoding   TaskType = "coding"   // 写代码
	TaskTypeReading  TaskType = "reading"  // 阅读资料 / 文献
	TaskTypeWriting  TaskType = "writing"  // 写文章 / 文档
	TaskTypeExercise TaskType = "exercise" // 运动 / 健身
	TaskTypeMeeting  TaskType = "meeting"  // 会议 / 沟通
	TaskTypeStudy    TaskType = "study"    // 学习 / 课程
	TaskTypeOther    TaskType = "other"    // 其他（默认值，未明确分类时落这里）
)

// IsValidTaskType 校验 TaskType 是否在合法枚举中。
func IsValidTaskType(t TaskType) bool {
	switch t {
	case TaskTypeCoding, TaskTypeReading, TaskTypeWriting,
		TaskTypeExercise, TaskTypeMeeting, TaskTypeStudy, TaskTypeOther:
		return true
	}
	return false
}

// Task 任务（隶属于 Milestone + User）。
//
// 字段说明：
//   - GoalID 是冗余字段，方便按用户聚合查询（不用每次 JOIN milestones）。
//   - TaskType 用于 Learn 端按类型聚合 UserTimeProfile。Sprint 5 引入。
//   - EstimatedMinutes / ActualMinutes / StartedAt / CompletedAt
//     是 Phase 2 Timer 服务的写入位；Phase 1 由用户手动填写。
//   - Status 字段切换必须走 PATCH /tasks/:id/status，不能在 Update 里随便改。
type Task struct {
	BaseModel
	UserID           uint64       `gorm:"not null;index:idx_task_user_milestone,priority:1" json:"user_id"`
	MilestoneID      uint64       `gorm:"not null;index:idx_task_user_milestone,priority:2" json:"milestone_id"`
	GoalID           uint64       `gorm:"not null;index" json:"goal_id"`
	Title            string       `gorm:"size:200;not null" json:"title"`
	Description      string       `gorm:"size:2000" json:"description"`
	Status           TaskStatus   `gorm:"not null;default:0" json:"status"`
	Priority         TaskPriority `gorm:"not null;default:2" json:"priority"`
	TaskType         TaskType     `gorm:"size:20;not null;default:'other';index:idx_task_user_type,priority:1" json:"task_type"`
	EstimatedMinutes int          `gorm:"not null;default:0" json:"estimated_minutes"`
	ActualMinutes    int          `gorm:"not null;default:0" json:"actual_minutes"`
	DueDate          *time.Time   `json:"due_date,omitempty"`
	StartedAt        *time.Time   `json:"started_at,omitempty"`
	CompletedAt      *time.Time   `json:"completed_at,omitempty"`
	OrderIndex       int          `gorm:"not null;default:0" json:"order_index"`
}

// TableName 指定表名。
func (t *Task) TableName() string {
	return "tasks"
}
