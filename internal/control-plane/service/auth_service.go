package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"path/filepath"
	"time"

	"github.com/StellarisJAY/cloudemu/internal/control-plane/contract"
	"github.com/StellarisJAY/cloudemu/internal/control-plane/model"
	"github.com/StellarisJAY/cloudemu/internal/pkg/apperror"
	jwtutil "github.com/StellarisJAY/cloudemu/internal/pkg/jwt"

	"github.com/google/uuid"
	"github.com/wenlng/go-captcha/v2/slide"
	"golang.org/x/crypto/bcrypt"
)

// AuthService 认证业务逻辑实现
type AuthService struct {
	userRepo              contract.UserRepo
	emailVerificationRepo contract.EmailVerificationRepo
	refreshTokenRepo      contract.RefreshTokenRepo
	passwordResetRepo     contract.PasswordResetRepo
	captchaCache          contract.CaptchaCache
	slideCaptcha          slide.Captcha
	limiter               Limiter
	jwtSecret             []byte
	minioFunc             contract.MinioFunc
	minioBucket           string
	emailSender           contract.EmailSender
	frontendBaseURL       string
}

// Limiter 频率限制接口（由 cache/Limiter 实现）
type Limiter interface {
	IncrLoginAttempt(ctx context.Context, ip string) (int64, error)      // 增加登录失败计数
	IsLoginLocked(ctx context.Context, ip string) (bool, error)          // 检查登录是否被锁定
	IncrVerifyAttempt(ctx context.Context, email string) (int64, error)  // 增加验证尝试计数
	IsVerifyLocked(ctx context.Context, email string) (bool, error)      // 检查验证是否被锁定
	CheckResendCooldown(ctx context.Context, email string) (bool, error) // 检查重发冷却期
	SetResendCooldown(ctx context.Context, email string) error           // 设置重发冷却标记
	IncrForgotPassword(ctx context.Context, ip string) (int64, error)    // 增加忘记密码尝试计数
	IsForgotPasswordLocked(ctx context.Context, ip string) (bool, error) // 检查忘记密码是否被锁定
	CheckForgotCooldown(ctx context.Context, email string) (bool, error) // 检查忘记密码邮件重发冷却期
	SetForgotCooldown(ctx context.Context, email string) error           // 设置忘记密码邮件重发冷却标记
}

// NewAuthService 创建 AuthService 实例
func NewAuthService(
	userRepo contract.UserRepo,
	emailVerificationRepo contract.EmailVerificationRepo,
	refreshTokenRepo contract.RefreshTokenRepo,
	passwordResetRepo contract.PasswordResetRepo,
	captchaCache contract.CaptchaCache,
	slideCaptcha slide.Captcha,
	limiter Limiter,
	jwtSecret []byte,
	minioFunc contract.MinioFunc,
	minioBucket string,
	emailSender contract.EmailSender,
	frontendBaseURL string,
) *AuthService {
	return &AuthService{
		userRepo:              userRepo,
		emailVerificationRepo: emailVerificationRepo,
		refreshTokenRepo:      refreshTokenRepo,
		passwordResetRepo:     passwordResetRepo,
		captchaCache:          captchaCache,
		slideCaptcha:          slideCaptcha,
		limiter:               limiter,
		jwtSecret:             jwtSecret,
		minioFunc:             minioFunc,
		minioBucket:           minioBucket,
		emailSender:           emailSender,
		frontendBaseURL:       frontendBaseURL,
	}
}

// Register 用户注册
// 流程：滑块验证码校验 → 校验用户名/邮箱唯一 → bcrypt加密密码 → 插入users(status=0) → 生成6位验证码 → 插入email_verifications
func (s *AuthService) Register(ctx context.Context, req contract.RegisterReq) (*model.User, error) {
	verified, err := s.captchaCache.ConsumeVerified(ctx, req.CaptchaKey)
	if err != nil || !verified {
		return nil, apperror.ErrCaptchaNotVerified
	}

	existing, _ := s.userRepo.ByEmail(ctx, req.Email)
	if existing != nil {
		return nil, apperror.ErrUserExists
	}
	existing, _ = s.userRepo.ByUsername(ctx, req.Username)
	if existing != nil {
		return nil, apperror.ErrUserExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, apperror.ErrInternal
	}

	user := &model.User{
		ID:           uuid.Must(uuid.NewV7()),
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hash),
		Status:       0,
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, apperror.ErrInternal
	}

	code, err := s.generateCode()
	if err != nil {
		return nil, apperror.ErrInternal
	}

	verification := &model.EmailVerification{
		ID:               uuid.Must(uuid.NewV7()),
		UserID:           user.ID,
		Email:            req.Email,
		VerificationCode: code,
		ExpiresAt:        time.Now().Add(15 * time.Minute),
	}
	if err := s.emailVerificationRepo.Create(ctx, verification); err != nil {
		return nil, apperror.ErrInternal
	}

	_ = s.limiter.SetResendCooldown(ctx, req.Email)

	// 异步发送邮箱验证码
	email := req.Email
	go func() {
		if err := s.emailSender.Send(context.Background(),
			email,
			"CloudEmu 邮箱验证",
			fmt.Sprintf("您的验证码是：%s\n有效期 15 分钟，请尽快完成验证。", code),
		); err != nil {
			slog.Error("failed to send verification email", "error", err, "to", email)
		}
	}()

	return user, nil
}

