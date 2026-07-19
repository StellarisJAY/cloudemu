package handler

import (
	"github.com/StellarisJAY/cloudemu/internal/control-plane/contract"
	"github.com/StellarisJAY/cloudemu/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RoomHandler 房间相关 HTTP 处理器
type RoomHandler struct {
	svc contract.RoomService
}

// NewRoomHandler 创建 RoomHandler 实例
func NewRoomHandler(svc contract.RoomService) *RoomHandler {
	return &RoomHandler{svc: svc}
}

// Create POST /api/rooms/create — 创建房间（需登录）
func (h *RoomHandler) Create(c *gin.Context) {
	var req contract.CreateRoomReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	hostID := c.MustGet("user_id").(uuid.UUID)

	room, err := h.svc.Create(c.Request.Context(), hostID, req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, room)
}

// List GET /api/rooms — 列出当前用户参与的活跃房间（需登录）
func (h *RoomHandler) List(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	rooms, err := h.svc.List(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, rooms)
}

// InviteToRoom POST /api/rooms/invite — 房主邀请好友加入已有房间，直接加入无需接受（需登录）
func (h *RoomHandler) InviteToRoom(c *gin.Context) {
	var req contract.InviteToRoomReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	hostID := c.MustGet("user_id").(uuid.UUID)

	if err := h.svc.InviteToRoom(c.Request.Context(), hostID, *req.RoomID, req.InviteeIDs); err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, nil)
}

// ChangeRole POST /api/rooms/change-role — 房主调整成员角色（需登录）
func (h *RoomHandler) ChangeRole(c *gin.Context) {
	var req contract.ChangeRoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	hostID := c.MustGet("user_id").(uuid.UUID)

	if err := h.svc.ChangeRole(c.Request.Context(), hostID, req); err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, nil)
}

// SelectRom POST /api/rooms/select-rom — 房主选择/切换房间的 ROM（需登录）
func (h *RoomHandler) SelectRom(c *gin.Context) {
	var req contract.SelectRomReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	hostID := c.MustGet("user_id").(uuid.UUID)

	if err := h.svc.SelectRom(c.Request.Context(), hostID, req); err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, nil)
}

// SwitchRom POST /api/rooms/switch-rom — 房主在游戏中热切换 ROM（需登录，仅 playing 状态）
func (h *RoomHandler) SwitchRom(c *gin.Context) {
	var req contract.SwitchRomReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	hostID := c.MustGet("user_id").(uuid.UUID)

	if err := h.svc.SwitchRom(c.Request.Context(), hostID, req); err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, nil)
}

// Start POST /api/rooms/start — 房主启动游戏（需登录）
func (h *RoomHandler) Start(c *gin.Context) {
	var req contract.StartRoomReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	hostID := c.MustGet("user_id").(uuid.UUID)

	resp, err := h.svc.Start(c.Request.Context(), hostID, *req.RoomID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, resp)
}

// GetMembers GET /api/rooms/:id/members — 获取房间成员列表（需登录）
func (h *RoomHandler) GetMembers(c *gin.Context) {
	roomID, err := parseUUIDParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)

	members, err := h.svc.GetMembers(c.Request.Context(), userID, roomID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, members)
}

// KickPlayer POST /api/rooms/kick — 房主踢出玩家（需登录）
func (h *RoomHandler) KickPlayer(c *gin.Context) {
	var req contract.KickPlayerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	hostID := c.MustGet("user_id").(uuid.UUID)

	if err := h.svc.KickPlayer(c.Request.Context(), hostID, req); err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, nil)
}

// GetLivekitToken GET /api/rooms/:id/livekit — 获取 LiveKit token（需登录）
// 房主通过 Start 接口直接返回 token，其他玩家调用此接口轮询获取
// 游戏未开始时返回 { waiting: true }
func (h *RoomHandler) GetLivekitToken(c *gin.Context) {
	roomID, err := parseUUIDParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)

	resp, err := h.svc.GetLivekitToken(c.Request.Context(), userID, roomID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, resp)
}

