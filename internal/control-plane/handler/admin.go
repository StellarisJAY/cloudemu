package handler

import (
	"io"
	"strconv"

	"github.com/StellarisJAY/cloudemu/internal/control-plane/contract"
	"github.com/StellarisJAY/cloudemu/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AdminHandler 管理员相关 HTTP 处理器（当前仅平台内置 ROM 管理）
type AdminHandler struct {
	svc contract.RomService
}

// NewAdminHandler 创建 AdminHandler 实例
func NewAdminHandler(svc contract.RomService) *AdminHandler {
	return &AdminHandler{svc: svc}
}

// ListBuiltin GET /api/admin/roms — 列出全部平台内置 ROM（需管理员）
func (h *AdminHandler) ListBuiltin(c *gin.Context) {
	roms, err := h.svc.ListBuiltin(c.Request.Context())
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

// UploadBuiltin POST /api/admin/roms/upload — 上传平台内置 ROM（需管理员，multipart/form-data）
// 文件通过 form "rom" 字段提交，title 通过表单字段提交
func (h *AdminHandler) UploadBuiltin(c *gin.Context) {
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

	adminID := c.MustGet("user_id").(uuid.UUID)

	rom, err := h.svc.UploadBuiltin(c.Request.Context(), adminID, contract.UploadRomReq{Title: title}, file, header.Filename, header.Size)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, rom)
}

// UpdateBuiltin PUT /api/admin/roms/:id — 更新内置 ROM 标题和封面（需管理员，multipart/form-data）
// 标题通过表单字段 title 提交，封面图片通过 form "cover" 可选提交
func (h *AdminHandler) UpdateBuiltin(c *gin.Context) {
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

	var coverFile io.Reader
	var coverFileName string
	file, header, err := c.Request.FormFile("cover")
	if err == nil {
		defer file.Close()
		coverFile = file
		coverFileName = header.Filename
	}

	rom, err := h.svc.UpdateBuiltin(c.Request.Context(), romID, contract.UpdateRomReq{Title: &title}, coverFile, coverFileName)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, rom)
}

// DeleteBuiltin DELETE /api/admin/roms/:id — 删除内置 ROM（需管理员）
func (h *AdminHandler) DeleteBuiltin(c *gin.Context) {
	romID, err := parseUUIDParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}

	if err := h.svc.DeleteBuiltin(c.Request.Context(), romID); err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, gin.H{"deleted": romID.String()})
}