// Login 用户登录
// 流程：频率限制检查 → 校验验证码(一次性消费) → 查用户(支持邮箱/用户名) → bcrypt密码比对 → 检查账户状态 → 生成JWT双Token
func (s *AuthService) Login(ctx context.Context, req contract.LoginReq) (*contract.LoginResp, error) {
	locked, _ := s.limiter.IsLoginLocked(ctx, req.Account)
	if locked {
		return nil, apperror.ErrTooManyAttempts
	}

	// 校验滑块验证码状态（阶段2：必须已完成阶段1校验）
	verified, err := s.captchaCache.ConsumeVerified(ctx, req.CaptchaKey)
	if err != nil || !verified {
		return nil, apperror.ErrCaptchaNotVerified
	}

	user, err := s.userRepo.ByEmail(ctx, req.Account)
	if user == nil {
		user, err = s.userRepo.ByUsername(ctx, req.Account)
	}
	if user == nil || err != nil {
		if _, ierr := s.limiter.IncrLoginAttempt(ctx, req.Account); ierr != nil {
			// ignore limiter error
		}
		return nil, apperror.ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		if _, ierr := s.limiter.IncrLoginAttempt(ctx, req.Account); ierr != nil {
			// ignore
		}
		return nil, apperror.ErrInvalidCredentials
	}

	if user.Status != 1 {
		return nil, apperror.ErrUserNotActive
	}

	pair, err := s.generateTokenPair(ctx, user.ID, user.Username)
	if err != nil {
		return nil, apperror.ErrInternal
	}

	_ = s.userRepo.UpdateLastLogin(ctx, user.ID)

	return &contract.LoginResp{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresIn:    pair.ExpiresIn,
	}, nil
}

// VerifyEmail 邮箱验证
// 流程：频率检查 → 查最新验证码记录 → 校验code是否匹配且未过期未使用 → 激活用户(status=1) → 标记验证码已使用
func (s *AuthService) VerifyEmail(ctx context.Context, req contract.VerifyEmailReq) error {
	locked, _ := s.limiter.IsVerifyLocked(ctx, req.Email)
	if locked {
		return apperror.ErrTooManyAttempts
	}

	record, err := s.emailVerificationRepo.LatestByEmail(ctx, req.Email)
	if err != nil || record == nil {
		if _, ierr := s.limiter.IncrVerifyAttempt(ctx, req.Email); ierr != nil {
			// ignore
		}
		return apperror.ErrInvalidCode
	}

	if record.VerifiedAt != nil {
		return nil
	}

	if record.VerificationCode != req.Code || time.Now().After(record.ExpiresAt) {
		if _, ierr := s.limiter.IncrVerifyAttempt(ctx, req.Email); ierr != nil {
			// ignore
		}
		return apperror.ErrInvalidCode
	}

	if err := s.userRepo.UpdateStatus(ctx, record.UserID, 1); err != nil {
		return apperror.ErrInternal
	}

	if err := s.emailVerificationRepo.MarkVerified(ctx, record.ID); err != nil {
		return apperror.ErrInternal
	}

	return nil
}

