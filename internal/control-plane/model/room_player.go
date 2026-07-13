package model

import (
	"time"

	"github.com/google/uuid"
)

// RoomPlayer 房间座位表，对应数据库表 room_players
// 记录每个玩家在房间中的角色和手柄分配情况
type RoomPlayer struct {
	ID       uuid.UUID  `gorm:"type:uuid;primaryKey"`                     // 主键
	RoomID   uuid.UUID  `gorm:"type:uuid;not null;index"`                 // 关联的房间ID
	UserID   uuid.UUID  `gorm:"type:uuid;not null;index"`                 // 关联的用户ID
	Role     int16      `gorm:"type:smallint;not null;default:0"`         // 玩家角色：0=房主, 1=操作者(有手柄), 2=旁观者
	Port     *int16     `gorm:"type:smallint"`                            // 绑定的手柄端口号，0-based，NULL=旁观（无手柄）
	JoinedAt time.Time  `gorm:"type:timestamptz;not null;autoCreateTime"` // 加入房间时间
	LeftAt   *time.Time `gorm:"type:timestamptz"`                         // 离开房间时间，NULL=仍在房间中
}

func (RoomPlayer) TableName() string { return "room_players" }
