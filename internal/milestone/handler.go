package milestone

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

// Create POST /api/v1/goals/:goalID/milestones
func (h *Handler) Create(c *gin.Context) {
	goalID, err := strconv.ParseUint(c.Param("goalID"), 10, 64)
	if err != nil {
		response.Fail(c, response.CodeInvalidParam, "goalID 无效")
		return
	}
	var req CreateMilestoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParam, "参数无效: "+err.Error())
		return
	}
	userID := middleware.MustUserID(c)
	resp, err := h.svc.Create(c.Request.Context(), userID, goalID, &req)
	if err != nil {
		h.translateErr(c, "创建失败", err)
		return
	}
	response.OKWithMessage(c, "创建成功", resp)
}

// List GET /api/v1/goals/:goalID/milestones?page=1&page_size=20
func (h *Handler) List(c *gin.Context) {
	goalID, err := strconv.ParseUint(c.Param("goalID"), 10, 64)
	if err != nil {
		response.Fail(c, response.CodeInvalidParam, "goalID 无效")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	userID := middleware.MustUserID(c)
	resp, err := h.svc.List(c.Request.Context(), userID, goalID, page, pageSize)
	if err != nil {
		h.translateErr(c, "查询失败", err)
		return
	}
	response.OK(c, resp)
}

// Get GET /api/v1/milestones/:id
func (h *Handler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, response.CodeInvalidParam, "id 无效")
		return
	}
	userID := middleware.MustUserID(c)
	resp, err := h.svc.Get(c.Request.Context(), userID, id)
	if err != nil {
		h.translateErr(c, "查询失败", err)
		return
	}
	response.OK(c, resp)
}

// Update PUT /api/v1/milestones/:id
func (h *Handler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, response.CodeInvalidParam, "id 无效")
		return
	}
	var req UpdateMilestoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParam, "参数无效: "+err.Error())
		return
	}
	userID := middleware.MustUserID(c)
	resp, err := h.svc.Update(c.Request.Context(), userID, id, &req)
	if err != nil {
		h.translateErr(c, "更新失败", err)
		return
	}
	response.OKWithMessage(c, "更新成功", resp)
}

// Delete DELETE /api/v1/milestones/:id
func (h *Handler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, response.CodeInvalidParam, "id 无效")
		return
	}
	userID := middleware.MustUserID(c)
	if err := h.svc.Delete(c.Request.Context(), userID, id); err != nil {
		h.translateErr(c, "删除失败", err)
		return
	}
	response.OKWithMessage(c, "删除成功", nil)
}

// translateErr 业务错误翻译为响应码。
func (h *Handler) translateErr(c *gin.Context, fallbackMsg string, err error) {
	switch {
	case errors.Is(err, ErrMilestoneNotFound):
		response.Fail(c, response.CodeMilestoneNotFound, "阶段不存在")
	case errors.Is(err, ErrMilestoneAccessDenied):
		response.Fail(c, response.CodeMilestoneAccessDenied, "无权访问此阶段")
	case errors.Is(err, ErrMilestoneGoalNotFound):
		response.Fail(c, response.CodeMilestoneGoalNotFound, "隶属的目标不存在")
	case errors.Is(err, ErrMilestoneGoalAccessDenied):
		response.Fail(c, response.CodeMilestoneGoalAccessDenied, "无权在此目标下创建阶段")
	default:
		slog.Error(fallbackMsg, "err", err)
		response.Fail(c, response.CodeInternalError, fallbackMsg)
	}
}