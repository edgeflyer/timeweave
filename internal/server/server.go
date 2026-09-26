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
	"timeweave/internal/execution"
	"timeweave/internal/goal"
	"timeweave/internal/middleware"
	"timeweave/internal/milestone"
	"timeweave/internal/schedule"
	"timeweave/internal/task"
	"timeweave/internal/timeprofile"
	"timeweave/internal/timer"
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

	// 前端静态页面（单文件 SPA，零构建）
	engine.StaticFile("/", "./web/index.html")
	engine.StaticFile("/index.html", "./web/index.html")

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

	// Goal 模块（需要鉴权）
	goalRepo := goal.NewRepository(s.db)
	goalSvc := goal.NewService(goalRepo)
	goalHandler := goal.NewHandler(goalSvc)

	// Milestone 模块（需要鉴权）
	milestoneRepo := milestone.NewRepository(s.db)
	milestoneSvc := milestone.NewService(milestoneRepo, goalRepo)
	milestoneHandler := milestone.NewHandler(milestoneSvc)

	// Task 模块（需要鉴权）
	taskRepo := task.NewRepository(s.db)
	taskSvc := task.NewService(taskRepo, milestoneRepo)
	taskHandler := task.NewHandler(taskSvc)

	// Schedule 模块（需要鉴权）
	scheduleRepo := schedule.NewRepository(s.db)
	scheduleSvc := schedule.NewService(scheduleRepo, taskRepo)
	scheduleHandler := schedule.NewHandler(scheduleSvc)

	// Learn 层（timeprofile）：先于 Observe 构造，因为 Observe 通过接口回调 Learn。
	timeprofileRepo := timeprofile.NewRepository(s.db)
	timeprofileSvc := timeprofile.NewService(timeprofileRepo, taskRepo)
	timeprofileHandler := timeprofile.NewHandler(timeprofileSvc)

	// Observe 层（execution）：依赖 taskRepo（TaskAccessor）和 timeprofileSvc（TimeProfileUpdater）。
	executionRepo := execution.NewRepository(s.db)
	executionSvc := execution.NewService(executionRepo, taskRepo, timeprofileSvc)
	executionHandler := execution.NewHandler(executionSvc)

	// Timer 模块（需要鉴权，依赖 task / schedule / execution service）
	//   依赖顺序：execution 必须在 timer 之前装配（timer.complete 会回调 execution.RecordExecution）
	timerRepo := timer.NewRepository(s.db)
	timerSvc := timer.NewService(timerRepo, taskRepo, taskSvc, scheduleSvc, executionSvc)
	timerHandler := timer.NewHandler(timerSvc)

	apiGroup := s.engine.Group("/api/v1")
	apiGroup.Use(middleware.JWTAuth(s.cfg.JWT.Secret))
	{
		apiGroup.POST("/goals", goalHandler.Create)
		apiGroup.GET("/goals", goalHandler.List)
		apiGroup.GET("/goals/:goalID", goalHandler.Get)
		apiGroup.PUT("/goals/:goalID", goalHandler.Update)
		apiGroup.DELETE("/goals/:goalID", goalHandler.Delete)

		// Milestone 嵌套路由
		apiGroup.POST("/goals/:goalID/milestones", milestoneHandler.Create)
		apiGroup.GET("/goals/:goalID/milestones", milestoneHandler.List)
		apiGroup.GET("/milestones/:milestoneID", milestoneHandler.Get)
		apiGroup.PUT("/milestones/:milestoneID", milestoneHandler.Update)
		apiGroup.DELETE("/milestones/:milestoneID", milestoneHandler.Delete)

		// Task 嵌套路由
		apiGroup.POST("/milestones/:milestoneID/tasks", taskHandler.Create)
		apiGroup.GET("/milestones/:milestoneID/tasks", taskHandler.List)
		// Task 顶级路由
		apiGroup.GET("/tasks", taskHandler.ListMine)
		apiGroup.GET("/tasks/:id", taskHandler.Get)
		apiGroup.PUT("/tasks/:id", taskHandler.Update)
		apiGroup.PATCH("/tasks/:id/status", taskHandler.UpdateStatus)
		apiGroup.DELETE("/tasks/:id", taskHandler.Delete)

		// Schedule（Scheduler 排程）
		apiGroup.POST("/schedule/plan", scheduleHandler.Plan)
		apiGroup.POST("/schedule/replan", scheduleHandler.Replan)
		apiGroup.GET("/schedule", scheduleHandler.List)
		apiGroup.DELETE("/schedule/:id", scheduleHandler.Delete)

		// Timer（计时）
		apiGroup.POST("/timer/start", timerHandler.Start)
		apiGroup.POST("/timer/pause", timerHandler.Pause)
		apiGroup.POST("/timer/resume", timerHandler.Resume)
		apiGroup.POST("/timer/complete", timerHandler.Complete)
		apiGroup.POST("/timer/abandon", timerHandler.Abandon)
		apiGroup.GET("/timer/active", timerHandler.Active)
		apiGroup.GET("/timer/history", timerHandler.History)

		// Learn 端（UserTimeProfile）—— Sprint 5
		apiGroup.GET("/user/time-profiles", timeprofileHandler.List)
		apiGroup.GET("/user/time-profiles/:task_type", timeprofileHandler.Get)
		apiGroup.DELETE("/user/time-profiles/:task_type", timeprofileHandler.Reset)

		// Observe 端（TaskExecution 历史）—— Sprint 5
		apiGroup.GET("/executions", executionHandler.List)
		apiGroup.GET("/executions/:id", executionHandler.Get)
	}
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