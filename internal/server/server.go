package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"timeweave/internal/auth"
	"timeweave/internal/config"
	"timeweave/internal/middleware"
)

// Server 封装 HTTP 服务。
type Server struct {
	cfg     *config.Config
	db      *gorm.DB
	engine  *gin.Engine
	httpSrv *http.Server
}

// New 构造 Server（不启动）。
func New(cfg *config.Config, db *gorm.DB) *Server {
	// 生产模式关闭 debug 日志
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(middleware.Logger())

	// 健康检查
	engine.GET("/healthz", func(c *gin.Context) {
		if err := middleware.PingDB(db); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	s := &Server{
		cfg:    cfg,
		db:     db,
		engine: engine,
	}

	s.registerRoutes()

	s.httpSrv = &http.Server{
		Addr:              ":" + cfg.AppPort,
		Handler:           engine,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	return s
}

// registerRoutes 注册所有业务路由。
func (s *Server) registerRoutes() {
	// Auth 模块（不需要鉴权）
	authRepo := auth.NewRepository(s.db)
	authSvc := auth.NewService(authRepo, s.cfg.JWT.Secret, s.cfg.JWT.TTL)
	authHandler := auth.NewHandler(authSvc)
	authGroup := s.engine.Group("/auth")
	{
		authGroup.POST("/register", authHandler.Register)
		authGroup.POST("/login", authHandler.Login)
	}

	// 后续需要鉴权的路由示例（Phase 1 占位）
	// apiGroup := s.engine.Group("/api/v1")
	// apiGroup.Use(middleware.JWTAuth(s.cfg.JWT.Secret))
	// {
	//     apiGroup.GET("/me", func(c *gin.Context) {
	//         userID := middleware.MustUserID(c)
	//         c.JSON(200, gin.H{"user_id": userID})
	//     })
	// }
}

// Run 启动 HTTP 服务，阻塞直到收到退出信号。
func (s *Server) Run() error {
	slog.Info("http server listening", "addr", s.cfg.AppPort)
	if err := s.httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Shutdown 优雅退出。
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpSrv.Shutdown(ctx)
}

// Engine 返回 gin.Engine（用于测试）。
func (s *Server) Engine() *gin.Engine {
	return s.engine
}