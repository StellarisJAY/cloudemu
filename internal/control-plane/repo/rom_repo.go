package repo

import (
	"context"
	"errors"

	"github.com/StellarisJAY/cloudemu/internal/control-plane/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RomRepo ROM 表数据访问层
type RomRepo struct {
	db *gorm.DB
}

func NewRomRepo(db *gorm.DB) *RomRepo {
	return &RomRepo{db: db}
}

// Create 插入ROM记录
func (r *RomRepo) Create(ctx context.Context, rom *model.Rom) error {
	return r.db.WithContext(ctx).Create(rom).Error
}

// ByID 按主键ID查询ROM
func (r *RomRepo) ByID(ctx context.Context, id uuid.UUID) (*model.Rom, error) {
	var rom model.Rom
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&rom).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &rom, err
}

// ByUploader 查询指定用户上传的所有已通过 ROM（status=1）
func (r *RomRepo) ByUploader(ctx context.Context, userID uuid.UUID) ([]model.Rom, error) {
	var roms []model.Rom
	err := r.db.WithContext(ctx).
		Where("uploader_id = ? AND status = 1", userID).
		Order("created_at DESC").
		Find(&roms).Error
	return roms, err
}

// Update 更新ROM记录（标题、封面路径等可修改字段）
func (r *RomRepo) Update(ctx context.Context, rom *model.Rom) error {
	return r.db.WithContext(ctx).Save(rom).Error
}

// BySHA256 按上传者和SHA-256哈希查重（同一用户不可重复上传同一文件）
func (r *RomRepo) BySHA256(ctx context.Context, userID uuid.UUID, sha256 string) (*model.Rom, error) {
	var rom model.Rom
	err := r.db.WithContext(ctx).
		Where("uploader_id = ? AND sha256 = ?", userID, sha256).
		First(&rom).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &rom, err
}
