package timer

import (
	"errors"
	"log/slog"
	"strconv"

	"github.com/gin-gonic/gin"

	"timeweave/internal/middleware"
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

// Start POST /api/v1/timer/start
func (h *Handler) Start(c *gin.Context) {
	var req StartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParam, "参数无效: "+err.Error())
		return
	}
	userID := middleware.MustUserID(c)
	resp, err := h.svc.Start(c.Request.Context(), userID, &req)
	if err != nil {
		h.translateErr(c, "开始计时失败", err)
		return
	}
	response.OKWithMessage(c, "开始计时", resp)
}

// Pause POST /api/v1/timer/pause
func (h *Handler) Pause(c *gin.Context) {
	var req SessionIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParam, "参数无效: "+err.Error())
		return
	}
	userID := middleware.MustUserID(c)
	resp, err := h.svc.Pause(c.Request.Context(), userID, req.SessionID)
	if err != nil {
		h.translateErr(c, "暂停失败", err)
		return
	}
	response.OKWithMessage(c, "已暂停", resp)
}

// Resume POST /api/v1/timer/resume
func (h *Handler) Resume(c *gin.Context) {
	var req SessionIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParam, "参数无效: "+err.Error())
		return
	}
	userID := middleware.MustUserID(c)
	resp, err := h.svc.Resume(c.Request.Context(), userID, req.SessionID)
	if err != nil {
		h.translateErr(c, "恢复失败", err)
		return
	}
	response.OKWithMessage(c, "已恢复", resp)
}

// Complete POST /api/v1/timer/complete
func (h *Handler) Complete(c *gin.Context) {
	var req SessionIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParam, "参数无效: "+err.Error())
		return
	}
	userID := middleware.MustUserID(c)
	resp, err := h.svc.Complete(c.Request.Context(), userID, req.SessionID)
	if err != nil {
		h.translateErr(c, "完成失败", err)
		return
	}
	response.OKWithMessage(c, "已完成", resp)
}

// Abandon POST /api/v1/timer/abandon
func (h *Handler) Abandon(c *gin.Context) {
	var req AbandonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParam, "参数无效: "+err.Error())
		return
	}
	userID := middleware.MustUserID(c)
	resp, err := h.svc.Abandon(c.Request.Context(), userID, req.SessionID)
	if err != nil {
		h.translateErr(c, "放弃失败", err)
		return
	}
	response.OKWithMessage(c, "已放弃", resp)
}

// Active GET /api/v1/timer/active
func (h *Handler) Active(c *gin.Context) {
	userID := middleware.MustUserID(c)
	resp, err := h.svc.Active(c.Request.Context(), userID)
	if err != nil {
		h.translateErr(c, "查询失败", err)
		return
	}
	if resp == nil {
		response.OKWithMessage(c, "当前无活跃计时", nil)
		return
	}
	response.OK(c, resp)
}

// History GET /api/v1/timer/history?limit=20
func (h *Handler) History(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	userID := middleware.MustUserID(c)
	resp, err := h.svc.History(c.Request.Context(), userID, limit)
	if err != nil {
		h.translateErr(c, "查询失败", err)
		return
	}
	response.OK(c, resp)
}

// translateErr 业务错误翻译为响应码。
func (h *Handler) translateErr(c *gin.Context, fallbackMsg string, err error) {
	switch {
	case errors.Is(err, ErrTimerSessionNotFound):
		response.Fail(c, response.CodeTimerSessionNotFound, "计时会话不存在")
	case errors.Is(err, ErrTimerSessionAccessDenied):
		response.Fail(c, response.CodeTimerSessionAccessDenied, "无权访问此计时会话")
	case errors.Is(err, ErrTimerInvalidTransition):
		response.Fail(c, response.CodeTimerInvalidTransition, "计时会话状态机非法跃迁")
	case errors.Is(err, ErrTimerTaskNotFound):
		response.Fail(c, response.CodeTimerTaskNotFound, "关联的任务不存在")
	case errors.Is(err, ErrTimerTaskAccessDenied):
		response.Fail(c, response.CodeTimerTaskAccessDenied, "无权访问关联的任务")
	case errors.Is(err, ErrTimerTaskNotActive):
		response.Fail(c, response.CodeTimerTaskNotActive, "该任务已结束，无法计时")
	default:
		slog.Error(fallbackMsg, "err", err)
		response.Fail(c, response.CodeInternalError, fallbackMsg)
	}
}
