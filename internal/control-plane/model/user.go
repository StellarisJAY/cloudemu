package model

import (
	"time"

	"github.com/google/uuid"
)

// User 用户表，对应数据库表 users
type User struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`                             // 用户唯一标识，UUIDv7 主键
	Username     string     `gorm:"type:varchar(64);uniqueIndex;not null" json:"username"`      // 用户名，唯一，可用于登录
	Email        string     `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`        // 邮箱，唯一，注册必填，接收验证码
	PasswordHash string     `gorm:"type:varchar(255);not null" json:"-"`                        // bcrypt 哈希后的密码（不序列化到 JSON）
	Nickname     *string    `gorm:"type:varchar(64)" json:"nickname,omitempty"`                 // 昵称，展示用，可为空（空时前端展示 username）
	Avatar       *string    `gorm:"type:varchar(512)" json:"avatar,omitempty"`                  // 头像URL/路径，可为空（空时前端展示默认头像）
	Bio          *string    `gorm:"type:varchar(512)" json:"bio,omitempty"`                     // 个人简介，可为空
	Status       int16      `gorm:"type:smallint;not null;default:0;index" json:"status"`       // 账户状态：0=待激活, 1=已激活(可游戏), 2=已禁用
	IsAdmin      bool       `gorm:"type:boolean;not null;default:false;index" json:"is_admin"`  // 是否为管理员，管理员可上传/管理平台内置 ROM（手动改库授予）
	LastLoginAt  *time.Time `gorm:"type:timestamptz" json:"last_login_at,omitempty"`            // 最近一次登录时间
	CreatedAt    time.Time  `gorm:"type:timestamptz;not null;autoCreateTime" json:"created_at"` // 注册时间
	UpdatedAt    time.Time  `gorm:"type:timestamptz;not null;autoUpdateTime" json:"updated_at"` // 更新时间
}

func (User) TableName() string { return "users" }
