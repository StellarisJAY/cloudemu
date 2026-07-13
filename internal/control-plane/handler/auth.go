package handler

import (
	"io"

	"github.com/StellarisJAY/cloudemu/internal/control-plane/contract"
	"github.com/StellarisJAY/cloudemu/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AuthHandler 认证相关 HTTP 处理器，薄层，仅做参数绑定和响应序列化
type AuthHandler struct {
	svc contract.AuthService
}

// NewAuthHandler 创建 AuthHandler 实例
func NewAuthHandler(svc contract.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// Register POST /api/auth/register — 用户注册
func (h *AuthHandler) Register(c *gin.Context) {
	var req contract.RegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	user, err := h.svc.Register(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, user)
}

// Login POST /api/auth/login — 用户登录
func (h *AuthHandler) Login(c *gin.Context) {
	var req contract.LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	resp, err := h.svc.Login(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, resp)
}

// VerifyEmail POST /api/auth/verify-email — 邮箱验证激活
func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	var req contract.VerifyEmailReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	if err := h.svc.VerifyEmail(c.Request.Context(), req); err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, nil)
}

// ResendCode POST /api/auth/resend-code — 重发验证码
func (h *AuthHandler) ResendCode(c *gin.Context) {
	var req contract.ResendCodeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	if err := h.svc.ResendCode(c.Request.Context(), req); err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, nil)
}

// RefreshToken POST /api/auth/refresh — 刷新Token
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req contract.RefreshTokenReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	pair, err := h.svc.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, pair)
}

// Captcha GET /api/auth/captcha — 获取图形验证码
func (h *AuthHandler) Captcha(c *gin.Context) {
	resp, err := h.svc.Captcha(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, resp)
}

// VerifyCaptcha POST /api/auth/captcha/verify — 校验滑块验证码（阶段1：滑动后立即调用）
func (h *AuthHandler) VerifyCaptcha(c *gin.Context) {
	var req contract.VerifyCaptchaReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	if err := h.svc.VerifyCaptcha(c.Request.Context(), req); err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, nil)
}

// Me GET /api/auth/me — 获取当前用户信息（需登录）
func (h *AuthHandler) Me(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	user, err := h.svc.Me(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, user)
}

// Search GET /api/users/search — 按用户名模糊搜索其他用户（需登录）
func (h *AuthHandler) Search(c *gin.Context) {
	query := c.Query("q")
	userID := c.MustGet("user_id").(uuid.UUID)

	users, err := h.svc.Search(c.Request.Context(), query, userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, gin.H{"users": users})
}

// GetUser GET /api/users/:id — 获取指定用户的公开信息（需登录）
func (h *AuthHandler) GetUser(c *gin.Context) {
	targetID, err := parseUUIDParam(c, "id")
	if err != nil {
		response.BadRequest(c, "无效的用户ID")
		return
	}
	userID := c.MustGet("user_id").(uuid.UUID)

	profile, err := h.svc.GetUserProfile(c.Request.Context(), userID, targetID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, profile)
}

// UpdateProfile PUT /api/auth/profile — 更新个人信息（需登录，multipart/form-data）
// 昵称/简介通过表单字段提交，头像通过 "avatar" 文件字段提交（均可选）
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	nickname := c.PostForm("nickname")
	var nicknamePtr *string
	if nickname != "" {
		nicknamePtr = &nickname
	}

	bio := c.PostForm("bio")
	var bioPtr *string
	if bio != "" {
		bioPtr = &bio
	}

	// 获取头像文件（可选）
	var avatarFile io.Reader
	var avatarFileName string
	file, header, err := c.Request.FormFile("avatar")
	if err == nil {
		defer file.Close()
		avatarFile = file
		avatarFileName = header.Filename
	}

	req := contract.UpdateProfileReq{
		Nickname: nicknamePtr,
		Bio:      bioPtr,
	}

	user, err := h.svc.UpdateProfile(c.Request.Context(), userID, req, avatarFile, avatarFileName)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, user)
}

// UpdatePassword PUT /api/auth/password — 修改密码（需登录）
func (h *AuthHandler) UpdatePassword(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	var req contract.UpdatePasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	if err := h.svc.UpdatePassword(c.Request.Context(), userID, req); err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, nil)
}

// ForgotPassword POST /api/auth/forgot-password — 请求密码重置
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req contract.ForgotPasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	if err := h.svc.ForgotPassword(c.Request.Context(), req); err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, nil) // 无论邮箱是否存在都返回成功，防止用户枚举
}

// ResetPassword POST /api/auth/reset-password — 执行密码重置
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req contract.ResetPasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	if err := h.svc.ResetPassword(c.Request.Context(), req); err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, nil)
}
