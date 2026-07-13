package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/StellarisJAY/cloudemu/internal/control-plane/contract"
	"github.com/redis/go-redis/v9"
)

// Captcha 验证码缓存，基于 Redis，实现 contract.CaptchaCache 接口
// Key 格式：
//
//	captcha:{key}           — 滑块目标坐标（阶段1校验后删除）
//	captcha_verified:{key}  — 验证通过标记（阶段2登录消费后删除）
type Captcha struct {
	cli *redis.Client
}

func NewCaptcha(cli *redis.Client) *Captcha {
	return &Captcha{cli: cli}
}

// Set 存储滑块验证码目标坐标（JSON），设置 TTL 过期时间（默认5分钟）
func (c *Captcha) Set(ctx context.Context, key string, data *contract.SlideCaptchaData, ttl time.Duration) error {
	val, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return c.cli.Set(ctx, "captcha:"+key, string(val), ttl).Err()
}

// GetAndDel 获取验证码目标坐标并删除 key（阶段1一次性使用，防止重放攻击）
func (c *Captcha) GetAndDel(ctx context.Context, key string) (*contract.SlideCaptchaData, error) {
	fullKey := "captcha:" + key
	result, err := c.cli.Get(ctx, fullKey).Result()
	if err != nil {
		return nil, err
	}
	_ = c.cli.Del(ctx, fullKey)

	var data contract.SlideCaptchaData
	if err := json.Unmarshal([]byte(result), &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// SetVerified 标记验证码已通过阶段1校验（TTL 60 秒，超时未登录需重新验证）
func (c *Captcha) SetVerified(ctx context.Context, key string, ttl time.Duration) error {
	return c.cli.Set(ctx, "captcha_verified:"+key, "1", ttl).Err()
}

// ConsumeVerified 检查并删除验证通过标记（阶段2一次性消费）
// 返回 true 表示标记存在且已消费，false 表示标记不存在（未验证或已过期）
func (c *Captcha) ConsumeVerified(ctx context.Context, key string) (bool, error) {
	fullKey := "captcha_verified:" + key
	result, err := c.cli.Get(ctx, fullKey).Result()
	if err != nil {
		return false, nil // key 不存在视为未验证
	}
	_ = c.cli.Del(ctx, fullKey)
	return result == "1", nil
}
