package model

import (
	"time"

	"github.com/google/uuid"
)

// RefreshToken 刷新令牌记录，对应数据库表 refresh_tokens
// 每次用户登录或刷新Token时，旧记录删除、插入新记录，实现轮换制
type RefreshToken struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`                     // 主键
	UserID    uuid.UUID `gorm:"type:uuid;not null;index"`                 // 关联的用户ID
	TokenHash string    `gorm:"type:varchar(255);not null;uniqueIndex"`   // Refresh Token 的 SHA-256 哈希值
	ExpiresAt time.Time `gorm:"type:timestamptz;not null"`                // Token 过期时间（7天）
	CreatedAt time.Time `gorm:"type:timestamptz;not null;autoCreateTime"` // 创建时间
}

func (RefreshToken) TableName() string { return "refresh_tokens" }
