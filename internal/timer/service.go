package timer

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"timeweave/internal/execution"
	"timeweave/internal/models"
	"timeweave/internal/schedule"
	"timeweave/internal/task"
)

// Service Timer 业务逻辑层。
//
// 职责：
//   - 管理 TimerSession 生命周期（start/pause/resume/complete/abandon）
//   - 联动 Task 状态和 actual_minutes
//   - 联动 ScheduleBlock 状态
//   - 触发 Scheduler 重排（complete 时）
//   - 触发 Observe 层（execution.Service.RecordExecution）落库执行历史
type Service struct {
	repo        *Repository
	taskRepo    *task.Repository
	taskSvc     *task.Service
	scheduleSvc *schedule.Service
	execSvc     *execution.Service
}

// NewService 构造服务。
//
// 依赖顺序：execution 必须在 timer 之前构造（因为 timer.complete 需要回调 execution.RecordExecution）。
func NewService(
	repo *Repository,
	taskRepo *task.Repository,
	taskSvc *task.Service,
	scheduleSvc *schedule.Service,
	execSvc *execution.Service,
) *Service {
	return &Service{
		repo:        repo,
		taskRepo:    taskRepo,
		taskSvc:     taskSvc,
		scheduleSvc: scheduleSvc,
		execSvc:     execSvc,
	}
}

// Start 开始一次计时会话。
//
// 流程：
//  1. 校验 Task 存在 + 属于当前用户 + 状态可计时（todo/doing）
//  2. 强制 pause 当前用户的其他 running session（并发保护）
//  3. 创建新 TimerSession（status=Running, StartedAt=now, LastResumeAt=now）
//  4. 联动 Task.status = doing
//  5. 联动 ScheduleBlock.status = active（如果指定了）
func (s *Service) Start(ctx context.Context, userID uint64, req *StartRequest) (*TimerSessionResponse, error) {
	t, err := s.taskRepo.GetByID(ctx, req.TaskID)
	if err != nil {
		if errors.Is(err, task.ErrTaskNotFound) {
			return nil, ErrTimerTaskNotFound
		}
		return nil, err
	}
	if t.UserID != userID {
		return nil, ErrTimerTaskAccessDenied
	}
	// 已 done / skipped 的任务不能计时
	if t.Status == models.TaskStatusDone || t.Status == models.TaskStatusSkipped {
		return nil, ErrTimerTaskNotActive
	}

	now := time.Now()

	// 并发保护：先 pause 当前所有 running session
	if _, err := s.repo.ForcePauseAllRunning(ctx, userID, now); err != nil {
		return nil, err
	}

	// 创建新 session
	session := &models.TimerSession{
		UserID:        userID,
		TaskID:        req.TaskID,
		ScheduleBlockID: req.ScheduleBlockID,
		StartedAt:     now,
		LastResumeAt:  &now,
		Status:        models.TimerSessionStatusRunning,
	}
	if err := s.repo.Create(ctx, session); err != nil {
		return nil, err
	}

	// 联动：Task.status = doing（通过 service 走状态机）
	if t.Status == models.TaskStatusTodo || t.Status == models.TaskStatusOverdue {
		if _, err := s.taskSvc.UpdateStatus(ctx, userID, req.TaskID, models.TaskStatusDoing); err != nil {
			// 状态机不让跃迁（比如已是 doing）也不致命，记个日志
			slog.Warn("timer start: update task status failed", "task_id", req.TaskID, "err", err)
		}
	}

	// 联动：ScheduleBlock.status = active（如果指定了）
	if req.ScheduleBlockID != nil {
		if err := s.updateScheduleBlockStatus(ctx, *req.ScheduleBlockID, userID, models.ScheduleBlockStatusActive); err != nil {
			slog.Warn("timer start: update schedule block status failed", "block_id", *req.ScheduleBlockID, "err", err)
		}
	}

	return toResponse(session, now), nil
}

