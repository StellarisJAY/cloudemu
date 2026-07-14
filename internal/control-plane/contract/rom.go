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
	List(ctx context.Context, userID uuid.UUID) ([]model.Rom, error)                                                                                // 列出当前用户可用ROM（自有ROM + 全部平台内置ROM）
	Update(ctx context.Context, userID uuid.UUID, romID uuid.UUID, req UpdateRomReq, coverFile io.Reader, coverFileName string) (*model.Rom, error) // 更新自有ROM标题和封面（内置ROM不可改）

	// ---- 管理员：平台内置 ROM 管理 ----
	UploadBuiltin(ctx context.Context, adminID uuid.UUID, req UploadRomReq, romFile io.Reader, romFileName string, romFileSize int64) (*model.Rom, error) // 管理员上传平台内置ROM（is_builtin=true，全体用户可见）
	ListBuiltin(ctx context.Context) ([]model.Rom, error)                                                                                                 // 列出全部平台内置ROM（管理后台用）
	UpdateBuiltin(ctx context.Context, romID uuid.UUID, req UpdateRomReq, coverFile io.Reader, coverFileName string) (*model.Rom, error)                  // 管理员更新内置ROM标题和封面
	DeleteBuiltin(ctx context.Context, romID uuid.UUID) error                                                                                             // 管理员删除内置ROM（含MinIO文件与封面）
}

// RomRepo ROM 表数据访问接口
type RomRepo interface {
	Create(ctx context.Context, rom *model.Rom) error                                  // 插入ROM记录
	Update(ctx context.Context, rom *model.Rom) error                                  // 更新ROM记录
	Delete(ctx context.Context, id uuid.UUID) error                                    // 删除ROM记录
	ByID(ctx context.Context, id uuid.UUID) (*model.Rom, error)                        // 按ID查询ROM
	ListForUser(ctx context.Context, userID uuid.UUID) ([]model.Rom, error)            // 查用户可用ROM：自有非内置ROM + 全部内置ROM（status=1）
	ListBuiltin(ctx context.Context) ([]model.Rom, error)                              // 查全部内置ROM（status=1）
	BySHA256(ctx context.Context, userID uuid.UUID, sha256 string) (*model.Rom, error) // 按SHA-256查同一用户是否已上传过相同文件
	BuiltinBySHA256(ctx context.Context, sha256 string) (*model.Rom, error)            // 按SHA-256查是否已存在相同内置ROM（内置ROM全局去重）
}

// MinioFunc MinIO 对象存储操作接口
type MinioFunc interface {
	UploadFile(ctx context.Context, bucket, path string, reader io.Reader, size int64) error        // 上传文件到指定桶和路径
	GetURL(ctx context.Context, bucket, path string) (string, error)                                // 获取文件预签名URL（1小时有效期）
	PresignedGetURL(ctx context.Context, bucket, path string, expiry time.Duration) (string, error) // 获取文件预签名URL（自定有效期）
	PresignedPutURL(ctx context.Context, bucket, path string, expiry time.Duration) (string, error) // 获取文件预签名上传URL（自定有效期，用于上传存档二进制）
	GetFile(ctx context.Context, bucket, path string) (io.ReadCloser, error)                        // 读取文件内容（用于代理下载）
	RemoveFile(ctx context.Context, bucket, path string) error                                      // 删除文件（用于删除内置ROM文件与封面）
	EnsureBucket(ctx context.Context, bucket string) error                                          // 确保桶存在，不存在则创建
}
