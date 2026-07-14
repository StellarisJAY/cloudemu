package model

import (
	"time"

	"github.com/google/uuid"
)

// SaveState 游戏存档表，对应数据库表 save_states
// 一个存档由 room_id + emulator_type + rom_id + 序列化状态数据（存 MinIO）共同组成
// 读档时三要素（room_id / emulator_type / rom_id）须与当前房间/机种/加载的 ROM 全部匹配
// 存档不随房间关闭或 EmuRunner 销毁而删除，生命周期独立于房间与子进程
type SaveState struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`                             // 存档唯一标识
	RoomID       uuid.UUID `gorm:"type:uuid;not null;index" json:"room_id"`                    // 所属房间（不随房间关闭而删除）
	Name         string    `gorm:"type:varchar(64);not null" json:"name"`                      // 存档名称（房主可改名，默认按创建时间生成）
	EmulatorType string    `gorm:"type:varchar(32);not null" json:"emulator_type"`             // 模拟器类型：nes / gb / dos
	RomID        uuid.UUID `gorm:"type:uuid;not null;index" json:"rom_id"`                     // 存档对应的 ROM
	MinioPath    string    `gorm:"type:varchar(512);not null" json:"-"`                        // MinIO 上的状态二进制路径（不暴露给前端）
	Size         int64     `gorm:"type:bigint;not null" json:"size"`                           // 序列化状态字节数
	CreatedBy    uuid.UUID `gorm:"type:uuid;not null" json:"created_by"`                       // 创建存档的用户（房主）
	CreatedAt    time.Time `gorm:"type:timestamptz;not null;autoCreateTime" json:"created_at"` // 存档时间
}

func (SaveState) TableName() string { return "save_states" }