// Pause 暂停当前 running 会话。
func (s *Service) Pause(ctx context.Context, userID, sessionID uint64) (*TimerSessionResponse, error) {
	sess, err := s.repo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if sess.UserID != userID {
		return nil, ErrTimerSessionAccessDenied
	}
	if sess.Status != models.TimerSessionStatusRunning {
		return nil, ErrTimerInvalidTransition
	}

	now := time.Now()
	updates := map[string]interface{}{
		"accumulated_seconds": gorm.Expr("accumulated_seconds + ?", int(now.Sub(*sess.LastResumeAt).Seconds())),
		"last_resume_at":      nil,
		"paused_at":           now,
		"status":              models.TimerSessionStatusPaused,
	}
	if err := s.repo.Update(ctx, sessionID, updates); err != nil {
		return nil, err
	}

	sess.Status = models.TimerSessionStatusPaused
	sess.PausedAt = &now
	sess.LastResumeAt = nil
	// accumulated_seconds 由 DB 计算，这里简单估算
	sess.AccumulatedSeconds += int(now.Sub(*sess.LastResumeAt).Seconds())
	return toResponse(sess, now), nil
}

// Resume 恢复 paused 会话。
func (s *Service) Resume(ctx context.Context, userID, sessionID uint64) (*TimerSessionResponse, error) {
	sess, err := s.repo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if sess.UserID != userID {
		return nil, ErrTimerSessionAccessDenied
	}
	if sess.Status != models.TimerSessionStatusPaused {
		return nil, ErrTimerInvalidTransition
	}

	now := time.Now()
	updates := map[string]interface{}{
		"last_resume_at": now,
		"paused_at":      nil,
		"status":         models.TimerSessionStatusRunning,
	}
	if err := s.repo.Update(ctx, sessionID, updates); err != nil {
		return nil, err
	}

	sess.Status = models.TimerSessionStatusRunning
	sess.LastResumeAt = &now
	sess.PausedAt = nil
	return toResponse(sess, now), nil
}

