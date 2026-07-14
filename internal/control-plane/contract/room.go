package contract

import (
	"context"

	"github.com/StellarisJAY/cloudemu/internal/control-plane/model"
	"github.com/google/uuid"
)

// RoomService 房间业务逻辑接口
type RoomService interface {
	Create(ctx context.Context, hostID uuid.UUID, req CreateRoomReq) (*model.Room, error)               // 创建房间，房主自动入座Port0，invitee_ids中的好友自动加入（默认旁观）
	List(ctx context.Context, userID uuid.UUID) ([]model.Room, error)                                   // 列出当前用户参与的所有活跃房间
	InviteToRoom(ctx context.Context, hostID uuid.UUID, roomID uuid.UUID, inviteeIDs []uuid.UUID) error // 房主邀请好友加入已有房间，直接加入（无需接受）
	ChangeRole(ctx context.Context, hostID uuid.UUID, req ChangeRoleReq) error                          // 房主调整成员角色：提升为玩家（分配/转移端口）或降级为旁观（port=0归还房主，其他释放）
	SelectRom(ctx context.Context, hostID uuid.UUID, req SelectRomReq) error                            // 房主选择/切换房间的 ROM
	Start(ctx context.Context, hostID uuid.UUID, roomID uuid.UUID) (*StartGameResponse, error)          // 房主启动游戏，调度Worker→gRPC调用→房间状态变为playing→返回LiveKit token
	Pause(ctx context.Context, hostID uuid.UUID, roomID uuid.UUID) error                                // 房主暂停游戏，通知Worker→DataChannel→EmuRunner
	Resume(ctx context.Context, hostID uuid.UUID, roomID uuid.UUID) error                               // 房主继续游戏，通知Worker→DataChannel→EmuRunner
	Stop(ctx context.Context, hostID uuid.UUID, roomID uuid.UUID) error                                 // 房主停止游戏，杀EmuRunner进程→重置非房主→清除缓存→房间回到waiting态
	Delete(ctx context.Context, hostID uuid.UUID, roomID uuid.UUID) error                               // 房主删除房间，仅waiting态可删→全员MarkLeft→清Redis→status=2
	GetLivekitToken(ctx context.Context, userID uuid.UUID, roomID uuid.UUID) (*LivekitTokenResp, error) // 获取LiveKit token，游戏未开始时返回waiting=true
	GetMembers(ctx context.Context, userID uuid.UUID, roomID uuid.UUID) ([]RoomMemberInfo, error)       // 获取房间成员列表
	KickPlayer(ctx context.Context, hostID uuid.UUID, req KickPlayerReq) error                          // 房主踢出玩家
	Leave(ctx context.Context, userID uuid.UUID, roomID uuid.UUID) error                                // 玩家离开房间，房主离开时自动转移/关闭
	SaveState(ctx context.Context, hostID uuid.UUID, roomID uuid.UUID) (*model.SaveState, error)        // 房主保存存档：校验房主+playing+已选ROM→预签名PUT→通知Worker→落库
	LoadState(ctx context.Context, hostID uuid.UUID, req LoadStateReq) error                            // 房主读取存档：校验房主+playing+三要素匹配→预签名GET→通知Worker反序列化
	LoadLatestState(ctx context.Context, hostID uuid.UUID, roomID uuid.UUID) error                      // 房主加载最新存档：取当前机种+ROM的最新存档并读取
	RenameSaveState(ctx context.Context, hostID uuid.UUID, req RenameSaveStateReq) error                // 房主重命名存档
	DeleteSaveState(ctx context.Context, hostID uuid.UUID, req DeleteSaveStateReq) error                // 房主删除存档（同时删除 MinIO 二进制）
	ListSaveStates(ctx context.Context, userID uuid.UUID, roomID uuid.UUID) ([]model.SaveState, error)  // 列出房间存档（房间成员可查，仅返回与当前机种+ROM匹配的存档，时间倒序）
}

