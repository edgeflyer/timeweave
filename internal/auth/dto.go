package auth

// RegisterRequest 注册请求体。
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email,max=191"`
	Password string `json:"password" binding:"required,min=8,max=64"`
	Nickname string `json:"nickname" binding:"required,min=1,max=64"`
}

// LoginRequest 登录请求体。
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=1"`
}

// AuthResponse 注册/登录成功响应。
type AuthResponse struct {
	UserID    uint64 `json:"user_id"`
	Email     string `json:"email"`
	Nickname  string `json:"nickname"`
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"` // Unix 时间戳，秒
}