// Complete 完成会话（终态）。
//
// 流程：
//  1. 校验状态机
//  2. 累计最终执行时长
//  3. 关闭会话
//  4. 联动 Task.status = done + Task.actual_minutes += elapsed_minutes
//  5. 联动 ScheduleBlock.status = done（如果关联）
//  6. 触发 Observe（execution.RecordExecution）→ Learn（timeprofile.UpdateProfile）
//  7. 触发 Scheduler.Replan（释放时间块，可能塞下一个 Task）
func (s *Service) Complete(ctx context.Context, userID, sessionID uint64) (*TimerSessionResponse, error) {
	sess, err := s.repo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if sess.UserID != userID {
		return nil, ErrTimerSessionAccessDenied
	}
	if sess.Status != models.TimerSessionStatusRunning && sess.Status != models.TimerSessionStatusPaused {
		return nil, ErrTimerInvalidTransition
	}

	now := time.Now()

	// 计算最终累计时长
	accumulated := sess.AccumulatedSeconds
	if sess.Status == models.TimerSessionStatusRunning && sess.LastResumeAt != nil {
		accumulated += int(now.Sub(*sess.LastResumeAt).Seconds())
	}

	updates := map[string]interface{}{
		"accumulated_seconds": accumulated,
		"last_resume_at":      nil,
		"paused_at":           nil,
		"status":              models.TimerSessionStatusCompleted,
		"completed_at":        now,
	}
	if err := s.repo.Update(ctx, sessionID, updates); err != nil {
		return nil, err
	}

	// 联动：Task.status = done + 累计 actual_minutes
	elapsedMinutes := accumulated / 60
	if elapsedMinutes > 0 {
		t, terr := s.taskRepo.GetByID(ctx, sess.TaskID)
		if terr == nil && t.UserID == userID {
			newActual := t.ActualMinutes + elapsedMinutes
			updates := map[string]interface{}{
				"actual_minutes": newActual,
			}
			// 通过 repo 写（不走状态机，避免互相耦合）
			if uerr := s.taskRepo.Update(ctx, sess.TaskID, updates); uerr != nil {
				slog.Warn("timer complete: update task actual_minutes failed", "task_id", sess.TaskID, "err", uerr)
			}
		}
	}
	if _, err := s.taskSvc.UpdateStatus(ctx, userID, sess.TaskID, models.TaskStatusDone); err != nil {
		slog.Warn("timer complete: update task status failed", "task_id", sess.TaskID, "err", err)
	}

	// 联动：ScheduleBlock.status = done
	if sess.ScheduleBlockID != nil {
		if err := s.updateScheduleBlockStatus(ctx, *sess.ScheduleBlockID, userID, models.ScheduleBlockStatusDone); err != nil {
			slog.Warn("timer complete: update schedule block status failed", "block_id", *sess.ScheduleBlockID, "err", err)
		}
	}

	// 触发 Observe（落库 TaskExecution + Learn 自动接续）
	if s.execSvc != nil {
		// 暂停时长 = 整段时间 - 累计活跃秒数（TimerSession 没有单独的 pause_seconds 字段）
		wallSec := int(now.Sub(sess.StartedAt).Seconds())
		pauseSec := wallSec - accumulated
		if pauseSec < 0 {
			pauseSec = 0
		}
		if _, err := s.execSvc.RecordExecution(ctx, &execution.RecordExecutionRequest{
			TimerSessionID: sess.ID,
			UserID:         userID,
			TaskID:         sess.TaskID,
			StartAt:        sess.StartedAt,
			EndAt:          now,
			PauseSeconds:   pauseSec,
			Result:         int8(models.ExecutionResultCompleted),
		}); err != nil {
			slog.Warn("timer complete: record execution failed", "task_id", sess.TaskID, "err", err)
		}
	}

	// 触发 Replan（释放时间块，可能塞下一个 Task）
	if s.scheduleSvc != nil {
		if _, err := s.scheduleSvc.Replan(ctx, userID, now); err != nil {
			slog.Warn("timer complete: replan failed", "err", err)
		}
	}

	sess.Status = models.TimerSessionStatusCompleted
	sess.CompletedAt = &now
	sess.AccumulatedSeconds = accumulated
	return toResponse(sess, now), nil
}

// Abandon 放弃会话（终态，不联动 Task，Task 状态由用户决定）。
func (s *Service) Abandon(ctx context.Context, userID, sessionID uint64) (*TimerSessionResponse, error) {
	sess, err := s.repo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if sess.UserID != userID {
		return nil, ErrTimerSessionAccessDenied
	}
	if sess.Status != models.TimerSessionStatusRunning && sess.Status != models.TimerSessionStatusPaused {
		return nil, ErrTimerInvalidTransition
	}

	now := time.Now()

	// 累计到放弃时刻
	accumulated := sess.AccumulatedSeconds
	if sess.Status == models.TimerSessionStatusRunning && sess.LastResumeAt != nil {
		accumulated += int(now.Sub(*sess.LastResumeAt).Seconds())
	}

	updates := map[string]interface{}{
		"accumulated_seconds": accumulated,
		"last_resume_at":      nil,
		"paused_at":           nil,
		"status":              models.TimerSessionStatusAbandoned,
		"abandoned_at":        now,
	}
	if err := s.repo.Update(ctx, sessionID, updates); err != nil {
		return nil, err
	}

	// ScheduleBlock 回到 pending（让用户可以重新执行）
	if sess.ScheduleBlockID != nil {
		if err := s.updateScheduleBlockStatus(ctx, *sess.ScheduleBlockID, userID, models.ScheduleBlockStatusPending); err != nil {
			slog.Warn("timer abandon: update schedule block status failed", "err", err)
		}
	}

	// 触发 Observe（落库 TaskExecution，但 Result=Abandoned 不进 Learn 统计）
	if s.execSvc != nil {
		wallSec := int(now.Sub(sess.StartedAt).Seconds())
		pauseSec := wallSec - accumulated
		if pauseSec < 0 {
			pauseSec = 0
		}
		if _, err := s.execSvc.RecordExecution(ctx, &execution.RecordExecutionRequest{
			TimerSessionID: sess.ID,
			UserID:         userID,
			TaskID:         sess.TaskID,
			StartAt:        sess.StartedAt,
			EndAt:          now,
			PauseSeconds:   pauseSec,
			Result:         int8(models.ExecutionResultAbandoned),
		}); err != nil {
			slog.Warn("timer abandon: record execution failed", "task_id", sess.TaskID, "err", err)
		}
	}

	sess.Status = models.TimerSessionStatusAbandoned
	sess.AbandonedAt = &now
	sess.AccumulatedSeconds = accumulated
	return toResponse(sess, now), nil
}

