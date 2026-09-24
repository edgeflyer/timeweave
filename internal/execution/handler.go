package execution

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"timeweave/internal/middleware"
	"timeweave/internal/models"
	"timeweave/internal/response"
)

// Handler HTTP 处理器。
//
// execution 的写入完全由 Timer.complete / Timer.abandon 触发，
// 不暴露 HTTP 写入端点。这里只暴露「查询最近执行历史」给前端
// dashboard / 调试用。
type Handler struct {
	svc *Service
}

// NewHandler 构造处理器。
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// List GET /api/v1/executions?task_type=...&page=1&page_size=20
//
// 列当前用户的执行历史（按 id DESC）。task_type 可选过滤。
func (h *Handler) List(c *gin.Context) {
	userID := middleware.MustUserID(c)
	taskType := c.Query("task_type")
	if taskType != "" && !models.IsValidTaskType(models.TaskType(taskType)) {
		response.Fail(c, response.CodeTimeProfileInvalidType, "非法的 task_type")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	resp, err := h.svc.ListByUser(c.Request.Context(), userID, taskType, page, pageSize)
	if err != nil {
		h.translateErr(c, "查询执行历史失败", err)
		return
	}
	response.OK(c, resp)
}

// Get GET /api/v1/executions/:id
//
// 单条执行记录详情（debug / 时间轴视图用）。
func (h *Handler) Get(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.Fail(c, response.CodeInvalidParam, "id 无效")
		return
	}
	userID := middleware.MustUserID(c)
	exec, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		h.translateErr(c, "查询执行记录失败", err)
		return
	}
	// 权限校验：只能看自己的
	if exec.UserID != userID {
		c.AbortWithStatusJSON(http.StatusForbidden, response.Body{
			Code:    response.CodeExecutionTaskAccessDenied,
			Message: "无权访问此执行记录",
			Data:    nil,
		})
		return
	}
	response.OK(c, exec)
}

// translateErr 业务错误翻译为响应码。
func (h *Handler) translateErr(c *gin.Context, fallbackMsg string, err error) {
	switch {
	case errors.Is(err, ErrExecutionNotFound):
		response.Fail(c, response.CodeExecutionNotFound, "执行记录不存在")
	case errors.Is(err, ErrExecutionTaskNotFound):
		response.Fail(c, response.CodeExecutionTaskNotFound, "关联的任务不存在")
	case errors.Is(err, ErrExecutionTaskAccessDenied):
		response.Fail(c, response.CodeExecutionTaskAccessDenied, "无权访问关联的任务")
	default:
		slog.Error(fallbackMsg, "err", err)
		response.Fail(c, response.CodeInternalError, fallbackMsg)
	}
}