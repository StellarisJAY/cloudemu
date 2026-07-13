package contract

import (
	"context"
	"io"
	"time"

	"github.com/StellarisJAY/cloudemu/internal/control-plane/model"
	"github.com/google/uuid"
)

// AuthService 认证业务逻辑接口
type AuthService interface {
	Register(ctx context.Context, req RegisterReq) (*model.User, error)                                                                          // 用户注册，创建待激活账户并发送邮箱验证码
	Login(ctx context.Context, req LoginReq) (*LoginResp, error)                                                                                 // 用户登录，校验验证码状态+账号密码，返回JWT双Token
	VerifyEmail(ctx context.Context, req VerifyEmailReq) error                                                                                   // 校验邮箱验证码，激活账户
	ResendCode(ctx context.Context, req ResendCodeReq) error                                                                                     // 重发邮箱验证码（60秒冷却）
	RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error)                                                                   // 使用RefreshToken刷新，旧Token作废，返回新Token对
	Captcha(ctx context.Context) (*CaptchaResp, error)                                                                                           // 生成图形验证码，返回key+base64图片
	VerifyCaptcha(ctx context.Context, req VerifyCaptchaReq) error                                                                               // 校验滑块验证码（阶段1），通过后记录captcha_verified标记
	Me(ctx context.Context, userID uuid.UUID) (*model.User, error)                                                                               // 获取当前登录用户信息
	Search(ctx context.Context, query string, userID uuid.UUID) ([]UserSearchItem, error)                                                        // 按用户名模糊搜索其他用户（排除自己）
	GetUserProfile(ctx context.Context, userID uuid.UUID, targetID uuid.UUID) (*UserProfile, error)                                              // 获取指定用户的公开信息（头像/昵称/简介）
	UpdateProfile(ctx context.Context, userID uuid.UUID, req UpdateProfileReq, avatarFile io.Reader, avatarFileName string) (*model.User, error) // 更新个人信息（昵称/简介/头像）
	UpdatePassword(ctx context.Context, userID uuid.UUID, req UpdatePasswordReq) error                                                           // 修改密码（需验证旧密码）
	ForgotPassword(ctx context.Context, req ForgotPasswordReq) error                                                                             // 忘记密码，生成重置 token 并发送重置邮件
	ResetPassword(ctx context.Context, req ResetPasswordReq) error                                                                               // 使用重置 token 设置新密码
}

// UserRepo 用户表数据访问接口
type UserRepo interface {
	Create(ctx context.Context, user *model.User) error                                                 // 插入新用户
	ByID(ctx context.Context, id uuid.UUID) (*model.User, error)                                        // 按ID查询用户
	ByEmail(ctx context.Context, email string) (*model.User, error)                                     // 按邮箱查询用户
	ByUsername(ctx context.Context, username string) (*model.User, error)                               // 按用户名查询用户
	Search(ctx context.Context, query string, excludeID uuid.UUID, limit int) ([]UserSearchItem, error) // 模糊搜索用户（排除自己，限制数量）
	UpdateStatus(ctx context.Context, id uuid.UUID, status int16) error                                 // 更新用户状态（激活/禁用）
	UpdateLastLogin(ctx context.Context, id uuid.UUID) error                                            // 更新最近登录时间
	UpdateProfile(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error              // 更新用户个人资料字段（nickname/bio/avatar等）
}

// EmailVerificationRepo 邮箱验证码记录数据访问接口
type EmailVerificationRepo interface {
	Create(ctx context.Context, record *model.EmailVerification) error                 // 插入验证码记录
	LatestByEmail(ctx context.Context, email string) (*model.EmailVerification, error) // 查某邮箱最新一条验证码记录
	MarkVerified(ctx context.Context, id uuid.UUID) error                              // 标记验证码为已验证
}

// RefreshTokenRepo 刷新令牌数据访问接口
type RefreshTokenRepo interface {
	Create(ctx context.Context, token *model.RefreshToken) error          // 插入刷新令牌记录
	ByHash(ctx context.Context, hash string) (*model.RefreshToken, error) // 按Token哈希查询
	DeleteByUser(ctx context.Context, userID uuid.UUID) error             // 删除某用户所有刷新令牌
	DeleteByHash(ctx context.Context, hash string) error                  // 按哈希删除单条刷新令牌
}

// PasswordResetRepo 密码重置记录数据访问接口
type PasswordResetRepo interface {
	Create(ctx context.Context, record *model.PasswordReset) error         // 插入重置记录
	ByHash(ctx context.Context, hash string) (*model.PasswordReset, error) // 按 token_hash 查询
	MarkUsed(ctx context.Context, id uuid.UUID) error                      // 标记 token 已使用
}

// SlideCaptchaData 滑块验证码目标坐标（序列化到 Redis JSON）
// 注意：Y 固定为 0，因为前端滑块仅在水平方向移动
type SlideCaptchaData struct {
	TargetX int `json:"tx"`
	TargetY int `json:"ty"`
}

// CaptchaCache 验证码缓存接口（Redis）
type CaptchaCache interface {
	Set(ctx context.Context, key string, data *SlideCaptchaData, ttl time.Duration) error // 存储验证码目标坐标，带TTL
	GetAndDel(ctx context.Context, key string) (*SlideCaptchaData, error)                 // 获取并删除验证码数据（阶段1一次性使用）
	SetVerified(ctx context.Context, key string, ttl time.Duration) error                 // 标记验证码已通过校验（阶段1成功后写入）
	ConsumeVerified(ctx context.Context, key string) (bool, error)                        // 检查并删除验证通过标记（阶段2一次性消费）
}

// EmailSender 邮件发送接口，由 email.SMTPSender 或 email.NoopSender 实现
// SMTP 未配置时使用 NoopSender（slog.Debug 打印邮件内容），生产环境配置 SMTP 后使用 SMTPSender
type EmailSender interface {
	Send(ctx context.Context, to, subject, body string) error // 向指定邮箱发送邮件，subject 和 body 支持中文
}
