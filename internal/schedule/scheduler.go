package schedule

import (
	"sort"
	"time"

	"timeweave/internal/models"
)

// ScoredTask 评分后的 Task（含排序依据）。
type ScoredTask struct {
	Task   models.Task
	Score  float64
	Reason string
}

// Slot 可用时间段。
type Slot struct {
	Start time.Time
	End   time.Time
}

// Plan 排程计划（中间结构）。
type Plan struct {
	Date   time.Time
	Slots  []Slot            // 当日可用时间段
	Tasks  []ScoredTask       // 评分排序后的任务池
	Blocks []*models.ScheduleBlock // 生成的时间块
}

// scoreTasks 给每个 Task 打分。
//
// 评分维度（Phase 3 简化版）：
//   - 优先级权重：1-4 档 → 10/20/30/40 分
//   - 紧急度：基于 due_date 距离 now 的时长
//   - 短任务加分：≤30 分钟额外 10 分（容易塞进时间缝）
func scoreTasks(tasks []models.Task, now time.Time) []ScoredTask {
	out := make([]ScoredTask, 0, len(tasks))
	for _, t := range tasks {
		score := 0.0
		reasons := make([]string, 0, 4)

		// 1. 优先级
		priScore := float64(t.Priority) * 10
		score += priScore
		reasons = append(reasons, priorityLabel(int8(t.Priority)))

		// 2. 紧急度
		if t.DueDate != nil {
			hoursLeft := t.DueDate.Sub(now).Hours()
			switch {
			case hoursLeft < 0:
				score += 40
				reasons = append(reasons, "已逾期")
			case hoursLeft < 24:
				score += 35
				reasons = append(reasons, "24小时内截止")
			case hoursLeft < 72:
				score += 25
				reasons = append(reasons, "3天内截止")
			case hoursLeft < 168:
				score += 15
				reasons = append(reasons, "本周截止")
			default:
				score += 5
				reasons = append(reasons, "长期任务")
			}
		} else {
			score += 10
			reasons = append(reasons, "无截止")
		}

		// 3. 短任务奖励（容易填充碎片时间）
		switch {
		case t.EstimatedMinutes > 0 && t.EstimatedMinutes <= 30:
			score += 10
			reasons = append(reasons, "短任务")
		case t.EstimatedMinutes > 30 && t.EstimatedMinutes <= 60:
			score += 5
		}

		reason := joinReasons(reasons)
		out = append(out, ScoredTask{Task: t, Score: score, Reason: reason})
	}

	// 按 Score 降序
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Score > out[j].Score
	})
	return out
}

// defaultAvailabilitySlots Phase 3 默认可用时间段：9:00-12:00, 14:00-18:00（中午 1 小时午休）。
// Phase 4+ 用户可自定义。
func defaultAvailabilitySlots(date time.Time, now time.Time) []Slot {
	y, m, d := date.Date()

	morningStart := time.Date(y, m, d, 9, 0, 0, 0, date.Location())
	morningEnd := time.Date(y, m, d, 12, 0, 0, 0, date.Location())
	afternoonStart := time.Date(y, m, d, 14, 0, 0, 0, date.Location())
	afternoonEnd := time.Date(y, m, d, 18, 0, 0, 0, date.Location())

	slots := []Slot{}
	// 早于当前时间的 slot 跳过
	if morningEnd.After(now) {
		slotStart := morningStart
		if slotStart.Before(now) {
			slotStart = now
		}
		slots = append(slots, Slot{Start: slotStart, End: morningEnd})
	}
	if afternoonEnd.After(now) {
		slotStart := afternoonStart
		if slotStart.Before(now) {
			slotStart = now
		}
		slots = append(slots, Slot{Start: slotStart, End: afternoonEnd})
	}
	return slots
}

