package repo

import (
	"context"
	"errors"

	"github.com/StellarisJAY/cloudemu/internal/control-plane/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SaveStateRepo 游戏存档表数据访问层
type SaveStateRepo struct {
	db *gorm.DB
}

func NewSaveStateRepo(db *gorm.DB) *SaveStateRepo {
	return &SaveStateRepo{db: db}
}

// Create 插入存档记录
func (r *SaveStateRepo) Create(ctx context.Context, ss *model.SaveState) error {
	return r.db.WithContext(ctx).Create(ss).Error
}

// ByID 按主键ID查询存档
func (r *SaveStateRepo) ByID(ctx context.Context, id uuid.UUID) (*model.SaveState, error) {
	var ss model.SaveState
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&ss).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &ss, err
}

// ListByRoom 查询指定房间的所有存档（创建时间倒序）
func (r *SaveStateRepo) ListByRoom(ctx context.Context, roomID uuid.UUID) ([]model.SaveState, error) {
	var states []model.SaveState
	err := r.db.WithContext(ctx).
		Where("room_id = ?", roomID).
		Order("created_at DESC").
		Find(&states).Error
	return states, err
}

// ListByRoomRom 查询指定房间、模拟器类型、ROM 三者匹配的存档（创建时间倒序）
// 用于房间存档列表，避免展示当前 ROM/机种以外的存档
func (r *SaveStateRepo) ListByRoomRom(ctx context.Context, roomID uuid.UUID, emulatorType string, romID uuid.UUID) ([]model.SaveState, error) {
	var states []model.SaveState
	err := r.db.WithContext(ctx).
		Where("room_id = ? AND emulator_type = ? AND rom_id = ?", roomID, emulatorType, romID).
		Order("created_at DESC").
		Find(&states).Error
	return states, err
}
