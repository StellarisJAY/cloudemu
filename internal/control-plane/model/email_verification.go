package model

import (
	"time"

	"github.com/google/uuid"
)

// EmailVerification 邮箱验证码记录，对应数据库表 email_verifications
type EmailVerification struct {
	ID               uuid.UUID  `gorm:"type:uuid;primaryKey"`                     // 主键
	UserID           uuid.UUID  `gorm:"type:uuid;not null;index"`                 // 关联的用户ID
	Email            string     `gorm:"type:varchar(255);not null;index"`         // 发送到的邮箱地址
	VerificationCode string     `gorm:"type:varchar(8);not null"`                 // 6位数字验证码
	ExpiresAt        time.Time  `gorm:"type:timestamptz;not null"`                // 验证码过期时间（15分钟有效期）
	VerifiedAt       *time.Time `gorm:"type:timestamptz"`                         // 验证完成时间，NULL表示尚未验证
	CreatedAt        time.Time  `gorm:"type:timestamptz;not null;autoCreateTime"` // 发送时间
}

func (EmailVerification) TableName() string { return "email_verifications" }
