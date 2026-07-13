package contract

import (
	"context"
	"io"
	"time"

	"github.com/StellarisJAY/cloudemu/internal/control-plane/model"
	"github.com/google/uuid"
)

// RomService ROM 管理业务逻辑接口
type RomService interface {
	Upload(ctx context.Context, userID uuid.UUID, req UploadRomReq, romFile io.Reader, romFileName string, romFileSize int64) (*model.Rom, error)   // 上传ROM文件，校验格式/大小/SHA-256去重后存入MinIO
	List(ctx context.Context, userID uuid.UUID) ([]model.Rom, error)                                                                                // 列出当前用户上传的所有已通过ROM
	Update(ctx context.Context, userID uuid.UUID, romID uuid.UUID, req UpdateRomReq, coverFile io.Reader, coverFileName string) (*model.Rom, error) // 更新ROM标题和封面
}

// RomRepo ROM 表数据访问接口
type RomRepo interface {
	Create(ctx context.Context, rom *model.Rom) error                                  // 插入ROM记录
	Update(ctx context.Context, rom *model.Rom) error                                  // 更新ROM记录
	ByID(ctx context.Context, id uuid.UUID) (*model.Rom, error)                        // 按ID查询ROM
	ByUploader(ctx context.Context, userID uuid.UUID) ([]model.Rom, error)             // 查某用户所有已通过ROM
	BySHA256(ctx context.Context, userID uuid.UUID, sha256 string) (*model.Rom, error) // 按SHA-256查同一用户是否已上传过相同文件
}

// MinioFunc MinIO 对象存储操作接口
type MinioFunc interface {
	UploadFile(ctx context.Context, bucket, path string, reader io.Reader, size int64) error        // 上传文件到指定桶和路径
	GetURL(ctx context.Context, bucket, path string) (string, error)                                // 获取文件预签名URL（1小时有效期）
	PresignedGetURL(ctx context.Context, bucket, path string, expiry time.Duration) (string, error) // 获取文件预签名URL（自定有效期）
	GetFile(ctx context.Context, bucket, path string) (io.ReadCloser, error)                        // 读取文件内容（用于代理下载）
	EnsureBucket(ctx context.Context, bucket string) error                                          // 确保桶存在，不存在则创建
}
