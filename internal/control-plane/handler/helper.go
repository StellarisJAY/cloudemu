package handler

import (
	"github.com/StellarisJAY/cloudemu/internal/pkg/apperror"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// parseUUIDParam 从 gin.Context 的 URL Path 参数中解析 UUID
// 拒绝空串、非法格式以及全零 UUID（uuid.Nil）
// 返回值：解析得到的 uuid.UUID 与 nil error；失败时返回 uuid.Nil 与 apperror.ErrInvalidParam
func parseUUIDParam(c *gin.Context, key string) (uuid.UUID, error) {
	raw := c.Param(key)
	if raw == "" {
		return uuid.Nil, apperror.ErrInvalidParam
	}
	id, err := uuid.Parse(raw)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, apperror.ErrInvalidParam
	}
	return id, nil
}
