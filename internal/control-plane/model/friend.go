package model

import (
	"time"

	"github.com/google/uuid"
)

// Friend 好友关系表，对应数据库表 friends
// 使用 (LEAST(user_id, friend_id), GREATEST(user_id, friend_id)) 唯一约束保证不会重复建立关系
type Friend struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey"`                     // 主键
	UserID     uuid.UUID  `gorm:"type:uuid;not null;index"`                 // 好友关系发起方
	FriendID   uuid.UUID  `gorm:"type:uuid;not null;index"`                 // 好友关系接收方
	Status     int16      `gorm:"type:smallint;not null;default:0;index"`   // 关系状态：0=待接受, 1=已接受(好友), 2=已拉黑, 3=已拒绝
	AcceptedAt *time.Time `gorm:"type:timestamptz"`                         // 接受好友请求的时间
	CreatedAt  time.Time  `gorm:"type:timestamptz;not null;autoCreateTime"` // 创建时间
	UpdatedAt  time.Time  `gorm:"type:timestamptz;not null;autoUpdateTime"` // 更新时间
}

func (Friend) TableName() string { return "friends" }
