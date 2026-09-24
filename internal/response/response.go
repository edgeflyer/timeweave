// Package response 提供统一的 HTTP 响应包装。
//
// 所有 API 返回的 JSON 格式固定为：
//
//	{
//	  "code":    <int>,         // 业务状态码，0 表示成功
//	  "message": <string>,      // 给前端展示的信息
//	  "data":    <any>          // 业务数据，失败时为 null
//	}
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Code 定义业务错误码区间：
//   - 0: 成功
//   - 1xxx: 通用错误（参数、认证、资源）
//   - 2xxx: Auth 模块
//   - 9xxx: 系统错误（数据库、内部异常）
const (
	CodeOK             = 0
	CodeInvalidParam   = 1001
	CodeUnauthorized   = 1002
	CodeNotFound       = 1003
	CodeAlreadyExists  = 1004
	CodeInternalError  = 9001
	CodeDBError        = 9002
)

// Auth 模块专用错误码（2xxx）
const (
	CodeEmailExists     = 2001 // 注册时邮箱已被占用
	CodeUserNotFound    = 2002 // 登录时邮箱不存在
	CodeWrongPassword   = 2003 // 登录时密码错误
	CodeUserDisabled    = 2004 // 用户被禁用
	CodeInvalidToken    = 2005 // JWT 无效或过期
)

// Goal 模块专用错误码（3xxx）
const (
	CodeGoalNotFound     = 3001 // 目标不存在
	CodeGoalAccessDenied = 3002 // 无权访问此目标（非所属用户）
)

// Milestone 模块专用错误码（4xxx）
const (
	CodeMilestoneNotFound         = 4001 // 阶段不存在
	CodeMilestoneAccessDenied     = 4002 // 无权访问此阶段
	CodeMilestoneGoalNotFound     = 4003 // 隶属的目标不存在
	CodeMilestoneGoalAccessDenied = 4004 // 无权在此目标下创建阶段
)

// Task 模块专用错误码（5xxx）
const (
	CodeTaskNotFound                  = 5001
	CodeTaskAccessDenied              = 5002
	CodeTaskMilestoneNotFound         = 5003
	CodeTaskMilestoneAccessDenied     = 5004
	CodeTaskInvalidStatusTransition   = 5005
	CodeTaskAlreadyDone               = 5006
)

// Schedule 模块专用错误码（6xxx）
const (
	CodeScheduleBlockNotFound     = 6001
	CodeScheduleBlockAccessDenied = 6002
	CodeScheduleTimeConflict      = 6003
	CodeScheduleNoTasks           = 6004
	CodeScheduleInsufficientTime  = 6005
)

// Timer 模块专用错误码（7xxx）
const (
	CodeTimerSessionNotFound     = 7001
	CodeTimerSessionAccessDenied = 7002
	CodeTimerInvalidTransition   = 7003
	CodeTimerTaskNotFound        = 7004
	CodeTimerTaskAccessDenied    = 7005
	CodeTimerTaskNotActive       = 7006
)

// Execution 模块专用错误码（8xxx）—— Observe 层：Timer complete 落库 TaskExecution。
const (
	CodeExecutionNotFound         = 8001 // TaskExecution 不存在
	CodeExecutionTaskNotFound     = 8002 // 关联的 Task 不存在
	CodeExecutionTaskAccessDenied = 8003 // 无权访问关联的 Task
)

// TimeProfile 模块专用错误码（9xxx）—— Learn 层：UserTimeProfile 画像管理。
//
// 注：9001/9002 已被通用错误码占用，本模块从 9003 开始。
const (
	CodeTimeProfileNotFound    = 9003 // 某 task_type 的画像不存在
	CodeTimeProfileInvalidType = 9004 // task_type 枚举值非法
)

// Body 是统一响应体。
type Body struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// OK 写入 200 + 业务成功响应。
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Body{
		Code:    CodeOK,
		Message: "ok",
		Data:    data,
	})
}

// OKWithMessage 成功响应，自定义消息。
func OKWithMessage(c *gin.Context, msg string, data interface{}) {
	c.JSON(http.StatusOK, Body{
		Code:    CodeOK,
		Message: msg,
		Data:    data,
	})
}

// Fail 写入错误响应，HTTP 状态码固定为 200（业务失败由 code 区分）。
//
// 原因：HTTP 层只关心网络层错误（4xx/5xx），业务失败统一 200 + 非零 code，
// 这样前端只需判断 res.data.code === "1" 即可。
func Fail(c *gin.Context, code int, msg string) {
	c.JSON(http.StatusOK, Body{
		Code:    code,
		Message: msg,
		Data:    nil,
	})
}

// FailWithData 错误响应，附带错误数据（如字段级校验错误）。
func FailWithData(c *gin.Context, code int, msg string, data interface{}) {
	c.JSON(http.StatusOK, Body{
		Code:    code,
		Message: msg,
		Data:    data,
	})
}

// AbortWithFail 终止后续 handler，返回错误响应（用于 middleware）。
func AbortWithFail(c *gin.Context, code int, msg string) {
	AbortWithStatus(c, http.StatusUnauthorized, code, msg)
}

// AbortWithStatus 终止 + 自定义 HTTP 状态码。
func AbortWithStatus(c *gin.Context, status, code int, msg string) {
	c.AbortWithStatusJSON(status, Body{
		Code:    code,
		Message: msg,
		Data:    nil,
	})
}