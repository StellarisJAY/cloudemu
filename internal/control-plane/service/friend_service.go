package service

import (
	"context"

	"github.com/StellarisJAY/cloudemu/internal/control-plane/contract"
	"github.com/StellarisJAY/cloudemu/internal/control-plane/model"
	"github.com/StellarisJAY/cloudemu/internal/pkg/apperror"

	"github.com/google/uuid"
)

// FriendService 好友关系业务逻辑实现
type FriendService struct {
	friendRepo contract.FriendRepo
	userRepo   contract.UserRepo
}

// NewFriendService 创建 FriendService 实例
func NewFriendService(friendRepo contract.FriendRepo, userRepo contract.UserRepo) *FriendService {
	return &FriendService{friendRepo: friendRepo, userRepo: userRepo}
}

// Add 发送好友申请
// 规则：不能添加自己 → 目标用户必须存在 → 不能重复申请（双向查重）→ 插入好友记录(status=0=pending)
func (s *FriendService) Add(ctx context.Context, userID, friendID uuid.UUID) error {
	if userID == friendID {
		return apperror.ErrFriendSelf
	}

	target, err := s.userRepo.ByID(ctx, friendID)
	if err != nil || target == nil {
		return apperror.ErrUserNotFound
	}

	existing, _ := s.friendRepo.ByPair(ctx, userID, friendID)
	if existing != nil {
		return apperror.ErrFriendExists
	}

	friend := &model.Friend{
		ID:       uuid.Must(uuid.NewV7()),
		UserID:   userID,
		FriendID: friendID,
		Status:   0,
	}

	if err := s.friendRepo.Create(ctx, friend); err != nil {
		return apperror.ErrInternal
	}

	return nil
}

// Accept 接受好友申请
// 规则：只能由申请接收方操作 → 状态必须为pending → 更新为accepted(status=1)
func (s *FriendService) Accept(ctx context.Context, userID, friendID uuid.UUID) error {
	existing, err := s.friendRepo.ByPair(ctx, userID, friendID)
	if err != nil || existing == nil {
		return apperror.ErrFriendNotFound
	}

	if existing.FriendID != userID || existing.Status != 0 {
		return apperror.ErrFriendNotFound
	}

	if err := s.friendRepo.UpdateStatus(ctx, existing.ID, 1); err != nil {
		return apperror.ErrInternal
	}

	return nil
}

// List 列出当前用户所有已接受的好友（含好友的用户信息）
func (s *FriendService) List(ctx context.Context, userID uuid.UUID) ([]contract.FriendWithUser, error) {
	return s.friendRepo.AcceptedWithUser(ctx, userID)
}

// Pending 列出当前用户收到的待处理好友请求（含发起者信息）
func (s *FriendService) Pending(ctx context.Context, userID uuid.UUID) ([]contract.FriendPendingItem, error) {
	return s.friendRepo.PendingByUser(ctx, userID)
}

// Reject 拒绝好友申请
// 规则：只能由申请接收方操作 → 状态必须为pending(0) → 更新为rejected(3)
func (s *FriendService) Reject(ctx context.Context, userID, friendID uuid.UUID) error {
	existing, err := s.friendRepo.ByPair(ctx, userID, friendID)
	if err != nil || existing == nil {
		return apperror.ErrFriendNotFound
	}

	if existing.FriendID != userID || existing.Status != 0 {
		return apperror.ErrFriendNotFound
	}

	if err := s.friendRepo.UpdateStatus(ctx, existing.ID, 3); err != nil {
		return apperror.ErrInternal
	}

	return nil
}
