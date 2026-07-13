package contract

import (
	"context"

	"github.com/StellarisJAY/cloudemu/internal/control-plane/model"
	"github.com/google/uuid"
)

// FriendService 好友关系业务逻辑接口
type FriendService interface {
	Add(ctx context.Context, userID, friendID uuid.UUID) error                  // 发送好友申请（不能自己加自己，不能重复申请）
	Accept(ctx context.Context, userID, friendID uuid.UUID) error               // 接受好友申请（只能由接收方操作）
	Reject(ctx context.Context, userID, friendID uuid.UUID) error               // 拒绝好友申请（只能由接收方操作，状态变为 3=rejected）
	List(ctx context.Context, userID uuid.UUID) ([]FriendWithUser, error)       // 列出当前用户所有已接受的好友（含用户信息）
	Pending(ctx context.Context, userID uuid.UUID) ([]FriendPendingItem, error) // 列出当前用户收到的待处理好友请求（含发起者信息）
}

// FriendRepo 好友关系表数据访问接口
type FriendRepo interface {
	Create(ctx context.Context, friend *model.Friend) error                           // 插入好友关系记录
	ByPair(ctx context.Context, a, b uuid.UUID) (*model.Friend, error)                // 查询两个用户之间的好友关系（双向匹配）
	AcceptedByUser(ctx context.Context, userID uuid.UUID) ([]model.Friend, error)     // 查某用户所有已接受的好友（仅 friend 表）
	AcceptedWithUser(ctx context.Context, userID uuid.UUID) ([]FriendWithUser, error) // 查某用户所有已接受的好友（JOIN users 表，全结果）
	PendingByUser(ctx context.Context, userID uuid.UUID) ([]FriendPendingItem, error) // 查某用户收到的待处理好友请求（JOIN users 表，返回发起者信息）
	UpdateStatus(ctx context.Context, id uuid.UUID, status int16) error               // 更新好友关系状态
}
