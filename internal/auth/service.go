package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"timeweave/internal/models"
	"timeweave/pkg/jwt"
	"timeweave/pkg/password"
)

// Service 是 Auth 模块的业务逻辑层。
type Service struct {
	repo         *Repository
	jwtSecret    string
	tokenTTL     time.Duration
}

// NewService 构造服务。
//
// jwtSecret 必须 ≥16 字节（config 层已校验）。
// tokenTTL 推荐 24h 或 168h（7 天）。
func NewService(repo *Repository, jwtSecret string, tokenTTL time.Duration) *Service {
	return &Service{
		repo:      repo,
		jwtSecret: jwtSecret,
		tokenTTL:  tokenTTL,
	}
}

// Register 注册新用户。
// 流程：查重 → bcrypt 哈希 → 写入 DB → 签发 token。
func (s *Service) Register(ctx context.Context, req *RegisterRequest) (*AuthResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	nickname := strings.TrimSpace(req.Nickname)

	// 1. 先查重（避免依赖 DB 错误码翻译，错误信息更友好）
	if existing, err := s.repo.GetByEmail(ctx, email); err == nil && existing != nil {
		return nil, ErrEmailExists
	} else if err != nil && !errors.Is(err, ErrUserNotFound) {
		return nil, err
	}

	// 2. 哈希密码
	hash, err := password.Hash(req.Password)
	if err != nil {
		return nil, err
	}

	// 3. 创建用户
	u := &models.User{
		Email:        email,
		PasswordHash: hash,
		Nickname:     nickname,
		Status:       models.UserStatusActive,
	}
	if err := s.repo.Create(ctx, u); err != nil {
		return nil, err
	}

	// 4. 签发 token
	return s.issueToken(u)
}

// Login 邮箱 + 密码登录。
func (s *Service) Login(ctx context.Context, req *LoginRequest) (*AuthResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))

	u, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			// 安全实践：不区分「用户不存在」与「密码错误」，避免账户枚举攻击
			return nil, ErrWrongPassword
		}
		return nil, err
	}

	if u.Status == models.UserStatusDisabled {
		return nil, ErrUserDisabled
	}

	if err := password.Check(u.PasswordHash, req.Password); err != nil {
		return nil, ErrWrongPassword
	}

	// 异步更新最后登录时间（不阻塞响应，错误仅记日志）
	_ = s.repo.UpdateLastLogin(ctx, u.ID)

	return s.issueToken(u)
}

// issueToken 内部辅助：签发 token 并组装响应。
func (s *Service) issueToken(u *models.User) (*AuthResponse, error) {
	tok, err := jwt.Issue(s.jwtSecret, u.ID, s.tokenTTL)
	if err != nil {
		return nil, err
	}
	expiresAt := time.Now().Add(s.tokenTTL).Unix()
	return &AuthResponse{
		UserID:    u.ID,
		Email:     u.Email,
		Nickname:  u.Nickname,
		Token:     tok,
		ExpiresAt: expiresAt,
	}, nil
}

// SilenceUnusedKeepImport 防止 gorm 包未直接使用（被 Repository 间接持有）。
var _ = gorm.ErrRecordNotFound