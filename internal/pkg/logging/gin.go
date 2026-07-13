package logging

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// GinLogger 创建 gin 请求日志中间件，使用 slog 记录
// 替代 gin.Default() 内置的 gin.Logger()
// 记录字段：method、path、status、latency、client_ip
func GinLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		rawQuery := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)

		// 拼接 path + query string（如有）
		if rawQuery != "" {
			path = path + "?" + rawQuery
		}

		attrs := []slog.Attr{
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.Int("status", c.Writer.Status()),
			slog.Duration("latency", latency),
			slog.String("ip", c.ClientIP()),
		}

		// user_id 从 JWT 中间件注入的 context 提取
		if uid, exists := c.Get("user_id"); exists {
			attrs = append(attrs, slog.Any("user_id", uid))
		}

		msg := "request"
		status := c.Writer.Status()
		switch {
		case status >= 500:
			logger.LogAttrs(c.Request.Context(), slog.LevelError, msg, attrs...)
		case status >= 400:
			logger.LogAttrs(c.Request.Context(), slog.LevelWarn, msg, attrs...)
		default:
			logger.LogAttrs(c.Request.Context(), slog.LevelInfo, msg, attrs...)
		}
	}
}
