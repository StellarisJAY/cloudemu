package model

import (
	"time"

	"github.com/google/uuid"
)

// Rom ROM 库表，对应数据库表 roms
// 每个用户的 ROM 互相隔离，通过 uploader_id 过滤；同一用户不可重复上传同一文件（SHA-256 去重）
type Rom struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`                             // 主键
	UploaderID   uuid.UUID `gorm:"type:uuid;not null;index" json:"-"`                          // 上传者ID（不暴露给前端，仅后端过滤用）
	Title        string    `gorm:"type:varchar(255);not null" json:"title"`                    // ROM 标题，用户自定义
	FileName     string    `gorm:"type:varchar(255);not null" json:"-"`                        // 原始文件名（不暴露）
	EmulatorType string    `gorm:"type:varchar(32);not null;index" json:"emulator_type"`       // 模拟器类型：nes, gb, dos
	FileSize     int64     `gorm:"type:bigint;not null" json:"file_size"`                      // 文件大小（字节）
	SHA256       string    `gorm:"type:varchar(64);not null;index" json:"-"`                   // 文件 SHA-256 哈希值（不暴露）
	Status       int16     `gorm:"type:smallint;not null;default:0;index" json:"status"`       // ROM 状态：0=待审核, 1=已通过(可用), 2=已拒绝
	IsBuiltin    bool      `gorm:"type:boolean;not null;default:false;index" json:"is_builtin"` // 是否为平台内置 ROM：true=管理员上传，全体用户可见可用、不可修改
	MinioPath    string    `gorm:"type:varchar(512);not null" json:"-"`                        // MinIO 上的 ROM 文件存储路径（不暴露）
	CoverPath    *string   `gorm:"type:varchar(512)" json:"cover_path,omitempty"`              // MinIO 上的封面图路径，NULL=无封面（前端用默认图）
	CreatedAt    time.Time `gorm:"type:timestamptz;not null;autoCreateTime" json:"created_at"` // 上传时间
	UpdatedAt    time.Time `gorm:"type:timestamptz;not null;autoUpdateTime" json:"updated_at"` // 更新时间
}

func (Rom) TableName() string { return "roms" }