// ResendCode 重发邮箱验证码
// 流程：冷却检查(60秒) → 查用户是否存在且状态为pending → 生成新验证码 → 插入新记录 → 设置冷却
func (s *AuthService) ResendCode(ctx context.Context, req contract.ResendCodeReq) error {
	onCooldown, _ := s.limiter.CheckResendCooldown(ctx, req.Email)
	if onCooldown {
		return apperror.ErrResendCooldown
	}

	user, err := s.userRepo.ByEmail(ctx, req.Email)
	if err != nil || user == nil {
		return apperror.ErrUserNotFound
	}

	if user.Status != 0 {
		return nil
	}

	code, err := s.generateCode()
	if err != nil {
		return apperror.ErrInternal
	}

	verification := &model.EmailVerification{
		ID:               uuid.Must(uuid.NewV7()),
		UserID:           user.ID,
		Email:            req.Email,
		VerificationCode: code,
		ExpiresAt:        time.Now().Add(15 * time.Minute),
	}
	if err := s.emailVerificationRepo.Create(ctx, verification); err != nil {
		return apperror.ErrInternal
	}

	_ = s.limiter.SetResendCooldown(ctx, req.Email)

	// 异步发送邮箱验证码
	email := req.Email
	go func() {
		if err := s.emailSender.Send(context.Background(),
			email,
			"CloudEmu 邮箱验证",
			fmt.Sprintf("您的验证码是：%s\n有效期 15 分钟，请尽快完成验证。", code),
		); err != nil {
			slog.Error("failed to send verification email", "error", err, "to", email)
		}
	}()

	return nil
}

// RefreshToken 刷新 Access Token（轮换制）
// 流程：对Token做SHA-256 → 查refresh_tokens表 → 校验未过期 → 删除旧记录 → 生成新Token对
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*contract.TokenPair, error) {
	hash := sha256Hex(refreshToken)

	record, err := s.refreshTokenRepo.ByHash(ctx, hash)
	if err != nil || record == nil {
		return nil, apperror.ErrRefreshTokenExpired
	}

	if time.Now().After(record.ExpiresAt) {
		_ = s.refreshTokenRepo.DeleteByHash(ctx, hash)
		return nil, apperror.ErrRefreshTokenExpired
	}

	if err := s.refreshTokenRepo.DeleteByHash(ctx, hash); err != nil {
		return nil, apperror.ErrInternal
	}

	user, err := s.userRepo.ByID(ctx, record.UserID)
	if err != nil || user == nil {
		return nil, apperror.ErrUserNotFound
	}

	return s.generateTokenPair(ctx, user.ID, user.Username)
}

// Captcha 生成滑块拼图验证码
// 返回 go-captcha-vue 官方组件所需数据：captcha_key + image/thumb 图片 + thumbX/thumbY 拼图块坐标 + 尺寸
func (s *AuthService) Captcha(ctx context.Context) (*contract.CaptchaResp, error) {
	captData, err := s.slideCaptcha.Generate()
	if err != nil {
		return nil, apperror.ErrInternal
	}

	block := captData.GetData()
	if block == nil {
		return nil, apperror.ErrInternal
	}

	masterBase64, err := captData.GetMasterImage().ToBase64()
	if err != nil {
		return nil, apperror.ErrInternal
	}

	tileBase64, err := captData.GetTileImage().ToBase64()
	if err != nil {
		return nil, apperror.ErrInternal
	}

	key := uuid.Must(uuid.NewV7()).String()

	// 存储目标坐标到 Redis（阶段1校验用），TTL 5 分钟
	// TargetY 使用 hole 的 Y 坐标（基本模式下拼图块与 hole 同 Y 轴，仅水平滑动）
	targetData := &contract.SlideCaptchaData{
		TargetX: block.X,
		TargetY: block.Y,
	}
	if err := s.captchaCache.Set(ctx, key, targetData, 5*time.Minute); err != nil {
		return nil, apperror.ErrInternal
	}

	return &contract.CaptchaResp{
		CaptchaKey:     key,
		MasterBgBase64: masterBase64,
		TileBase64:     tileBase64,
		ThumbX:         block.TileX,
		ThumbY:         block.TileY,
		TileWidth:      block.Width,
		TileHeight:     block.Height,
	}, nil
}

// VerifyCaptcha 校验滑块验证码（阶段1：滑动后立即调用）
// 流程：一次性获取并删除 captcha 坐标 → 校验 X 坐标（容忍度5px）→ 通过后写入 captcha_verified 标记（TTL 60s）
func (s *AuthService) VerifyCaptcha(ctx context.Context, req contract.VerifyCaptchaReq) error {
	data, err := s.captchaCache.GetAndDel(ctx, req.CaptchaKey)
	if err != nil || data == nil {
		return apperror.ErrCaptchaExpired
	}
	if !slide.Validate(req.SlideX, req.SlideY, data.TargetX, data.TargetY, 5) {
		return apperror.ErrInvalidCaptcha
	}

	if err := s.captchaCache.SetVerified(ctx, req.CaptchaKey, 60*time.Second); err != nil {
		return apperror.ErrInternal
	}

	return nil
}

