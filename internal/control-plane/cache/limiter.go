package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Limiter 频率限制缓存，基于 Redis，实现 service.Limiter 接口
// 提供登录防爆破、验证码重试限制、重发冷却等功能
type Limiter struct {
	cli *redis.Client
}

func NewLimiter(cli *redis.Client) *Limiter {
	return &Limiter{cli: cli}
}

// IncrLoginAttempt 增加登录失败计数（key: login_attempt:{ip}），首次设置15分钟过期
func (l *Limiter) IncrLoginAttempt(ctx context.Context, ip string) (int64, error) {
	key := "login_attempt:" + ip
	val, err := l.cli.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	if val == 1 {
		l.cli.Expire(ctx, key, 15*time.Minute)
	}
	return val, nil
}

// IsLoginLocked 检查登录是否被锁定（失败达到5次）
func (l *Limiter) IsLoginLocked(ctx context.Context, ip string) (bool, error) {
	key := "login_attempt:" + ip
	val, err := l.cli.Get(ctx, key).Int64()
	if err != nil {
		return false, err
	}
	return val >= 5, nil
}

// IncrVerifyAttempt 增加验证尝试计数（key: verify_attempt:{email}），首次设置5分钟过期
func (l *Limiter) IncrVerifyAttempt(ctx context.Context, email string) (int64, error) {
	key := "verify_attempt:" + email
	val, err := l.cli.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	if val == 1 {
		l.cli.Expire(ctx, key, 5*time.Minute)
	}
	return val, nil
}

// IsVerifyLocked 检查验证是否被锁定（失败达到3次）
func (l *Limiter) IsVerifyLocked(ctx context.Context, email string) (bool, error) {
	key := "verify_attempt:" + email
	val, err := l.cli.Get(ctx, key).Int64()
	if err != nil {
		return false, err
	}
	return val >= 3, nil
}

// CheckResendCooldown 检查重发冷却期是否未过（key: resend_cooldown:{email}，60秒TTL）
func (l *Limiter) CheckResendCooldown(ctx context.Context, email string) (bool, error) {
	key := "resend_cooldown:" + email
	exists, err := l.cli.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

// SetResendCooldown 设置重发冷却标记，60秒后自动过期
func (l *Limiter) SetResendCooldown(ctx context.Context, email string) error {
	return l.cli.Set(ctx, "resend_cooldown:"+email, "1", 60*time.Second).Err()
}

// IncrForgotPassword 增加忘记密码尝试计数（key: forgot_pwd:{ip}），首次设置5分钟过期
func (l *Limiter) IncrForgotPassword(ctx context.Context, ip string) (int64, error) {
	key := "forgot_pwd:" + ip
	val, err := l.cli.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	if val == 1 {
		l.cli.Expire(ctx, key, 5*time.Minute)
	}
	return val, nil
}

// IsForgotPasswordLocked 检查忘记密码请求是否被锁定（失败达到3次）
func (l *Limiter) IsForgotPasswordLocked(ctx context.Context, ip string) (bool, error) {
	key := "forgot_pwd:" + ip
	val, err := l.cli.Get(ctx, key).Int64()
	if err != nil {
		return false, err
	}
	return val >= 3, nil
}

// CheckForgotCooldown 检查忘记密码邮件重发冷却期（key: forgot_cooldown:{email}，60秒TTL）
func (l *Limiter) CheckForgotCooldown(ctx context.Context, email string) (bool, error) {
	key := "forgot_cooldown:" + email
	exists, err := l.cli.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

// SetForgotCooldown 设置忘记密码邮件重发冷却标记，60秒后自动过期
func (l *Limiter) SetForgotCooldown(ctx context.Context, email string) error {
	return l.cli.Set(ctx, "forgot_cooldown:"+email, "1", 60*time.Second).Err()
}
