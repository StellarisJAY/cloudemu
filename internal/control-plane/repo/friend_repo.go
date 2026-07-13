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

// FriendRepo 好友关系表数据访问层
type FriendRepo struct {
	db *gorm.DB
}

func NewFriendRepo(db *gorm.DB) *FriendRepo {
	return &FriendRepo{db: db}
}

// Create 插入好友关系记录（初始状态为 pending）
func (r *FriendRepo) Create(ctx context.Context, friend *model.Friend) error {
	return r.db.WithContext(ctx).Create(friend).Error
}

// ByPair 查询两个用户之间的好友关系（双向匹配：A→B 或 B→A）
func (r *FriendRepo) ByPair(ctx context.Context, a, b uuid.UUID) (*model.Friend, error) {
	var f model.Friend
	err := r.db.WithContext(ctx).
		Where("(user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)", a, b, b, a).
		First(&f).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &f, err
}

// AcceptedByUser 查询某用户所有已接受的好友（status=1，用户ID可为 sender 或 receiver）
func (r *FriendRepo) AcceptedByUser(ctx context.Context, userID uuid.UUID) ([]model.Friend, error) {
	var friends []model.Friend
	err := r.db.WithContext(ctx).
		Where("(user_id = ? OR friend_id = ?) AND status = 1", userID, userID).
		Order("accepted_at DESC").
		Find(&friends).Error
	return friends, err
}

// UpdateStatus 更新好友关系状态，若变为 accepted 则同时设置 accepted_at
func (r *FriendRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status int16) error {
	updates := map[string]interface{}{"status": status}
	if status == 1 {
		now := time.Now
		updates["accepted_at"] = now()
	}
	return r.db.WithContext(ctx).Model(&model.Friend{}).Where("id = ?", id).Updates(updates).Error
}

// AcceptedWithUser 查某用户所有已接受的好友，JOIN users 表获取好友的个人信息
func (r *FriendRepo) AcceptedWithUser(ctx context.Context, userID uuid.UUID) ([]contract.FriendWithUser, error) {
	var results []contract.FriendWithUser
	err := r.db.WithContext(ctx).
		Table("friends").
		Select(`
			friends.id, friends.user_id, friends.friend_id, friends.status,
			friends.accepted_at, friends.created_at,
			users.username, users.nickname, users.avatar
		`).
		Joins("INNER JOIN users ON users.id = CASE WHEN friends.user_id = ? THEN friends.friend_id ELSE friends.user_id END", userID).
		Where("(friends.user_id = ? OR friends.friend_id = ?) AND friends.status = 1", userID, userID).
		Order("friends.accepted_at DESC").
		Scan(&results).Error
	return results, err
}

// PendingByUser 查某用户收到的待处理好友请求（该用户是接收方），JOIN users 表返回发起者信息
func (r *FriendRepo) PendingByUser(ctx context.Context, userID uuid.UUID) ([]contract.FriendPendingItem, error) {
	var results []contract.FriendPendingItem
	err := r.db.WithContext(ctx).
		Table("friends").
		Select(`
			friends.id, friends.user_id, friends.friend_id, friends.created_at,
			users.username, users.nickname, users.avatar
		`).
		Joins("INNER JOIN users ON users.id = friends.user_id").
		Where("friends.friend_id = ? AND friends.status = 0", userID).
		Order("friends.created_at DESC").
		Scan(&results).Error
	return results, err
}
