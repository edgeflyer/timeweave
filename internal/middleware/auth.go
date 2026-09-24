// Package middleware 存放所有 gin middleware。
package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"timeweave/internal/response"
	"timeweave/pkg/jwt"
)

// ContextKeyUserID 是注入到 gin.Context 的 user_id key。
// handler 通过 c.GetUint64(ContextKeyUserID) 拿到登录用户 ID。
const ContextKeyUserID = "user_id"

// JWTAuth 解析 Authorization: Bearer <token>，验证通过后注入 user_id。
//
// 用法：
//
//	api := r.Group("/api", middleware.JWTAuth(secret))
//
// token 来源：Authorization 头。也可以从 query (?token=) 兜底（仅调试用）。
func JWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := extractToken(c)
		if raw == "" {
			response.AbortWithFail(c, response.CodeUnauthorized, "未提供 token")
			return
		}

		uid, err := jwt.Parse(secret, raw)
		if err != nil {
			switch {
			case strings.Contains(err.Error(), "expired"):
				response.AbortWithFail(c, response.CodeInvalidToken, "token 已过期")
			default:
				response.AbortWithFail(c, response.CodeInvalidToken, "token 无效")
			}
			return
		}

		c.Set(ContextKeyUserID, uid)
		c.Next()
	}
}

// extractToken 从 header 或 query 中抽取 token。
func extractToken(c *gin.Context) string {
	// 1. Authorization: Bearer <token>
	h := c.GetHeader("Authorization")
	if h != "" {
		parts := strings.SplitN(h, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return strings.TrimSpace(parts[1])
		}
	}
	// 2. 兜底：query ?token=xxx（仅供调试）
	return c.Query("token")
}

// MustUserID 从 context 拿 user_id，类型安全便捷函数。
func MustUserID(c *gin.Context) uint64 {
	v, ok := c.Get(ContextKeyUserID)
	if !ok {
		return 0
	}
	id, _ := v.(uint64)
	return id
}