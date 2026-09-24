package task

import (
	"errors"
	"log/slog"
	"strconv"

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

// Create POST /api/v1/milestones/:milestoneID/tasks
func (h *Handler) Create(c *gin.Context) {
	milestoneID, err := strconv.ParseUint(c.Param("milestoneID"), 10, 64)
	if err != nil {
		response.Fail(c, response.CodeInvalidParam, "milestoneID 无效")
		return
	}
	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParam, "参数无效: "+err.Error())
		return
	}
	userID := middleware.MustUserID(c)
	resp, err := h.svc.Create(c.Request.Context(), userID, milestoneID, &req)
	if err != nil {
		h.translateErr(c, "创建失败", err)
		return
	}
	response.OKWithMessage(c, "创建成功", resp)
}

// List GET /api/v1/milestones/:milestoneID/tasks?page=1&page_size=20
func (h *Handler) List(c *gin.Context) {
	milestoneID, err := strconv.ParseUint(c.Param("milestoneID"), 10, 64)
	if err != nil {
		response.Fail(c, response.CodeInvalidParam, "milestoneID 无效")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	userID := middleware.MustUserID(c)
	resp, err := h.svc.List(c.Request.Context(), userID, milestoneID, page, pageSize)
	if err != nil {
		h.translateErr(c, "查询失败", err)
		return
	}
	response.OK(c, resp)
}

// ListMine GET /api/v1/tasks?status=1&page=1&page_size=20
func (h *Handler) ListMine(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	var status *int8
	if s := c.Query("status"); s != "" {
		v, err := strconv.ParseInt(s, 10, 8)
		if err != nil || v < 0 || v > 4 {
			response.Fail(c, response.CodeInvalidParam, "status 必须在 0-4 之间")
			return
		}
		vv := int8(v)
		status = &vv
	}
	userID := middleware.MustUserID(c)
	resp, err := h.svc.ListByUser(c.Request.Context(), userID, status, page, pageSize)
	if err != nil {
		h.translateErr(c, "查询失败", err)
		return
	}
	response.OK(c, resp)
}

// Get GET /api/v1/tasks/:id
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

// Update PUT /api/v1/tasks/:id
func (h *Handler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, response.CodeInvalidParam, "id 无效")
		return
	}
	var req UpdateTaskRequest
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

// UpdateStatus PATCH /api/v1/tasks/:id/status
func (h *Handler) UpdateStatus(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, response.CodeInvalidParam, "id 无效")
		return
	}
	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParam, "参数无效: "+err.Error())
		return
	}
	userID := middleware.MustUserID(c)
	resp, err := h.svc.UpdateStatus(c.Request.Context(), userID, id, models.TaskStatus(req.Status))
	if err != nil {
		h.translateErr(c, "状态切换失败", err)
		return
	}
	response.OKWithMessage(c, "状态切换成功", resp)
}

// Delete DELETE /api/v1/tasks/:id
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
	case errors.Is(err, ErrTaskNotFound):
		response.Fail(c, response.CodeTaskNotFound, "任务不存在")
	case errors.Is(err, ErrTaskAccessDenied):
		response.Fail(c, response.CodeTaskAccessDenied, "无权访问此任务")
	case errors.Is(err, ErrTaskMilestoneNotFound):
		response.Fail(c, response.CodeTaskMilestoneNotFound, "隶属的阶段不存在")
	case errors.Is(err, ErrTaskMilestoneAccessDenied):
		response.Fail(c, response.CodeTaskMilestoneAccessDenied, "无权在此阶段下创建任务")
	case errors.Is(err, ErrTaskInvalidStatusTransition):
		response.Fail(c, response.CodeTaskInvalidStatusTransition, "状态机非法跃迁")
	case errors.Is(err, ErrTaskAlreadyDone):
		response.Fail(c, response.CodeTaskAlreadyDone, "已完成的任务不能再切换状态")
	default:
		slog.Error(fallbackMsg, "err", err)
		response.Fail(c, response.CodeInternalError, fallbackMsg)
	}
}
