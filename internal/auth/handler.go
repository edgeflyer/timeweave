package auth

import (
	"errors"
	"log/slog"
	"strings"

	"github.com/gin-gonic/gin"

	"timeweave/internal/response"
)

// Handler 暴露 HTTP 接口。
type Handler struct {
	svc *Service
}

// NewHandler 构造处理器。
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Register POST /auth/register
func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParam, "参数无效: "+err.Error())
		return
	}

	resp, err := h.svc.Register(c.Request.Context(), &req)
	if err != nil {
		switch {
		case errors.Is(err, ErrEmailExists):
			response.Fail(c, response.CodeEmailExists, "邮箱已被注册")
		default:
			slog.Error("register failed", "err", err)
			response.Fail(c, response.CodeInternalError, "注册失败")
		}
		return
	}

	response.OKWithMessage(c, "注册成功", resp)
}

// Login POST /auth/login
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParam, "参数无效: "+err.Error())
		return
	}

	resp, err := h.svc.Login(c.Request.Context(), &req)
	if err != nil {
		switch {
		case errors.Is(err, ErrWrongPassword):
			// 不区分「用户不存在」与「密码错误」，统一模糊返回
			response.Fail(c, response.CodeWrongPassword, "邮箱或密码错误")
		case errors.Is(err, ErrUserDisabled):
			response.Fail(c, response.CodeUserDisabled, "账号已被禁用")
		default:
			slog.Error("login failed", "err", err)
			response.Fail(c, response.CodeInternalError, "登录失败")
		}
		return
	}

	response.OKWithMessage(c, "登录成功", resp)
}

// TrimUsername 公共小工具：trim 前后空格 + 大小写归一。
// 暂未直接使用，留作后续 OAuth/SSO 接入时复用。
func TrimUsername(s string) string {
	return strings.TrimSpace(s)
}