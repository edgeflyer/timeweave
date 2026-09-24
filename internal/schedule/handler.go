package schedule

import (
	"errors"
	"log/slog"
	"strconv"
	"time"

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

// Plan POST /api/v1/schedule/plan
func (h *Handler) Plan(c *gin.Context) {
	var req PlanRequest
	// body 可空（不传则给今日排程）
	_ = c.ShouldBindJSON(&req)

	planDate, err := parseDateOrToday(req.Date)
	if err != nil {
		response.Fail(c, response.CodeInvalidParam, err.Error())
		return
	}

	userID := middleware.MustUserID(c)
	resp, err := h.svc.Plan(c.Request.Context(), userID, planDate, req.ReplaceExisting)
	if err != nil {
		h.translateErr(c, "排程失败", err)
		return
	}
	response.OKWithMessage(c, "排程完成", resp)
}

// Replan POST /api/v1/schedule/replan
func (h *Handler) Replan(c *gin.Context) {
	var req ReplanRequest
	_ = c.ShouldBindJSON(&req)

	date, err := parseDateOrToday(req.Date)
	if err != nil {
		response.Fail(c, response.CodeInvalidParam, err.Error())
		return
	}

	userID := middleware.MustUserID(c)
	resp, err := h.svc.Replan(c.Request.Context(), userID, date)
	if err != nil {
		h.translateErr(c, "重排失败", err)
		return
	}
	response.OKWithMessage(c, "重排完成", resp)
}

// List GET /api/v1/schedule?date=YYYY-MM-DD
func (h *Handler) List(c *gin.Context) {
	dateStr := c.DefaultQuery("date", "")
	date, err := parseDateOrToday(dateStr)
	if err != nil {
		response.Fail(c, response.CodeInvalidParam, err.Error())
		return
	}

	userID := middleware.MustUserID(c)
	resp, err := h.svc.ListByDate(c.Request.Context(), userID, date)
	if err != nil {
		h.translateErr(c, "查询失败", err)
		return
	}
	response.OK(c, resp)
}

// Delete DELETE /api/v1/schedule/:id
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
	case errors.Is(err, ErrScheduleBlockNotFound):
		response.Fail(c, response.CodeScheduleBlockNotFound, "时间块不存在")
	case errors.Is(err, ErrScheduleBlockAccessDenied):
		response.Fail(c, response.CodeScheduleBlockAccessDenied, "无权访问此时间块")
	case errors.Is(err, ErrScheduleTimeConflict):
		response.Fail(c, response.CodeScheduleTimeConflict, "时间段冲突")
	case errors.Is(err, ErrScheduleInvalidTimeRange):
		response.Fail(c, response.CodeInvalidParam, "时间范围无效")
	case errors.Is(err, ErrScheduleNoTasksToPlan):
		response.Fail(c, response.CodeScheduleNoTasks, "当前没有可排程的任务")
	case errors.Is(err, ErrScheduleInsufficientTime):
		response.Fail(c, response.CodeScheduleInsufficientTime, "可用时间不够，无法排程")
	default:
		slog.Error(fallbackMsg, "err", err)
		response.Fail(c, response.CodeInternalError, fallbackMsg)
	}
}

// parseDateOrToday 解析日期字符串，空则今天，格式错误则报错。
func parseDateOrToday(s string) (time.Time, error) {
	if s == "" {
		return time.Now(), nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, errors.New("日期格式无效，应为 YYYY-MM-DD")
	}
	return t, nil
}
