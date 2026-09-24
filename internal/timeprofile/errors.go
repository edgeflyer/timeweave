// Package timeprofile 是 QuestOS 闭环里的 Learn 层：
//
//   - 维护 user_time_profiles 表（按 user × task_type 聚合耗时画像）
//   - 每次 TaskExecution 完成 → execution.Service 回调 UpdateProfile → 算法更新画像
//   - 提供 HTTP 接口让用户查看画像 + 手动 reset
//   - 预留 AdjustByAgent 内部方法给 Phase 5 的 Review Agent 调用
//
// 关键算法（详见 service.go）：
//   - rolling_avg = (old_avg × old_count + actual) / (old_count + 1)  简单滚动平均
//   - sample_count < 10：adjusted = 0.7 × initial + 0.3 × rolling（冷启动加权）
//   - sample_count >= 10：adjusted = rolling（热态纯均值）
//   - 偏差 > 10% 时才回写到同类未完成 Task，避免抖动
package timeprofile

import "errors"

// 业务错误，service 层返回，handler 层翻译为响应码。
var (
	// ErrTimeProfileNotFound 指定 task_type 的画像不存在。
	ErrTimeProfileNotFound = errors.New("timeprofile: not found")
	// ErrTimeProfileInvalidType task_type 枚举值非法。
	ErrTimeProfileInvalidType = errors.New("timeprofile: invalid task_type")
	// ErrTimeProfileInvalidAgentValue Agent 给出的 adjusted 值非法（< 0）。
	ErrTimeProfileInvalidAgentValue = errors.New("timeprofile: invalid agent value")
)