// 数据传输对象（DTO），定义所有 API 请求/响应结构体

package contract

import (
	"time"

	"github.com/google/uuid"
)

// CaptchaResp 滑块验证码响应（go-captcha-vue 官方组件所需数据格式）
type CaptchaResp struct {
	CaptchaKey     string `json:"captcha_key"`      // 验证码唯一标识
	MasterBgBase64 string `json:"master_bg_base64"` // 主背景图（含缺口）JPEG Base64，对应官方 image 字段
	TileBase64     string `json:"tile_base64"`      // 拼图块 PNG Base64，对应官方 thumb 字段
	ThumbX         int    `json:"thumb_x"`          // 拼图块在背景图上的起始 X 坐标（px）
	ThumbY         int    `json:"thumb_y"`          // 拼图块在背景图上的起始 Y 坐标（px）
	TileWidth      int    `json:"tile_width"`       // 拼图块宽度（px）
	TileHeight     int    `json:"tile_height"`      // 拼图块高度（px）
}

// VerifyCaptchaReq 滑动验证码校验请求（阶段 1：滑动后立即调用）
type VerifyCaptchaReq struct {
	CaptchaKey string `json:"captcha_key" binding:"required"` // 验证码 key，来自 /captcha 接口返回
	SlideX     int    `json:"slide_x"     binding:"required"` // 滑块横向定位（px）
	SlideY     int    `json:"slide_y"     binding:"required"` // 滑块纵向定位（px）
}

// LoginReq 用户登录请求
type LoginReq struct {
	Account    string `json:"account"     binding:"required"` // 登录账号（用户名或邮箱均可）
	Password   string `json:"password"    binding:"required"` // 明文密码
	CaptchaKey string `json:"captcha_key" binding:"required"` // 验证码 key，登录时携带（必须先通过阶段 1 校验）
}

// RegisterReq 用户注册请求
type RegisterReq struct {
	Username   string `json:"username"    binding:"required,min=3,max=64"`  // 用户名，3-64字符
	Email      string `json:"email"       binding:"required,email,max=255"` // 邮箱地址
	Password   string `json:"password"    binding:"required,min=6,max=128"` // 明文密码，6-128字符
	CaptchaKey string `json:"captcha_key" binding:"required"`               // 验证码 key，必须先通过阶段 1 校验
}

// LoginResp 登录成功响应
type LoginResp struct {
	AccessToken  string `json:"access_token"`  // JWT Access Token，24小时有效
	RefreshToken string `json:"refresh_token"` // Refresh Token，7天有效
	ExpiresIn    int64  `json:"expires_in"`    // Access Token 过期倒计时（秒）
}

// TokenPair Token 对，用于刷新时的响应
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

// VerifyEmailReq 邮箱验证请求
type VerifyEmailReq struct {
	Email string `json:"email" binding:"required,email"` // 要验证的邮箱
	Code  string `json:"code"  binding:"required,len=6"` // 6位数字验证码
}

// ResendCodeReq 重发验证码请求
type ResendCodeReq struct {
	Email string `json:"email" binding:"required,email"` // 接收验证码的邮箱
}

// RefreshTokenReq 刷新Token请求
type RefreshTokenReq struct {
	RefreshToken string `json:"refresh_token" binding:"required"` // 当前的RefreshToken
}

// UpdateProfileReq 更新个人信息请求
// 注意：头像通过 multipart/form-data 的 "avatar" 文件字段提交，不在结构体中定义
type UpdateProfileReq struct {
	Nickname *string `json:"nickname" binding:"omitempty,max=64"`  // 昵称，空字符串视为清空
	Bio      *string `json:"bio"      binding:"omitempty,max=512"` // 个人简介，空字符串视为清空
}

// UpdatePasswordReq 修改密码请求
type UpdatePasswordReq struct {
	OldPassword string `json:"old_password" binding:"required,min=6,max=128"` // 当前密码
	NewPassword string `json:"new_password" binding:"required,min=6,max=128"` // 新密码
}

// CreateRoomReq 创建房间请求
type CreateRoomReq struct {
	Title        string      `json:"title"         binding:"required,max=128"`           // 房间名称
	EmulatorType string      `json:"emulator_type" binding:"required,oneof=nes gb dos"` // 模拟器类型：nes / gb / dos
	RomID        *uuid.UUID  `json:"rom_id"`                                             // 要游玩的ROM ID（可选，进入房间后再选择）
	MaxPorts     int16       `json:"max_ports"     binding:"required,min=1,max=4"`       // 最大手柄端口数
	InviteeIDs   []uuid.UUID `json:"invitee_ids"`                                        // 邀请的好友ID列表（可选）
}

