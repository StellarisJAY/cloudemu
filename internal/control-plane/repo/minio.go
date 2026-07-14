package repo

import (
	"bytes"
	"context"
	"io"
	"time"

	"github.com/StellarisJAY/cloudemu/internal/control-plane/contract"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinioAdapter 封装 minio-go SDK，实现 contract.MinioFunc 接口
type MinioAdapter struct {
	cli *minio.Client
}

// 编译期检查 MinioAdapter 是否实现了 MinioFunc 接口
var _ contract.MinioFunc = (*MinioAdapter)(nil)

// NewMinioAdapter 创建 MinIO 客户端适配器
func NewMinioAdapter(endpoint, accessKey, secretKey string, useSSL bool) (*MinioAdapter, error) {
	cli, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}
	return &MinioAdapter{cli: cli}, nil
}

// UploadFile 上传文件到指定桶和路径
func (m *MinioAdapter) UploadFile(ctx context.Context, bucket, path string, reader io.Reader, size int64) error {
	_, err := m.cli.PutObject(ctx, bucket, path, reader, size, minio.PutObjectOptions{})
	return err
}

// GetURL 获取文件预签名下载URL（有效期1小时）
func (m *MinioAdapter) GetURL(ctx context.Context, bucket, path string) (string, error) {
	url, err := m.cli.PresignedGetObject(ctx, bucket, path, 3600, nil)
	if err != nil {
		return "", err
	}
	return url.String(), nil
}

// PresignedGetURL 获取文件预签名下载URL（自定有效期）
func (m *MinioAdapter) PresignedGetURL(ctx context.Context, bucket, path string, expiry time.Duration) (string, error) {
	url, err := m.cli.PresignedGetObject(ctx, bucket, path, expiry, nil)
	if err != nil {
		return "", err
	}
	return url.String(), nil
}

// PresignedPutURL 获取文件预签名上传URL（自定有效期），Worker 用于上传存档二进制到 MinIO
func (m *MinioAdapter) PresignedPutURL(ctx context.Context, bucket, path string, expiry time.Duration) (string, error) {
	url, err := m.cli.PresignedPutObject(ctx, bucket, path, expiry)
	if err != nil {
		return "", err
	}
	return url.String(), nil
}

// EnsureBucket 确保桶存在，不存在则自动创建
func (m *MinioAdapter) EnsureBucket(ctx context.Context, bucket string) error {
	exists, err := m.cli.BucketExists(ctx, bucket)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return m.cli.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
}
func (m *MinioAdapter) GetFile(ctx context.Context, bucket, path string) (io.ReadCloser, error) {
	obj, err := m.cli.GetObject(ctx, bucket, path, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, obj); err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(buf.Bytes())), nil
}

// RemoveFile 删除指定桶和路径的文件（用于删除内置 ROM 文件与封面）
func (m *MinioAdapter) RemoveFile(ctx context.Context, bucket, path string) error {
	return m.cli.RemoveObject(ctx, bucket, path, minio.RemoveObjectOptions{})
}
