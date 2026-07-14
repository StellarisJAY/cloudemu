package handler

import (
	"io"
	"strconv"

	"github.com/StellarisJAY/cloudemu/internal/control-plane/contract"
	"github.com/StellarisJAY/cloudemu/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RomHandler ROM 管理相关 HTTP 处理器
type RomHandler struct {
	svc contract.RomService
}

// NewRomHandler 创建 RomHandler 实例
func NewRomHandler(svc contract.RomService) *RomHandler {
	return &RomHandler{svc: svc}
}

// Upload POST /api/roms/upload — 上传 ROM（需登录，multipart/form-data）
// 文件通过 form "rom" 字段提交，title 通过表单字段提交
func (h *RomHandler) Upload(c *gin.Context) {
	title := c.PostForm("title")
	if title == "" {
		response.BadRequest(c, "参数错误: title is required")
		return
	}

	file, header, err := c.Request.FormFile("rom")
	if err != nil {
		response.BadRequest(c, "参数错误: 缺少 rom 文件")
		return
	}
	defer file.Close()

	userID := c.MustGet("user_id").(uuid.UUID)

	req := contract.UploadRomReq{
		Title: title,
	}

	rom, err := h.svc.Upload(c.Request.Context(), userID, req, file, header.Filename, header.Size)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, rom)
}

// List GET /api/roms — 列出当前用户的所有 ROM（需登录）
func (h *RomHandler) List(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	roms, err := h.svc.List(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	type romItem struct {
		ID           string  `json:"id"`
		Title        string  `json:"title"`
		EmulatorType string  `json:"emulator_type"`
		FileSize     int64   `json:"file_size"`
		CoverPath    *string `json:"cover_path"`
		IsBuiltin    bool    `json:"is_builtin"`
		CreatedAt    string  `json:"created_at"`
	}

	result := make([]romItem, 0, len(roms))
	for _, r := range roms {
		result = append(result, romItem{
			ID:           r.ID.String(),
			Title:        r.Title,
			EmulatorType: r.EmulatorType,
			FileSize:     r.FileSize,
			CoverPath:    r.CoverPath,
			IsBuiltin:    r.IsBuiltin,
			CreatedAt:    r.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	response.OK(c, gin.H{"roms": result, "total": strconv.Itoa(len(roms))})
}

// Update PUT /api/roms/:id — 更新ROM标题和封面（需登录，multipart/form-data）
// 标题通过表单字段 title 提交，封面图片通过 form "cover" 可选提交
func (h *RomHandler) Update(c *gin.Context) {
	romID, err := parseUUIDParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}

	title := c.PostForm("title")
	if title == "" {
		response.BadRequest(c, "参数错误: title is required")
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)

	req := contract.UpdateRomReq{
		Title: &title,
	}

	var coverFile io.Reader
	var coverFileName string
	file, header, err := c.Request.FormFile("cover")
	if err == nil {
		defer file.Close()
		coverFile = file
		coverFileName = header.Filename
	}
	// cover 字段不存在时不报错，仅跳过封面更新

	rom, err := h.svc.Update(c.Request.Context(), userID, romID, req, coverFile, coverFileName)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, rom)
}