// ChangeRoleReq 房主调整成员角色请求
// role=1(玩家)时 port 必填，后端分配/转移端口；role=2(旁观)时无需 port，后端自动查询已有端口并处理
type ChangeRoleReq struct {
	RoomID *uuid.UUID `json:"room_id" binding:"required,notnil_uuid"` // 房间ID
	UserID *uuid.UUID `json:"user_id" binding:"required,notnil_uuid"` // 目标玩家ID
	Role   int16      `json:"role"    binding:"required,oneof=1 2"`   // 目标角色：1=玩家, 2=旁观
	Port   *int16     `json:"port"`                                    // 手柄端口号（role=1 时必填，0-based）
}

// UploadRomReq ROM 上传请求
// 注意：ROM文件通过 multipart/form-data 的 "rom" 字段提交，不在结构体中定义
type UploadRomReq struct {
	Title string `form:"title" binding:"required"` // ROM 标题
}

// UpdateRomReq ROM 更新请求
// 注意：封面图片通过 multipart/form-data 的 "cover" 字段提交，不在结构体中定义
type UpdateRomReq struct {
	Title *string `form:"title" binding:"required"` // ROM 标题
}

// DeleteRomReq 删除 ROM 请求
type DeleteRomReq struct {
	RomID *uuid.UUID `json:"rom_id" binding:"required,notnil_uuid"` // 要删除的 ROM ID
}

// FriendAddReq 添加好友请求
type FriendAddReq struct {
	FriendID *uuid.UUID `json:"friend_id" binding:"required,notnil_uuid"` // 要添加的好友用户ID
}

// FriendAcceptReq 接受好友请求
type FriendAcceptReq struct {
	FriendID *uuid.UUID `json:"friend_id" binding:"required,notnil_uuid"` // 好友申请发起方用户ID
}

// InviteToRoomReq 邀请好友加入房间请求
type InviteToRoomReq struct {
	RoomID     *uuid.UUID  `json:"room_id"     binding:"required,notnil_uuid"` // 目标房间ID
	InviteeIDs []uuid.UUID `json:"invitee_ids" binding:"required,min=1"`       // 被邀请的好友ID列表
}

// SelectRomReq 选择/切换房间 ROM 请求
type SelectRomReq struct {
	RoomID *uuid.UUID `json:"room_id" binding:"required,notnil_uuid"` // 房间ID
	RomID  *uuid.UUID `json:"rom_id"  binding:"required,notnil_uuid"` // 要切换的 ROM ID
}

// SwitchRomReq 游戏中热切换 ROM 请求（仅在 room.Status=1 playing 状态下可用）
type SwitchRomReq struct {
	RoomID *uuid.UUID `json:"room_id" binding:"required,notnil_uuid"` // 房间ID
	RomID  *uuid.UUID `json:"rom_id"  binding:"required,notnil_uuid"` // 要切换的 ROM ID
}

// StartRoomReq 开始游戏请求
type StartRoomReq struct {
	RoomID *uuid.UUID `json:"room_id" binding:"required,notnil_uuid"` // 要开始的房间ID
}

// LeaveRoomReq 离开房间请求
type LeaveRoomReq struct {
	RoomID *uuid.UUID `json:"room_id" binding:"required,notnil_uuid"` // 要离开的房间ID
}

// PauseRoomReq 暂停游戏请求
type PauseRoomReq struct {
	RoomID *uuid.UUID `json:"room_id" binding:"required,notnil_uuid"` // 要暂停的房间ID
}

// ResumeRoomReq 继续游戏请求
type ResumeRoomReq struct {
	RoomID *uuid.UUID `json:"room_id" binding:"required,notnil_uuid"` // 要继续的房间ID
}

// StopRoomReq 停止游戏请求
type StopRoomReq struct {
	RoomID *uuid.UUID `json:"room_id" binding:"required,notnil_uuid"` // 要停止的房间ID
}

// DeleteRoomReq 删除房间请求
type DeleteRoomReq struct {
	RoomID *uuid.UUID `json:"room_id" binding:"required,notnil_uuid"` // 要删除的房间ID
}

// SaveStateReq 保存存档请求
type SaveStateReq struct {
	RoomID *uuid.UUID `json:"room_id" binding:"required,notnil_uuid"` // 要保存存档的房间ID
}

// LoadStateReq 读取存档请求
type LoadStateReq struct {
	RoomID      *uuid.UUID `json:"room_id"       binding:"required,notnil_uuid"` // 房间ID
	SaveStateID *uuid.UUID `json:"save_state_id" binding:"required,notnil_uuid"` // 要读取的存档ID
}

// LoadLatestStateReq 加载最新存档请求
type LoadLatestStateReq struct {
	RoomID *uuid.UUID `json:"room_id" binding:"required,notnil_uuid"` // 房间ID
}

// RenameSaveStateReq 重命名存档请求
type RenameSaveStateReq struct {
	RoomID      *uuid.UUID `json:"room_id"       binding:"required,notnil_uuid"`      // 房间ID
	SaveStateID *uuid.UUID `json:"save_state_id" binding:"required,notnil_uuid"`      // 要重命名的存档ID
	Name        string     `json:"name"          binding:"required,min=1,max=64"`     // 新名称
}