// Me 获取当前登录用户信息
func (s *AuthService) Me(ctx context.Context, userID uuid.UUID) (*model.User, error) {
	user, err := s.userRepo.ByID(ctx, userID)
	if err != nil || user == nil {
		return nil, apperror.ErrUserNotFound
	}
	return user, nil
}

// Search 按用户名模糊搜索其他用户（排除自己，最多返回 20 条）
func (s *AuthService) Search(ctx context.Context, query string, userID uuid.UUID) ([]contract.UserSearchItem, error) {
	if query == "" {
		return []contract.UserSearchItem{}, nil
	}
	return s.userRepo.Search(ctx, query, userID, 20)
}

// GetUserProfile 获取指定用户的公开信息（头像/昵称/简介）
func (s *AuthService) GetUserProfile(ctx context.Context, userID uuid.UUID, targetID uuid.UUID) (*contract.UserProfile, error) {
	user, err := s.userRepo.ByID(ctx, targetID)
	if err != nil || user == nil {
		return nil, apperror.ErrUserNotFound
	}
	return &contract.UserProfile{
		ID:       user.ID,
		Username: user.Username,
		Nickname: user.Nickname,
		Avatar:   user.Avatar,
		Bio:      user.Bio,
	}, nil
}

// UpdateProfile 更新个人信息（昵称/简介/头像）
// 流程：如果上传了新头像 → 上传到 MinIO 并更新 avatar 字段 → 更新 nickname/bio → 返回最新用户信息
func (s *AuthService) UpdateProfile(ctx context.Context, userID uuid.UUID, req contract.UpdateProfileReq, avatarFile io.Reader, avatarFileName string) (*model.User, error) {
	updates := make(map[string]interface{})

	if req.Nickname != nil {
		if *req.Nickname == "" {
			updates["nickname"] = nil
		} else {
			updates["nickname"] = *req.Nickname
		}
	}

	if req.Bio != nil {
		if *req.Bio == "" {
			updates["bio"] = nil
		} else {
			updates["bio"] = *req.Bio
		}
	}

	// 处理头像上传：如果有新头像文件，上传到 MinIO，记录路径
	if avatarFile != nil && avatarFileName != "" && s.minioFunc != nil {
		ext := filepath.Ext(avatarFileName)
		if ext == "" {
			ext = ".png"
		}
		avatarID := uuid.Must(uuid.NewV7()).String()
		minioPath := fmt.Sprintf("avatar/%s/%s%s", userID.String(), avatarID, ext)

		avatarBytes, err := io.ReadAll(avatarFile)
		if err != nil {
			return nil, apperror.ErrInternal
		}
		if err := s.minioFunc.UploadFile(ctx, s.minioBucket, minioPath, bytes.NewReader(avatarBytes), int64(len(avatarBytes))); err != nil {
			return nil, apperror.ErrInternal
		}
		updates["avatar"] = minioPath
	}

	if len(updates) == 0 {
		return s.Me(ctx, userID)
	}

	if err := s.userRepo.UpdateProfile(ctx, userID, updates); err != nil {
		return nil, apperror.ErrInternal
	}

	return s.Me(ctx, userID)
}

// UpdatePassword 修改密码
// 流程：查询用户 → bcrypt 校验旧密码 → 生成新密码哈希 → 更新 password_hash
func (s *AuthService) UpdatePassword(ctx context.Context, userID uuid.UUID, req contract.UpdatePasswordReq) error {
	user, err := s.userRepo.ByID(ctx, userID)
	if err != nil || user == nil {
		return apperror.ErrUserNotFound
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)); err != nil {
		return apperror.ErrInvalidCredentials
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return apperror.ErrInternal
	}

	return s.userRepo.UpdateProfile(ctx, userID, map[string]interface{}{
		"password_hash": string(newHash),
	})
}

