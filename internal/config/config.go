// Package config 负责从环境变量（或本地 .env 文件）加载整个应用的配置。
//
// 加载策略：
//   - 优先读取进程环境变量
//   - 进程里没有的字段，再退回到本地 .env 文件
//   - .env 文件不存在不报错（生产环境通常纯靠环境变量启动）
//
// 用法：
//
//	cfg, err := config.Load()
//	if err != nil { ... }
//	fmt.Println(cfg.AppPort, cfg.DSN())
package config

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/joho/godotenv"
)

// Config 是整个应用的全局配置。所有字段都从环境变量加载，不在代码里写死。
type Config struct {
	AppEnv   string     // development | staging | production
	AppPort  string     // HTTP 监听端口，例如 "8080"
	LogLevel slog.Level // slog 日志级别

	DB  DBConfig  // 数据库配置
	JWT JWTConfig // JWT 鉴权配置
}

// DBConfig 是 MySQL 连接所需的所有字段。
type DBConfig struct {
	Host     string // 127.0.0.1
	Port     string // 3306
	User     string // root
	Password string
	Name     string // 数据库名，例如 questos
}

// JWTConfig 是 JWT 签发 / 解析所需的字段。
type JWTConfig struct {
	Secret string        // 签名密钥，**生产环境必须**设置强随机串
	TTL    time.Duration // token 有效期，默认 24h
}

// Load 加载并校验配置。返回的 Config 可直接传给 db.Open 等下游模块。
func Load() (*Config, error) {
	// .env 不存在也不报错（生产环境纯靠环境变量启动）
	_ = godotenv.Load()

	cfg := &Config{
		AppEnv:   getEnv("APP_ENV", "development"),
		AppPort:  getEnv("APP_PORT", "8080"),
		LogLevel: parseLogLevel(getEnv("LOG_LEVEL", "info")),
		DB: DBConfig{
			Host:     getEnv("DB_HOST", "127.0.0.1"),
			Port:     getEnv("DB_PORT", "3306"),
			User:     getEnv("DB_USER", "root"),
			Password: getEnv("DB_PASSWORD", ""),
			Name:     getEnv("DB_NAME", "questos"),
		},
		JWT: JWTConfig{
			Secret: getEnv("JWT_SECRET", ""),
			TTL:    getDurationEnv("JWT_TTL", 24*time.Hour),
		},
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// DSN 生成 GORM MySQL 驱动要求的连接字符串。
//
//	charset=utf8mb4  全字符集支持（包括 emoji）
//	parseTime=True   让 GORM 把 DATETIME 正确解析成 time.Time
//	loc=Local        使用本机时区，避免时间偏差
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.DB.User, c.DB.Password, c.DB.Host, c.DB.Port, c.DB.Name,
	)
}

// validate 校验关键字段。生产配置缺失关键字段时直接失败，不静默。
func (c *Config) validate() error {
	if c.JWT.Secret == "" {
		return fmt.Errorf("JWT_SECRET is required (generate with: openssl rand -hex 32)")
	}
	if c.DB.Name == "" {
		return fmt.Errorf("DB_NAME is required")
	}
	if len(c.JWT.Secret) < 16 {
		return fmt.Errorf("JWT_SECRET too short (need >= 16 chars)")
	}
	return nil
}

// ---------- helpers ----------

func getEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}

func getDurationEnv(key string, def time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

func parseLogLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}