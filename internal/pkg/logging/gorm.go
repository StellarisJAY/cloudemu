package logging

import (
	"context"
	"errors"
	"log/slog"
	"time"

	gormLogger "gorm.io/gorm/logger"
)

// NewGormLogger 创建适配 slog 的 Gorm Logger
// devMode=true 时输出所有 SQL（Info 级别），否则仅慢查询和错误
func NewGormLogger(logger *slog.Logger, devMode bool) gormLogger.Interface {
	level := gormLogger.Warn
	if devMode {
		level = gormLogger.Info
	}
	return &GormLogger{
		logger:                logger,
		SlowThreshold:         200 * time.Millisecond,
		LogLevel:              level,
		SkipErrRecordNotFound: true,
	}
}

// GormLogger 实现 gormLogger.Interface，将 Gorm 日志桥接到 slog
type GormLogger struct {
	logger                *slog.Logger
	SlowThreshold         time.Duration
	LogLevel              gormLogger.LogLevel
	SkipErrRecordNotFound bool
}

// LogMode 设置日志级别
func (l *GormLogger) LogMode(level gormLogger.LogLevel) gormLogger.Interface {
	newLogger := *l
	newLogger.LogLevel = level
	return &newLogger
}

// Info 普通日志
func (l *GormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormLogger.Info {
		l.logger.InfoContext(ctx, msg, data...)
	}
}

// Warn 警告日志
func (l *GormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormLogger.Warn {
		l.logger.WarnContext(ctx, msg, data...)
	}
}

// Error 错误日志
func (l *GormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormLogger.Error {
		l.logger.ErrorContext(ctx, msg, data...)
	}
}

// Trace 记录 SQL 执行详情
// 慢查询用 Warn，错误用 Error，正常 SQL 用 Info（仅开发模式）
func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if l.LogLevel <= gormLogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	attrs := []any{
		"sql", sql,
		"latency", elapsed.Milliseconds(),
		"rows", rows,
	}

	switch {
	case err != nil && (!l.SkipErrRecordNotFound || !errors.Is(err, gormLogger.ErrRecordNotFound)):
		l.logger.ErrorContext(ctx, "gorm query error", append(attrs, "error", err)...)
	case elapsed > l.SlowThreshold && l.SlowThreshold != 0:
		l.logger.WarnContext(ctx, "gorm slow query", attrs...)
	case l.LogLevel >= gormLogger.Info:
		l.logger.InfoContext(ctx, "gorm query", attrs...)
	}
}
