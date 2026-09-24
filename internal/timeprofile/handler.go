package timeprofile

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"timeweave/internal/middleware"
	"timeweave/internal/models"
	"timeweave/internal/response"
)

// Handler HTTP 处理器。
type Handler struct {
	svc *Service
}

// NewHandler 构造处理器。
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// List GET /api/v1/user/time-profiles
//
// 返回当前用户的所有 task_type 画像。
// 列表而非单端点：用户前端可以一次性看到自己的「时间学习画像」全景。
func (h *Handler) List(c *gin.Context) {
	userID := middleware.MustUserID(c)
	resp, err := h.svc.List(c.Request.Context(), userID)
	if err != nil {
		h.translateErr(c, "查询画像失败", err)
		return
	}
	response.OK(c, resp)
}

// Get GET /api/v1/user/time-profiles/:task_type
//
// 返回单个 task_type 的画像详情。
// 路径参数 task_type 必须合法（coding/reading/.../other）。
func (h *Handler) Get(c *gin.Context) {
	userID := middleware.MustUserID(c)
	taskType := c.Param("task_type")
	if !models.IsValidTaskType(models.TaskType(taskType)) {
		response.Fail(c, response.CodeTimeProfileInvalidType, "非法的 task_type")
		return
	}
	resp, err := h.svc.Get(c.Request.Context(), userID, models.TaskType(taskType))
	if err != nil {
		h.translateErr(c, "查询画像失败", err)
		return
	}
	response.OK(c, resp)
}

// Reset DELETE /api/v1/user/time-profiles/:task_type
//
// 删除指定 task_type 的画像（"重新教我"）。
// 删除后下次同类任务完成时会按全新画像重建（initial = 当前 Task 的 estimated_minutes）。
func (h *Handler) Reset(c *gin.Context) {
	userID := middleware.MustUserID(c)
	taskType := c.Param("task_type")
	if !models.IsValidTaskType(models.TaskType(taskType)) {
		response.Fail(c, response.CodeTimeProfileInvalidType, "非法的 task_type")
		return
	}
	if err := h.svc.Reset(c.Request.Context(), userID, models.TaskType(taskType)); err != nil {
		h.translateErr(c, "重置画像失败", err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ListExecutions 暴露在另一个模块（execution.Handler），不在这里重复声明。
// 此 handler 文件仅承担「画像本身」的读写。

// translateErr 业务错误翻译为响应码。
func (h *Handler) translateErr(c *gin.Context, fallbackMsg string, err error) {
	switch {
	case errors.Is(err, ErrTimeProfileNotFound):
		response.Fail(c, response.CodeTimeProfileNotFound, "画像不存在")
	case errors.Is(err, ErrTimeProfileInvalidType):
		response.Fail(c, response.CodeTimeProfileInvalidType, "非法的 task_type")
	case errors.Is(err, ErrTimeProfileInvalidAgentValue):
		response.Fail(c, response.CodeInvalidParam, "Agent 给出的 adjusted 值非法")
	default:
		slog.Error(fallbackMsg, "err", err)
		response.Fail(c, response.CodeInternalError, fallbackMsg)
	}
}