// DeleteSaveStateReq 删除存档请求
type DeleteSaveStateReq struct {
	RoomID      *uuid.UUID `json:"room_id"       binding:"required,notnil_uuid"` // 房间ID
	SaveStateID *uuid.UUID `json:"save_state_id" binding:"required,notnil_uuid"` // 要删除的存档ID
}

// LivekitTokenResp 查询 LiveKit token 响应
// 游戏未开始时返回 { waiting: true }，游戏进行中返回 { livekit_token, livekit_room, livekit_url }
type LivekitTokenResp struct {
	LivekitToken string `json:"livekit_token,omitempty"` // LiveKit 访问 token
	LivekitRoom  string `json:"livekit_room,omitempty"`  // LiveKit 房间名
	LivekitUrl   string `json:"livekit_url,omitempty"`   // LiveKit 服务端地址
	Waiting      bool   `json:"waiting"`                 // 是否等待游戏开始
}

// StartRoomResp 启动游戏响应
type StartRoomResp struct {
	LivekitToken string `json:"livekit_token"` // LiveKit access token，前端用此 token 连接推流
	LivekitRoom  string `json:"livekit_room"`  // LiveKit 房间名
	LivekitUrl   string `json:"livekit_url"`   // LiveKit 服务端地址
}

// RoomMemberInfo 房间成员信息，含用户信息
type RoomMemberInfo struct {
	UserID   uuid.UUID `json:"user_id"`
	Username string    `json:"username"`
	Nickname *string   `json:"nickname"`
	Avatar   *string   `json:"avatar"`
	Role     int16     `json:"role"`
	Port     *int16    `json:"port"`
}

// KickPlayerReq 踢出玩家请求
type KickPlayerReq struct {
	RoomID *uuid.UUID `json:"room_id" binding:"required,notnil_uuid"`
	UserID *uuid.UUID `json:"user_id" binding:"required,notnil_uuid"`
}

// FriendRejectReq 拒绝好友申请请求
type FriendRejectReq struct {
	FriendID *uuid.UUID `json:"friend_id" binding:"required,notnil_uuid"` // 好友申请发起方用户ID
}

// FriendWithUser 好友列表项，含好友的用户信息
type FriendWithUser struct {
	ID         uuid.UUID  `json:"id"`
	UserID     uuid.UUID  `json:"user_id"`
	FriendID   uuid.UUID  `json:"friend_id"`
	Status     int16      `json:"status"`
	AcceptedAt *time.Time `json:"accepted_at"`
	CreatedAt  time.Time  `json:"created_at"`
	Username   string     `json:"username"`
	Nickname   *string    `json:"nickname"`
	Avatar     *string    `json:"avatar"`
}

// FriendListResp 好友列表响应
type FriendListResp struct {
	Friends []FriendWithUser `json:"friends"`
}

// UserSearchItem 用户搜索结果项
type UserSearchItem struct {
	ID       string  `json:"id"`
	Username string  `json:"username"`
	Nickname *string `json:"nickname"`
	Avatar   *string `json:"avatar"`
}

// UserSearchResp 用户搜索响应
type UserSearchResp struct {
	Users []UserSearchItem `json:"users"`
}

// UserProfile 用户公开信息，用于查看其他用户资料
type UserProfile struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
	Nickname *string   `json:"nickname"`
	Avatar   *string   `json:"avatar"`
	Bio      *string   `json:"bio"`
}

// FriendPendingItem 待处理好友请求项，含发起者用户信息
type FriendPendingItem struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	FriendID  uuid.UUID `json:"friend_id"`
	CreatedAt time.Time `json:"created_at"`
	Username  string    `json:"username"`
	Nickname  *string   `json:"nickname"`
	Avatar    *string   `json:"avatar"`
}

// FriendPendingListResp 待处理好友请求列表响应
type FriendPendingListResp struct {
	Pending []FriendPendingItem `json:"pending"`
}

// ForgotPasswordReq 忘记密码请求
type ForgotPasswordReq struct {
	Email      string `json:"email"       binding:"required,email,max=255"` // 要重置密码的邮箱地址
	CaptchaKey string `json:"captcha_key" binding:"required"`               // 验证码 key，必须先通过阶段 1 校验
}

// ResetPasswordReq 重置密码请求
type ResetPasswordReq struct {
	Token       string `json:"token"        binding:"required"`               // 从邮件链接中获取的原始重置 token
	NewPassword string `json:"new_password" binding:"required,min=6,max=128"` // 新密码，6-128字符
}

const (
	AccessTokenTTL  = 24 * time.Hour     // Access Token 有效期：24小时
	RefreshTokenTTL = 7 * 24 * time.Hour // Refresh Token 有效期：7天
)
