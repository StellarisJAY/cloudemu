package handler

import (
	"github.com/StellarisJAY/cloudemu/internal/control-plane/contract"
	"github.com/StellarisJAY/cloudemu/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// FriendHandler 好友关系相关 HTTP 处理器
type FriendHandler struct {
	svc contract.FriendService
}

// NewFriendHandler 创建 FriendHandler 实例
func NewFriendHandler(svc contract.FriendService) *FriendHandler {
	return &FriendHandler{svc: svc}
}

// Add POST /api/friends/add — 发送好友申请（需登录）
func (h *FriendHandler) Add(c *gin.Context) {
	var req contract.FriendAddReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)

	if err := h.svc.Add(c.Request.Context(), userID, *req.FriendID); err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, nil)
}

// Accept POST /api/friends/accept — 接受好友申请（需登录）
func (h *FriendHandler) Accept(c *gin.Context) {
	var req contract.FriendAcceptReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)

	if err := h.svc.Accept(c.Request.Context(), userID, *req.FriendID); err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, nil)
}

// List GET /api/friends — 列出当前用户所有好友（需登录，含好友的用户信息）
func (h *FriendHandler) List(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	friends, err := h.svc.List(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, gin.H{"friends": friends})
}

// Pending GET /api/friends/pending — 列出当前用户收到的待处理好友请求（需登录）
func (h *FriendHandler) Pending(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	pending, err := h.svc.Pending(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, gin.H{"pending": pending})
}

// Reject POST /api/friends/reject — 拒绝好友申请（需登录）
func (h *FriendHandler) Reject(c *gin.Context) {
	var req contract.FriendRejectReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)

	if err := h.svc.Reject(c.Request.Context(), userID, *req.FriendID); err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, nil)
}
