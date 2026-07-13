package apperror

// AppError 统一业务错误类型，包含业务错误码、错误消息和对应 HTTP 状态码
type AppError struct {
	Code    int    // 业务错误码，按模块分段：1xxx=认证, 2xxx=房间, 3xxx=ROM, 4xxx=好友/通用
	Message string // 用户可读的错误消息
	HTTP    int    // 对应的 HTTP 状态码
}

func (e *AppError) Error() string   { return e.Message }
func (e *AppError) HTTPStatus() int { return e.HTTP }

// ---- 通用错误 ----
var (
	ErrInternal     = &AppError{5000, "服务器内部错误", 500}
	ErrUnauthorized = &AppError{4001, "未登录或 token 无效", 401}
	ErrBadRequest   = &AppError{4000, "请求参数错误", 400}
	ErrInvalidParam = &AppError{4002, "无效的请求参数", 400}
)

// ---- 认证模块错误 (1001-1010) ----
var (
	ErrUserExists          = &AppError{1001, "用户名或邮箱已存在", 409}
	ErrInvalidCaptcha      = &AppError{1002, "验证码错误", 400}
	ErrCaptchaExpired      = &AppError{1003, "验证码已过期", 400}
	ErrInvalidCredentials  = &AppError{1004, "用户名或密码错误", 400}
	ErrUserNotActive       = &AppError{1005, "账户未激活，请先验证邮箱", 403}
	ErrTooManyAttempts     = &AppError{1006, "尝试次数过多，请稍后再试", 429}
	ErrResendCooldown      = &AppError{1007, "发送过于频繁，请 60 秒后再试", 429}
	ErrInvalidCode         = &AppError{1008, "验证码错误或已过期", 400}
	ErrRefreshTokenExpired = &AppError{1009, "refresh token 无效或已过期", 401}
	ErrUserNotFound        = &AppError{1010, "用户不存在", 404}
	ErrCaptchaNotVerified  = &AppError{1011, "验证码未验证，请先完成安全验证", 400}
	ErrResetTokenInvalid   = &AppError{1012, "重置链接无效或已过期", 400}
	ErrResetTokenUsed      = &AppError{1013, "该重置链接已被使用", 400}
)

// ---- 房间模块错误 (2001-2015) ----
var (
	ErrRoomNotExist   = &AppError{2001, "房间不存在", 404}
	ErrNotRoomHost    = &AppError{2002, "仅房主可执行此操作", 403}
	ErrPortOccupied   = &AppError{2003, "该手柄已被占用", 409}
	ErrRoomFull       = &AppError{2005, "房间已满", 409}
	ErrRoomNotWaiting = &AppError{2006, "房间不在等待状态", 400}
	ErrRoomClosed     = &AppError{2007, "房间已关闭", 410}
	ErrAlreadyInRoom  = &AppError{2008, "你已经在该房间中", 409}
	ErrNotInRoom      = &AppError{2009, "你不在该房间中", 403}
	ErrNotFriend      = &AppError{2010, "只能邀请好友", 403}
	ErrPortInvalid    = &AppError{2011, "无效的手柄端口", 400}
	ErrKickSelf       = &AppError{2012, "不能踢出自己", 400}
	ErrKickHost       = &AppError{2013, "不能踢出房主", 400}
	ErrRomNotSelected = &AppError{2014, "房间尚未选择 ROM，无法开始游戏", 400}
	ErrRoomNotPlaying = &AppError{2015, "手柄分配仅限游戏中", 400}
)

// ---- ROM 模块错误 (3001-3005) ----
var (
	ErrRomNotExist      = &AppError{3001, "ROM 不存在", 404}
	ErrRomDuplicate     = &AppError{3002, "该 ROM 文件已上传", 409}
	ErrRomTooLarge      = &AppError{3003, "ROM 文件过大", 400}
	ErrRomInvalidFormat = &AppError{3004, "ROM 文件格式不正确", 400}
	ErrRomTypeMismatch  = &AppError{3005, "ROM 模拟器类型与房间不匹配", 400}
)

// ---- 好友模块错误 (4001-4004) ----
var (
	ErrFriendSelf           = &AppError{4001, "不能添加自己为好友", 400}
	ErrFriendExists         = &AppError{4002, "好友关系已存在", 409}
	ErrFriendNotFound       = &AppError{4003, "好友关系不存在", 404}
	ErrFriendAlreadyHandled = &AppError{4004, "该好友请求已被处理", 400}
)

// ---- Worker 调度错误 (5001-5002) ----
var (
	ErrNoAvailableWorker = &AppError{5001, "暂无可用游戏节点", 503}
	ErrWorkerUnavailable = &AppError{5002, "游戏节点不可用", 502}
)
