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

// ListForUser 查询用户可用 ROM：自有非内置 ROM + 全部平台内置 ROM（status=1）
// 内置 ROM 对全体用户可见，自有 ROM 仅本人可见；两个集合互斥，结果无重复
func (r *RomRepo) ListForUser(ctx context.Context, userID uuid.UUID) ([]model.Rom, error) {
	var roms []model.Rom
	err := r.db.WithContext(ctx).
		Where("status = 1 AND (is_builtin = true OR (uploader_id = ? AND is_builtin = false))", userID).
		Order("is_builtin DESC, created_at DESC").
		Find(&roms).Error
	return roms, err
}

// ListBuiltin 查询全部平台内置 ROM（status=1），供管理后台使用
func (r *RomRepo) ListBuiltin(ctx context.Context) ([]model.Rom, error) {
	var roms []model.Rom
	err := r.db.WithContext(ctx).
		Where("is_builtin = true AND status = 1").
		Order("created_at DESC").
		Find(&roms).Error
	return roms, err
}

// Update 更新ROM记录（标题、封面路径等可修改字段）
func (r *RomRepo) Update(ctx context.Context, rom *model.Rom) error {
	return r.db.WithContext(ctx).Save(rom).Error
}

// Delete 删除ROM记录（按主键，用于删除内置 ROM）
func (r *RomRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.Rom{}).Error
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

// BuiltinBySHA256 按SHA-256哈希查重内置 ROM（内置 ROM 全局唯一，不区分上传的管理员）
func (r *RomRepo) BuiltinBySHA256(ctx context.Context, sha256 string) (*model.Rom, error) {
	var rom model.Rom
	err := r.db.WithContext(ctx).
		Where("is_builtin = true AND sha256 = ?", sha256).
		First(&rom).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &rom, err
}
