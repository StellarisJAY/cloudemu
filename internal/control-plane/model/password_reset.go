package model

import (
	"time"

	"github.com/google/uuid"
)

// PasswordReset 密码重置记录，对应数据库表 password_resets
// 用户请求忘记密码后生成随机 token，SHA-256 哈希后存入 token_hash 字段
// 原始 token 仅通过邮件发送给用户（MVP 阶段暂不发送邮件）
// 用户在重置页面提交 token + 新密码后，校验 token_hash 匹配且未过期未使用
type PasswordReset struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey"`                     // 主键
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index"`                 // 关联的用户ID
	Email     string     `gorm:"type:varchar(255);not null;index"`         // 冗余邮箱，加速查询
	TokenHash string     `gorm:"type:varchar(255);not null;uniqueIndex"`   // 重置 token 的 SHA-256 哈希值
	ExpiresAt time.Time  `gorm:"type:timestamptz;not null"`                // token 过期时间（15分钟）
	UsedAt    *time.Time `gorm:"type:timestamptz"`                         // 使用时间，NULL表示尚未使用
	CreatedAt time.Time  `gorm:"type:timestamptz;not null;autoCreateTime"` // 创建时间
}

func (PasswordReset) TableName() string { return "password_resets" }
