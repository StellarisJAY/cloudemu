package repo

import (
	"context"
	"errors"
	"time"

	"github.com/StellarisJAY/cloudemu/internal/control-plane/contract"
	"github.com/StellarisJAY/cloudemu/internal/control-plane/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RoomRepo 房间表数据访问层
type RoomRepo struct {
	db *gorm.DB
}

func NewRoomRepo(db *gorm.DB) *RoomRepo {
	return &RoomRepo{db: db}
}

// Create 插入新房间
func (r *RoomRepo) Create(ctx context.Context, room *model.Room) error {
	return r.db.WithContext(ctx).Create(room).Error
}

// ByID 按主键ID查询房间
func (r *RoomRepo) ByID(ctx context.Context, id uuid.UUID) (*model.Room, error) {
	var room model.Room
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&room).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &room, err
}

// ActiveByUser 查询用户参与的所有活跃房间（host 或 player 且未离开）
func (r *RoomRepo) ActiveByUser(ctx context.Context, userID uuid.UUID) ([]model.Room, error) {
	subQuery := r.db.Model(&model.RoomPlayer{}).
		Select("room_id").
		Where("user_id = ? AND left_at IS NULL", userID)

	var rooms []model.Room
	err := r.db.WithContext(ctx).
		Where("status != 2 AND (host_id = ? OR id IN (?))", userID, subQuery).
		Order("created_at DESC").
		Find(&rooms).Error
	return rooms, err
}

// UpdateStatus 更新房间状态，根据状态值自动设置 started_at（开始游戏）或 closed_at（关闭房间）
func (r *RoomRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status int16) error {
	updates := map[string]interface{}{
		"status":     status,
		"closed_at":  nil,
		"started_at": nil,
	}
	now := time.Now()
	if status == 1 {
		updates["started_at"] = &now
	} else if status == 2 {
		updates["closed_at"] = &now
	}
	return r.db.WithContext(ctx).Model(&model.Room{}).Where("id = ?", id).Updates(updates).Error
}

// SetWorkerAddr 设置房间分配到的 Worker 地址（用于后续 StopGame 时知道往哪发送）
func (r *RoomRepo) SetWorkerAddr(ctx context.Context, id uuid.UUID, addr string) error {
	return r.db.WithContext(ctx).Model(&model.Room{}).Where("id = ?", id).Update("worker_addr", addr).Error
}

// UpdateRomID 更新房间当前游玩的 ROM；romID 为 nil 时写入 NULL（清空 ROM 选择）
func (r *RoomRepo) UpdateRomID(ctx context.Context, id uuid.UUID, romID *uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&model.Room{}).Where("id = ?", id).Update("rom_id", romID).Error
}

// ActiveByRomID 查询使用指定 ROM 的所有非关闭房间
func (r *RoomRepo) ActiveByRomID(ctx context.Context, romID uuid.UUID) ([]model.Room, error) {
	var rooms []model.Room
	err := r.db.WithContext(ctx).
		Where("rom_id = ? AND status != 2", romID).
		Find(&rooms).Error
	return rooms, err
}

// RoomPlayerRepo 房间座位表数据访问层
type RoomPlayerRepo struct {
	db *gorm.DB
}

func NewRoomPlayerRepo(db *gorm.DB) *RoomPlayerRepo {
	return &RoomPlayerRepo{db: db}
}

// Create 插入座位记录
func (r *RoomPlayerRepo) Create(ctx context.Context, player *model.RoomPlayer) error {
	return r.db.WithContext(ctx).Create(player).Error
}

// ActiveByRoom 查询房间内所有活跃玩家（left_at IS NULL）
func (r *RoomPlayerRepo) ActiveByRoom(ctx context.Context, roomID uuid.UUID) ([]model.RoomPlayer, error) {
	var players []model.RoomPlayer
	err := r.db.WithContext(ctx).
		Where("room_id = ? AND left_at IS NULL", roomID).
		Find(&players).Error
	return players, err
}

// ActiveByRoomWithUser 查询房间内所有活跃玩家（含用户信息，LEFT JOIN users 表）
func (r *RoomPlayerRepo) ActiveByRoomWithUser(ctx context.Context, roomID uuid.UUID) ([]contract.RoomMemberInfo, error) {
	var members []contract.RoomMemberInfo
	err := r.db.WithContext(ctx).
		Table("room_players").
		Select("room_players.user_id, users.username, users.nickname, users.avatar, room_players.role, room_players.port").
		Joins("LEFT JOIN users ON users.id = room_players.user_id").
		Where("room_players.room_id = ? AND room_players.left_at IS NULL", roomID).
		Order("room_players.role ASC, users.nickname ASC").
		Scan(&members).Error
	return members, err
}

// ActiveByUser 查询用户当前所在的所有座位
func (r *RoomPlayerRepo) ActiveByUser(ctx context.Context, userID uuid.UUID) ([]model.RoomPlayer, error) {
	var players []model.RoomPlayer
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND left_at IS NULL", userID).
		Find(&players).Error
	return players, err
}

// ByRoomAndUser 查询用户在指定房间的活跃座位
func (r *RoomPlayerRepo) ByRoomAndUser(ctx context.Context, roomID, userID uuid.UUID) (*model.RoomPlayer, error) {
	var player model.RoomPlayer
	err := r.db.WithContext(ctx).
		Where("room_id = ? AND user_id = ? AND left_at IS NULL", roomID, userID).
		First(&player).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &player, err
}

// UpdateRoleAndPort 更新玩家角色和手柄端口（用于分配/取消手柄）
func (r *RoomPlayerRepo) UpdateRoleAndPort(ctx context.Context, id uuid.UUID, role int16, port *int16) error {
	return r.db.WithContext(ctx).
		Model(&model.RoomPlayer{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{"role": role, "port": port}).Error
}

// MarkLeft 标记玩家离开房间（设置 left_at）
func (r *RoomPlayerRepo) MarkLeft(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.RoomPlayer{}).
		Where("id = ?", id).
		Update("left_at", &now).Error
}

// TransferHost 转移房主身份（事务：更新 room_players.role + rooms.host_id）
func (r *RoomPlayerRepo) TransferHost(ctx context.Context, roomID uuid.UUID, newHostID uuid.UUID) error {
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}

	if err := tx.Model(&model.RoomPlayer{}).
		Where("room_id = ? AND user_id = ? AND left_at IS NULL", roomID, newHostID).
		Update("role", 0).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Model(&model.Room{}).Where("id = ?", roomID).Update("host_id", newHostID).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}