// RoomRepo 房间表数据访问接口
type RoomRepo interface {
	Create(ctx context.Context, room *model.Room) error                       // 插入新房间
	ByID(ctx context.Context, id uuid.UUID) (*model.Room, error)              // 按ID查询房间
	ActiveByUser(ctx context.Context, userID uuid.UUID) ([]model.Room, error) // 查用户参与的所有非关闭房间
	UpdateStatus(ctx context.Context, id uuid.UUID, status int16) error       // 更新房间状态（同时设置started_at/closed_at）
	SetWorkerAddr(ctx context.Context, id uuid.UUID, addr string) error       // 设置房间分配到的 Worker 地址
	UpdateRomID(ctx context.Context, id uuid.UUID, romID *uuid.UUID) error    // 更新房间当前游玩的 ROM（romID 为 nil 时清空）
}

// RoomPlayerRepo 房间座位表数据访问接口
type RoomPlayerRepo interface {
	Create(ctx context.Context, player *model.RoomPlayer) error                             // 插入座位记录
	ActiveByRoom(ctx context.Context, roomID uuid.UUID) ([]model.RoomPlayer, error)         // 查房间内所有活跃玩家
	ActiveByRoomWithUser(ctx context.Context, roomID uuid.UUID) ([]RoomMemberInfo, error)   // 查房间内所有活跃玩家（含用户信息）
	ActiveByUser(ctx context.Context, userID uuid.UUID) ([]model.RoomPlayer, error)         // 查用户当前所在的所有座位
	ByRoomAndUser(ctx context.Context, roomID, userID uuid.UUID) (*model.RoomPlayer, error) // 查用户在指定房间的座位
	UpdateRoleAndPort(ctx context.Context, id uuid.UUID, role int16, port *int16) error     // 更新玩家角色和手柄端口
	MarkLeft(ctx context.Context, id uuid.UUID) error                                       // 标记玩家离开房间
	TransferHost(ctx context.Context, roomID uuid.UUID, newHostID uuid.UUID) error          // 转移房主身份给其他玩家
}

// RoomStateCache 房间实时状态缓存接口（Redis）
type RoomStateCache interface {
	SetPort(ctx context.Context, roomID uuid.UUID, port int16, userID uuid.UUID) error                // 设置某端口绑定的玩家
	RemovePort(ctx context.Context, roomID uuid.UUID, port int16) error                               // 移除某端口绑定
	GetPorts(ctx context.Context, roomID uuid.UUID) (map[int16]uuid.UUID, error)                      // 获取房间所有端口映射
	ClearRoom(ctx context.Context, roomID uuid.UUID) error                                            // 清除房间所有缓存数据
	SetLivekitInfo(ctx context.Context, roomID uuid.UUID, livekitUrl string, room string) error       // 存储 LiveKit 地址和房间名（token 由玩家独立生成，不再缓存）
	GetLivekitInfo(ctx context.Context, roomID uuid.UUID) (livekitUrl string, room string, err error) // 获取 LiveKit 地址和房间名
}

// SaveStateRepo 游戏存档表数据访问接口
type SaveStateRepo interface {
	Create(ctx context.Context, ss *model.SaveState) error                              // 插入存档记录
	ByID(ctx context.Context, id uuid.UUID) (*model.SaveState, error)                   // 按ID查询存档
	ListByRoom(ctx context.Context, roomID uuid.UUID) ([]model.SaveState, error)        // 查询指定房间的所有存档（创建时间倒序）
	ListByRoomRom(ctx context.Context, roomID uuid.UUID, emulatorType string, romID uuid.UUID) ([]model.SaveState, error) // 查询房间+机种+ROM 三者匹配的存档（创建时间倒序）
	LatestByRoomRom(ctx context.Context, roomID uuid.UUID, emulatorType string, romID uuid.UUID) (*model.SaveState, error) // 查询房间+机种+ROM 三者匹配的最新一条存档，无匹配返回 nil
	Rename(ctx context.Context, id uuid.UUID, name string) error                        // 修改存档名称
	Delete(ctx context.Context, id uuid.UUID) error                                     // 删除存档记录
}
