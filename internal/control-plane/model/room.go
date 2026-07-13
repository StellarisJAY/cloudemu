package model

import (
	"time"

	"github.com/google/uuid"
)

// Room 游戏房间表，对应数据库表 rooms
// 用户只能看到自己创建或已加入的房间，无公开大厅
type Room struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`                             // 房间唯一标识
	HostID       uuid.UUID  `gorm:"type:uuid;not null;index" json:"host_id"`                    // 房主ID，只有房主能启动游戏/分配手柄
	Title        string     `gorm:"type:varchar(128);not null" json:"title"`                    // 房间名称，房主自定义
	EmulatorType string     `gorm:"type:varchar(32);not null;index" json:"emulator_type"`       // 模拟器类型：nes, gba, dos
	RomID        *uuid.UUID `gorm:"type:uuid;index" json:"rom_id,omitempty"`                    // 关联的ROM ID，NULL=尚未选择ROM
	MaxPorts     int16      `gorm:"type:smallint;not null;default:4" json:"max_ports"`          // 最大手柄端口数，最多4人
	Status       int16      `gorm:"type:smallint;not null;default:0;index" json:"status"`       // 房间状态：0=等待中, 1=游戏中, 2=已关闭
	StartedAt    *time.Time `gorm:"type:timestamptz" json:"started_at,omitempty"`               // 游戏开始时间
	ClosedAt     *time.Time `gorm:"type:timestamptz" json:"closed_at,omitempty"`                // 房间关闭时间
	CreatedAt    time.Time  `gorm:"type:timestamptz;not null;autoCreateTime" json:"created_at"` // 房间创建时间
	UpdatedAt    time.Time  `gorm:"type:timestamptz;not null;autoUpdateTime" json:"updated_at"` // 更新时间
	WorkerAddr   string     `gorm:"type:varchar(64)" json:"worker_addr,omitempty"`              // 分配到哪个 Worker（用于 StopGame 时知道往哪发）
}

func (Room) TableName() string { return "rooms" }