// Leave POST /api/rooms/leave — 离开房间（需登录）
func (h *RoomHandler) Leave(c *gin.Context) {
	var req contract.LeaveRoomReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)

	if err := h.svc.Leave(c.Request.Context(), userID, *req.RoomID); err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, nil)
}

// Pause POST /api/rooms/pause — 房主暂停游戏（需登录）
func (h *RoomHandler) Pause(c *gin.Context) {
	var req contract.PauseRoomReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	hostID := c.MustGet("user_id").(uuid.UUID)

	if err := h.svc.Pause(c.Request.Context(), hostID, *req.RoomID); err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, nil)
}

// Resume POST /api/rooms/resume — 房主继续游戏（需登录）
func (h *RoomHandler) Resume(c *gin.Context) {
	var req contract.ResumeRoomReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	hostID := c.MustGet("user_id").(uuid.UUID)

	if err := h.svc.Resume(c.Request.Context(), hostID, *req.RoomID); err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, nil)
}

// Stop POST /api/rooms/stop — 房主停止游戏（需登录）
func (h *RoomHandler) Stop(c *gin.Context) {
	var req contract.StopRoomReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	hostID := c.MustGet("user_id").(uuid.UUID)

	if err := h.svc.Stop(c.Request.Context(), hostID, *req.RoomID); err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, nil)
}

// Delete POST /api/rooms/delete — 房主删除房间（需登录）
func (h *RoomHandler) Delete(c *gin.Context) {
	var req contract.DeleteRoomReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	hostID := c.MustGet("user_id").(uuid.UUID)

	if err := h.svc.Delete(c.Request.Context(), hostID, *req.RoomID); err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, nil)
}

// SaveState POST /api/rooms/save-state — 房主保存存档（需登录）
func (h *RoomHandler) SaveState(c *gin.Context) {
	var req contract.SaveStateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	hostID := c.MustGet("user_id").(uuid.UUID)

	ss, err := h.svc.SaveState(c.Request.Context(), hostID, *req.RoomID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, ss)
}

// LoadState POST /api/rooms/load-state — 房主读取存档（需登录）
func (h *RoomHandler) LoadState(c *gin.Context) {
	var req contract.LoadStateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	hostID := c.MustGet("user_id").(uuid.UUID)

	if err := h.svc.LoadState(c.Request.Context(), hostID, req); err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, nil)
}

// ListSaveStates GET /api/rooms/:id/save-states — 列出房间存档（房间成员可查，需登录）
func (h *RoomHandler) ListSaveStates(c *gin.Context) {
	roomID, err := parseUUIDParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)

	states, err := h.svc.ListSaveStates(c.Request.Context(), userID, roomID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, states)
}

// LoadLatestState POST /api/rooms/load-latest-state — 房主加载最新存档（需登录）
func (h *RoomHandler) LoadLatestState(c *gin.Context) {
	var req contract.LoadLatestStateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	hostID := c.MustGet("user_id").(uuid.UUID)

	if err := h.svc.LoadLatestState(c.Request.Context(), hostID, *req.RoomID); err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, nil)
}

// RenameSaveState POST /api/rooms/rename-save-state — 房主重命名存档（需登录）
func (h *RoomHandler) RenameSaveState(c *gin.Context) {
	var req contract.RenameSaveStateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	hostID := c.MustGet("user_id").(uuid.UUID)

	if err := h.svc.RenameSaveState(c.Request.Context(), hostID, req); err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, nil)
}

// DeleteSaveState POST /api/rooms/delete-save-state — 房主删除存档（需登录）
func (h *RoomHandler) DeleteSaveState(c *gin.Context) {
	var req contract.DeleteSaveStateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	hostID := c.MustGet("user_id").(uuid.UUID)

	if err := h.svc.DeleteSaveState(c.Request.Context(), hostID, req); err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, nil)
}
