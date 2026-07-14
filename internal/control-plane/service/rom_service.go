package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/StellarisJAY/cloudemu/internal/control-plane/contract"
	"github.com/StellarisJAY/cloudemu/internal/control-plane/model"
	"github.com/StellarisJAY/cloudemu/internal/pkg/apperror"

	"github.com/google/uuid"
)

// RomService ROM 管理业务逻辑实现
type RomService struct {
	romRepo     contract.RomRepo
	minioFunc   contract.MinioFunc
	minioBucket string
}

// NewRomService 创建 RomService 实例
func NewRomService(romRepo contract.RomRepo, minioFunc contract.MinioFunc, minioBucket string) *RomService {
	return &RomService{
		romRepo:     romRepo,
		minioFunc:   minioFunc,
		minioBucket: minioBucket,
	}
}

// maxSizeByEmulator 各模拟器类型对应的ROM文件大小上限（字节）
var maxSizeByEmulator = map[string]int64{
	"nes": 2 * 1024 * 1024,   // NES ROM 最大 2MB
	"gb": 32 * 1024 * 1024,   // GB/GBC/GBA ROM 最大 32MB
	"dos": 256 * 1024 * 1024, // DOS ROM 最大 256MB（dosbox 加载 zip 镜像）
}

// Upload 上传 ROM
// 流程：根据扩展名检测模拟器类型 → 校验文件大小 → 计算SHA-256去重 → 上传MinIO → 插入roms(status=1直接可用)
// 注意：封面图片处理留待后续，当前cover_path为空
func (s *RomService) Upload(ctx context.Context, userID uuid.UUID, req contract.UploadRomReq, romFile io.Reader, romFileName string, romFileSize int64) (*model.Rom, error) {
	emulatorType := detectEmulatorType(romFileName)

	maxSize, ok := maxSizeByEmulator[emulatorType]
	if !ok {
		return nil, apperror.ErrRomInvalidFormat
	}
	if romFileSize > maxSize {
		return nil, apperror.ErrRomTooLarge
	}

	romBytes, err := io.ReadAll(romFile)
	if err != nil {
		return nil, apperror.ErrInternal
	}

	hash := sha256.Sum256(romBytes)
	sha256Hex := fmt.Sprintf("%x", hash)

	existing, _ := s.romRepo.BySHA256(ctx, userID, sha256Hex)
	if existing != nil {
		return nil, apperror.ErrRomDuplicate
	}

	romID := uuid.Must(uuid.NewV7())
	ext := filepath.Ext(romFileName)
	if ext == "" {
		ext = ".rom"
	}
	minioPath := fmt.Sprintf("rom/%s/%s%s", userID.String(), romID.String(), ext)

	rom := &model.Rom{
		ID:           romID,
		UploaderID:   userID,
		Title:        req.Title,
		FileName:     romFileName,
		EmulatorType: emulatorType,
		FileSize:     romFileSize,
		SHA256:       sha256Hex,
		Status:       1, // MVP阶段上传即通过，无需审核
		MinioPath:    minioPath,
		CoverPath:    nil,
	}

	if err := s.minioFunc.UploadFile(ctx, s.minioBucket, minioPath, bytes.NewReader(romBytes), romFileSize); err != nil {
		slog.Error("upload rom to minio error", "error", err)
		return nil, apperror.ErrInternal
	}

	if err := s.romRepo.Create(ctx, rom); err != nil {
		slog.Error("insert rom error", "error", err)
		return nil, apperror.ErrInternal
	}

	return rom, nil
}

// List 列出当前用户上传的所有已通过 ROM
func (s *RomService) List(ctx context.Context, userID uuid.UUID) ([]model.Rom, error) {
	return s.romRepo.ByUploader(ctx, userID)
}

// Update 更新ROM标题和封面
// 流程：校验ROM存在且属于当前用户 → 更新标题 → 如果提供了封面文件则上传MinIO → 更新DB记录
func (s *RomService) Update(ctx context.Context, userID uuid.UUID, romID uuid.UUID, req contract.UpdateRomReq, coverFile io.Reader, coverFileName string) (*model.Rom, error) {
	rom, err := s.romRepo.ByID(ctx, romID)
	if err != nil {
		return nil, apperror.ErrInternal
	}
	if rom == nil {
		return nil, apperror.ErrRomNotExist
	}
	if rom.UploaderID != userID {
		return nil, apperror.ErrRomNotExist
	}

	rom.Title = *req.Title

	if coverFile != nil {
		ext := filepath.Ext(coverFileName)
		if ext == "" {
			ext = ".png"
		}
		coverPath := fmt.Sprintf("rom/%s/cover/%s%s", userID.String(), romID.String(), ext)

		coverBytes, err := io.ReadAll(coverFile)
		if err != nil {
			return nil, apperror.ErrInternal
		}
		if err := s.minioFunc.UploadFile(ctx, s.minioBucket, coverPath, bytes.NewReader(coverBytes), int64(len(coverBytes))); err != nil {
			return nil, apperror.ErrInternal
		}
		rom.CoverPath = &coverPath
	}

	if err := s.romRepo.Update(ctx, rom); err != nil {
		return nil, apperror.ErrInternal
	}

	return rom, nil
}

// detectEmulatorType 根据文件扩展名检测模拟器类型
func detectEmulatorType(fileName string) string {
	ext := strings.ToLower(filepath.Ext(fileName))
	switch ext {
	case ".nes":
		return "nes"
	case ".gba":
		return "gb"
	case ".gbc":
		return "gb"
	case ".zip":
		return "dos"
	default:
		return "unknown"
	}
}