// Active 查询当前活跃的 running session。
func (s *Service) Active(ctx context.Context, userID uint64) (*TimerSessionResponse, error) {
	sess, err := s.repo.GetActiveByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if sess == nil {
		return nil, nil
	}
	return toResponse(sess, time.Now()), nil
}

// History 列最近 N 个会话。
func (s *Service) History(ctx context.Context, userID uint64, limit int) ([]*TimerSessionResponse, error) {
	list, err := s.repo.ListByUserID(ctx, userID, limit)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	out := make([]*TimerSessionResponse, 0, len(list))
	for i := range list {
		out = append(out, toResponse(&list[i], now))
	}
	return out, nil
}

// updateScheduleBlockStatus 直接更新 ScheduleBlock 状态（不走 schedule.Service，避免循环）。
func (s *Service) updateScheduleBlockStatus(ctx context.Context, blockID, userID uint64, status models.ScheduleBlockStatus) error {
	// 这里直接用 GORM 更新（schedule.Service 没暴露单条 update 接口）
	// Phase 2 简化：只校验归属，不做复杂业务规则
	return s.dbUpdateBlockStatus(ctx, blockID, userID, status)
}

// dbUpdateBlockStatus 通过注入的 db 客户端更新（注入方式由 main.go 提供）。
// 实际实现放在 service 里——见 NewService 注入 schedule.Service。
// 这里简化为用 task.Repository.db（不优雅，但避免再开一个依赖）：
//   - Phase 2.1 可以改成注入一个 ScheduleBlockUpdater 接口
// 当前实现：从 schedule.Service 内部获取 db，但 schedule.Service 没暴露。
// 折中方案：在 main.go 装配时直接给 timer.Service 一个 db handle。
// 见 timer/wiring.go。
func (s *Service) dbUpdateBlockStatus(ctx context.Context, blockID, userID uint64, status models.ScheduleBlockStatus) error {
	// 通过 schedule.Service 更新
	return s.scheduleSvc.UpdateBlockStatus(ctx, userID, blockID, status)
}

// toResponse 模型转 DTO。
func toResponse(s *models.TimerSession, now time.Time) *TimerSessionResponse {
	return &TimerSessionResponse{
		ID:                 s.ID,
		UserID:             s.UserID,
		TaskID:             s.TaskID,
		ScheduleBlockID:    s.ScheduleBlockID,
		StartedAt:          s.StartedAt,
		LastResumeAt:       s.LastResumeAt,
		PausedAt:           s.PausedAt,
		AccumulatedSeconds: s.AccumulatedSeconds,
		CurrentElapsed:     s.CurrentElapsedSeconds(now),
		Status:             int8(s.Status),
		CompletedAt:        s.CompletedAt,
		AbandonedAt:        s.AbandonedAt,
		CreatedAt:          s.CreatedAt,
	}
}
