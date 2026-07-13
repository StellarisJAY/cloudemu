package repo

import (
	"context"
	"errors"
	"time"

	"github.com/StellarisJAY/cloudemu/internal/control-plane/contract"
	"github.com/StellarisJAY/cloudemu/internal/control-plane/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserRepo 用户表数据访问层，纯数据库操作，不包含业务逻辑
type UserRepo struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{db: db}
}

// Create 插入新用户
func (r *UserRepo) Create(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

// ByID 按主键ID查询用户
func (r *UserRepo) ByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, err
}

// ByEmail 按邮箱查询用户
func (r *UserRepo) ByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, err
}

// ByUsername 按用户名查询用户
func (r *UserRepo) ByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, err
}

// UpdateStatus 更新用户状态（0=待激活, 1=已激活, 2=已禁用）
func (r *UserRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status int16) error {
	return r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Update("status", status).Error
}

// UpdateLastLogin 更新最近登录时间为当前时间
func (r *UserRepo) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Update("last_login_at", &now).Error
}

// UpdateProfile 更新用户个人资料（昵称、头像、简介等），通过 map 灵活指定要更新的字段
func (r *UserRepo) UpdateProfile(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Updates(updates).Error
}

// Search 模糊搜索用户（按 username ILIKE 匹配），排除指定用户，限制返回数量
func (r *UserRepo) Search(ctx context.Context, query string, excludeID uuid.UUID, limit int) ([]contract.UserSearchItem, error) {
	var results []contract.UserSearchItem
	err := r.db.WithContext(ctx).
		Model(&model.User{}).
		Select("id, username, nickname, avatar").
		Where("username ILIKE ? AND id != ?", "%"+query+"%", excludeID).
		Where("status = 1").
		Order("username ASC").
		Limit(limit).
		Find(&results).Error
	return results, err
}

// EmailVerificationRepo 邮箱验证码记录数据访问层
type EmailVerificationRepo struct {
	db *gorm.DB
}

func NewEmailVerificationRepo(db *gorm.DB) *EmailVerificationRepo {
	return &EmailVerificationRepo{db: db}
}

// Create 插入验证码记录
func (r *EmailVerificationRepo) Create(ctx context.Context, record *model.EmailVerification) error {
	return r.db.WithContext(ctx).Create(record).Error
}

// LatestByEmail 查指定邮箱最新一条验证码记录（按 created_at DESC）
func (r *EmailVerificationRepo) LatestByEmail(ctx context.Context, email string) (*model.EmailVerification, error) {
	var record model.EmailVerification
	err := r.db.WithContext(ctx).
		Where("email = ?", email).
		Order("created_at DESC").
		First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &record, err
}

// MarkVerified 将验证码标记为已验证（设置 verified_at）
func (r *EmailVerificationRepo) MarkVerified(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.EmailVerification{}).
		Where("id = ?", id).
		Update("verified_at", &now).Error
}

// RefreshTokenRepo 刷新令牌数据访问层
type RefreshTokenRepo struct {
	db *gorm.DB
}

func NewRefreshTokenRepo(db *gorm.DB) *RefreshTokenRepo {
	return &RefreshTokenRepo{db: db}
}

// Create 插入新的刷新令牌记录（token_hash 为原始 token 的 SHA-256）
func (r *RefreshTokenRepo) Create(ctx context.Context, token *model.RefreshToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

// ByHash 按 token_hash 查询刷新令牌
func (r *RefreshTokenRepo) ByHash(ctx context.Context, hash string) (*model.RefreshToken, error) {
	var token model.RefreshToken
	err := r.db.WithContext(ctx).Where("token_hash = ?", hash).First(&token).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &token, err
}

// DeleteByUser 删除某用户的所有刷新令牌（用于强制下线）
func (r *RefreshTokenRepo) DeleteByUser(ctx context.Context, userID uuid.UUID) error {
	return r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&model.RefreshToken{}).Error
}

// DeleteByHash 按哈希删除单条刷新令牌（用于轮换）
func (r *RefreshTokenRepo) DeleteByHash(ctx context.Context, hash string) error {
	return r.db.WithContext(ctx).Where("token_hash = ?", hash).Delete(&model.RefreshToken{}).Error
}

// PasswordResetRepo 密码重置记录数据访问层
type PasswordResetRepo struct {
	db *gorm.DB
}

func NewPasswordResetRepo(db *gorm.DB) *PasswordResetRepo {
	return &PasswordResetRepo{db: db}
}

// Create 插入密码重置记录
func (r *PasswordResetRepo) Create(ctx context.Context, record *model.PasswordReset) error {
	return r.db.WithContext(ctx).Create(record).Error
}

// ByHash 按 token_hash 查询密码重置记录
func (r *PasswordResetRepo) ByHash(ctx context.Context, hash string) (*model.PasswordReset, error) {
	var record model.PasswordReset
	err := r.db.WithContext(ctx).Where("token_hash = ?", hash).First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &record, err
}

// MarkUsed 标记密码重置 token 已被使用（设置 used_at）
func (r *PasswordResetRepo) MarkUsed(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).
		Model(&model.PasswordReset{}).
		Where("id = ?", id).
		Update("used_at", &now).Error
}
