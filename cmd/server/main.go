package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"timeweave/internal/config"
	"timeweave/internal/db"
	"timeweave/internal/models"
	"timeweave/internal/server"
)

func main() {
	// 1. 加载配置
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config failed", "err", err)
		os.Exit(1)
	}

	// 2. 初始化全局 logger
	initLogger(cfg.LogLevel)

	slog.Info("starting timeweave server",
		"env", cfg.AppEnv,
		"addr", ":"+cfg.AppPort,
		"db", cfg.DB.Host,
	)

	// 3. 连接数据库
	gormDB, err := db.Open(cfg)
	if err != nil {
		slog.Error("connect db failed", "err", err)
		os.Exit(1)
	}
	defer func() {
		_ = db.Close(gormDB)
	}()

	// 4. 自动迁移（Phase 1 简化版，正式项目建议用 atlas / golang-migrate）
	if err := gormDB.AutoMigrate(&models.User{}); err != nil {
		slog.Error("automigrate failed", "err", err)
		os.Exit(1)
	}
	slog.Info("automigrate done")

	// 5. 构造 HTTP 服务
	srv := server.New(cfg, gormDB)

	// 6. 启动服务（goroutine）
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- srv.Run()
	}()

	// 7. 等待退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		slog.Info("received signal, shutting down", "sig", sig.String())
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error, shutting down", "err", err)
		}
	}

	// 8. 优雅退出（10s 超时）
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown failed", "err", err)
		os.Exit(1)
	}
	slog.Info("server stopped gracefully")
}

// initLogger 根据 slog.Level 初始化全局 logger。
func initLogger(lvl slog.Level) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
	slog.SetDefault(logger)
}