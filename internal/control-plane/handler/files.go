package handler

import (
	"io"
	"log/slog"
	"net/http"

	"github.com/StellarisJAY/cloudemu/internal/control-plane/contract"
	"github.com/StellarisJAY/cloudemu/internal/pkg/config"

	"github.com/gin-gonic/gin"
)

// FileHandler MinIO 文件代理 HTTP 处理器
// 通过 Control Plane 代理下载 MinIO 文件，不直接暴露 MinIO 地址给前端
type FileHandler struct {
	minioFunc contract.MinioFunc
	bucket    string
}

// NewFileHandler 创建 FileHandler 实例
func NewFileHandler(minioFunc contract.MinioFunc, cfg *config.Config) *FileHandler {
	return &FileHandler{minioFunc: minioFunc, bucket: cfg.MinioBucket}
}

// Proxy GET /api/files/*path — 代理获取 MinIO 文件
// 请求路径形如 /api/files/rom/{uploader_id}/{rom_id}.nes
func (h *FileHandler) Proxy(c *gin.Context) {
	path := c.Param("path")
	slog.Info("get minio file: ", "path", path)
	if path == "" || path == "/" {
		c.Status(http.StatusNotFound)
		return
	}

	reader, err := h.minioFunc.GetFile(c.Request.Context(), h.bucket, path)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	defer reader.Close()

	c.Status(http.StatusOK)
	io.Copy(c.Writer, reader)
}