// ForgotPassword 忘记密码——生成密码重置 token 并写入数据库
// 流程：滑块验证码校验（先于查用户，防枚举）→ 查用户（必须存在且 status=1）→ IP 级防滥发 → 邮箱冷却 → 生成随机 token → SHA-256 → 写入 password_resets → 设置冷却
// 安全设计：无论邮箱是否存在都统一返回 nil，防止用户枚举攻击
func (s *AuthService) ForgotPassword(ctx context.Context, req contract.ForgotPasswordReq) error {
	verified, err := s.captchaCache.ConsumeVerified(ctx, req.CaptchaKey)
	if err != nil || !verified {
		return apperror.ErrCaptchaNotVerified
	}

	user, err := s.userRepo.ByEmail(ctx, req.Email)
	if user == nil || err != nil {
		return nil // 统一返回成功，防止用户枚举
	}
	if user.Status != 1 {
		return nil // 未激活/已禁用用户也统一返回成功
	}

	locked, _ := s.limiter.IsForgotPasswordLocked(ctx, req.Email)
	if locked {
		return apperror.ErrTooManyAttempts
	}

	onCooldown, _ := s.limiter.CheckForgotCooldown(ctx, req.Email)
	if onCooldown {
		return apperror.ErrResendCooldown
	}

	rawToken, err := generateRefreshToken()
	if err != nil {
		return apperror.ErrInternal
	}
	tokenHash := sha256Hex(rawToken)

	record := &model.PasswordReset{
		ID:        uuid.Must(uuid.NewV7()),
		UserID:    user.ID,
		Email:     req.Email,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}
	if err := s.passwordResetRepo.Create(ctx, record); err != nil {
		return apperror.ErrInternal
	}

	_ = s.limiter.SetForgotCooldown(ctx, req.Email)
	if _, ierr := s.limiter.IncrForgotPassword(ctx, req.Email); ierr != nil {
		// ignore
	}

	// 异步发送密码重置邮件
	email := req.Email
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", s.frontendBaseURL, rawToken)
	go func() {
		if err := s.emailSender.Send(context.Background(),
			email,
			"CloudEmu 密码重置",
			fmt.Sprintf("您正在重置 CloudEmu 账户密码，\n请点击以下链接完成重置：\n%s\n\n该链接仅 15 分钟内有效，\n如您未请求该操作，请忽略此邮件。", resetURL),
		); err != nil {
			slog.Error("failed to send password reset email", "error", err, "to", email)
		}
	}()

	return nil
}

// ResetPassword 使用重置 token 设置新密码
// 流程：SHA-256(token) → 查 password_resets → 校验未使用未过期 → bcrypt 更新密码 → 标记 token 已使用
func (s *AuthService) ResetPassword(ctx context.Context, req contract.ResetPasswordReq) error {
	hash := sha256Hex(req.Token)

	record, err := s.passwordResetRepo.ByHash(ctx, hash)
	if err != nil || record == nil {
		return apperror.ErrResetTokenInvalid
	}

	if record.UsedAt != nil {
		return apperror.ErrResetTokenUsed
	}

	if time.Now().After(record.ExpiresAt) {
		return apperror.ErrResetTokenInvalid
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return apperror.ErrInternal
	}

	if err := s.userRepo.UpdateProfile(ctx, record.UserID, map[string]interface{}{
		"password_hash": string(newHash),
	}); err != nil {
		return apperror.ErrInternal
	}

	if err := s.passwordResetRepo.MarkUsed(ctx, record.ID); err != nil {
		return apperror.ErrInternal
	}

	return nil
}

// generateTokenPair 生成 JWT Access Token + Refresh Token 对
// Access Token 存入 JWT Claims，Refresh Token 原始值返回给客户端，SHA-256 哈希后存入数据库
func (s *AuthService) generateTokenPair(ctx context.Context, userID uuid.UUID, username string) (*contract.TokenPair, error) {
	accessToken, err := jwtutil.Generate(userID, username, s.jwtSecret, contract.AccessTokenTTL)
	if err != nil {
		return nil, err
	}

	refreshTokenRaw, err := generateRefreshToken()
	if err != nil {
		return nil, err
	}

	rt := &model.RefreshToken{
		ID:        uuid.Must(uuid.NewV7()),
		UserID:    userID,
		TokenHash: sha256Hex(refreshTokenRaw),
		ExpiresAt: time.Now().Add(contract.RefreshTokenTTL),
	}
	if err := s.refreshTokenRepo.Create(ctx, rt); err != nil {
		return nil, err
	}

	return &contract.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshTokenRaw,
		ExpiresIn:    int64(contract.AccessTokenTTL.Seconds()),
	}, nil
}

// generateCode 生成6位数字邮箱验证码
func (s *AuthService) generateCode() (string, error) {
	return s.generateSimpleCode(6)
}

// generateSimpleCode 生成指定位数的纯数字随机码
func (s *AuthService) generateSimpleCode(digits int) (string, error) {
	result := make([]byte, digits)
	for i := 0; i < digits; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		result[i] = byte('0') + byte(n.Int64())
	}
	return string(result), nil
}

// generateRefreshToken 生成 32 字节随机 Refresh Token，16 进制编码为 64 字符
func generateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// sha256Hex 对字符串做 SHA-256 哈希，返回 16 进制字符串
func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