// mergeBlocks 合并已有和新生成的时间块，按 start_at 排序，返回在 slot 范围内的占用段。
func mergeBlocks(slotStart, slotEnd time.Time, existing []*models.ScheduleBlock, newer []*models.ScheduleBlock) []Slot {
	busy := make([]Slot, 0)
	for _, b := range existing {
		start := b.StartAt
		end := b.EndAt
		if end.Before(slotStart) || start.After(slotEnd) {
			continue
		}
		if start.Before(slotStart) {
			start = slotStart
		}
		if end.After(slotEnd) {
			end = slotEnd
		}
		busy = append(busy, Slot{Start: start, End: end})
	}
	for _, b := range newer {
		start := b.StartAt
		end := b.EndAt
		if end.Before(slotStart) || start.After(slotEnd) {
			continue
		}
		if start.Before(slotStart) {
			start = slotStart
		}
		if end.After(slotEnd) {
			end = slotEnd
		}
		busy = append(busy, Slot{Start: start, End: end})
	}
	sort.Slice(busy, func(i, j int) bool {
		return busy[i].Start.Before(busy[j].Start)
	})

	// 合并相邻/重叠
	merged := make([]Slot, 0, len(busy))
	for _, b := range busy {
		if len(merged) == 0 {
			merged = append(merged, b)
			continue
		}
		last := &merged[len(merged)-1]
		if b.Start.Before(last.End) || b.Start.Equal(last.End) {
			if b.End.After(last.End) {
				last.End = b.End
			}
		} else {
			merged = append(merged, b)
		}
	}
	return merged
}

// findNextSlot 在 slot 集合里找下一个能放下 durationMin 分钟的连续时间段。
//
// 跳过已有时间块；如果最早的开始时间晚于 slot 开始，需要从 slot 开始找；
// 找不到返回 nil。
func findNextSlot(durationMin int, slots []Slot, busy []Slot) *Slot {
	if durationMin <= 0 {
		durationMin = 25 // 默认 25 分钟（番茄钟）
	}
	needed := time.Duration(durationMin) * time.Minute

	for _, slot := range slots {
		cursor := slot.Start
		for _, b := range busy {
			if b.End.Before(cursor) {
				continue
			}
			if b.Start.After(slot.End) {
				break
			}
			// cursor 到 b.Start 之间是否够放？
			if cursor.Add(needed).Before(b.Start) || cursor.Add(needed).Equal(b.Start) {
				return &Slot{Start: cursor, End: cursor.Add(needed)}
			}
			cursor = b.End
		}
		// slot 末尾
		if cursor.Add(needed).Before(slot.End) || cursor.Add(needed).Equal(slot.End) {
			return &Slot{Start: cursor, End: cursor.Add(needed)}
		}
	}
	return nil
}

// buildPlan 贪心排程核心算法。
//
// 参数：
//   - tasks：评分后的任务池（已按 Score 降序）
//   - slots：可用时间段（已过滤掉早于 now 的部分）
//   - existing：已有的时间块（避免重复排）
//   - userID：用户 ID
//
// 返回：生成的时间块列表（未写入 DB）。
func buildPlan(tasks []ScoredTask, slots []Slot, existing []models.ScheduleBlock, userID uint64) []*models.ScheduleBlock {
	var newBlocks []*models.ScheduleBlock
	// 转成指针切片方便复用 mergeBlocks
	existingPtrs := make([]*models.ScheduleBlock, len(existing))
	for i := range existing {
		existingPtrs[i] = &existing[i]
	}

	for _, st := range tasks {
		durationMin := st.Task.EstimatedMinutes
		if durationMin <= 0 {
			durationMin = 25
		}
		// 用最新视角的 busy 计算（包含已 newBlocks）
		busy := mergeBlocks(slots[0].Start, slots[len(slots)-1].End, existingPtrs, newBlocks)
		next := findNextSlot(durationMin, slots, busy)
		if next == nil {
			continue
		}

		block := &models.ScheduleBlock{
			UserID:        userID,
			TaskID:        st.Task.ID,
			GoalID:        st.Task.GoalID,
			StartAt:       next.Start,
			EndAt:         next.End,
			PriorityScore: st.Score,
			Reason:        st.Reason,
			Status:        models.ScheduleBlockStatusPending,
		}
		newBlocks = append(newBlocks, block)
	}
	return newBlocks
}

// priorityLabel 把 Priority 数字转成可读字符串。
func priorityLabel(p int8) string {
	switch p {
	case 4:
		return "紧急"
	case 3:
		return "高优"
	case 2:
		return "正常"
	case 1:
		return "低"
	default:
		return "未设"
	}
}

// joinReasons 拼成简短原因字符串。
func joinReasons(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += " + "
		}
		out += p
	}
	if len(out) > 200 {
		out = out[:200]
	}
	return out
}
