// logging 包提供基于 slog 的统一日志方案
// 支持按天轮转日志文件 + stdout 双输出
package logging

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/StellarisJAY/cloudemu/internal/pkg/config"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
)

// MustNew 创建 slog.Logger，启动失败直接 panic
// 输出同时写入 stdout 和按天轮转的日志文件
// 通过 cfg.LogLevel 控制日志级别，cfg.LogJSON 控制输出格式
func MustNew(cfg *config.Config) *slog.Logger {
	level := parseLevel(cfg.LogLevel)

	var handler slog.Handler

	if cfg.LogJSON {
		handler = slog.NewJSONHandler(createMultiWriter(cfg), &slog.HandlerOptions{
			Level:     level,
			AddSource: true,
		})
	} else {
		handler = slog.NewTextHandler(createMultiWriter(cfg), &slog.HandlerOptions{
			Level:     level,
			AddSource: true,
		})
	}

	return slog.New(handler)
}

// createMultiWriter 创建 stdout + 日志文件双输出 writer
// 日志文件按天轮转，保留 30 天
func createMultiWriter(cfg *config.Config) io.Writer {
	logDir := cfg.LogDir
	if logDir == "" {
		logDir = "logs"
	}

	// 确保日志目录存在
	if err := os.MkdirAll(logDir, 0755); err != nil {
		slog.Error("failed to create log directory, falling back to stdout only", "dir", logDir, "error", err)
		return os.Stdout
	}

	// 按天轮转：cloudemu-2026-06-04.log
	pattern := filepath.Join(logDir, "cloudemu-%Y-%m-%d.log")

	rotator, err := rotatelogs.New(
		pattern,
		rotatelogs.WithRotationTime(24*time.Hour),
		rotatelogs.WithMaxAge(30*24*time.Hour),
	)
	if err != nil {
		slog.Error("failed to create log rotator, falling back to stdout only", "error", err)
		return os.Stdout
	}

	return io.MultiWriter(os.Stdout, rotator)
}

// parseLevel 将字符串日志级别转为 slog.Level，默认 Info
func parseLevel(s string) slog.Level {
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
