package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/StellarisJAY/cloudemu/internal/control-plane/contract"
	"github.com/StellarisJAY/cloudemu/internal/control-plane/model"
	"github.com/StellarisJAY/cloudemu/internal/pkg/apperror"

	"github.com/google/uuid"
)

// RoomService 房间业务逻辑实现
type RoomService struct {
	roomRepo       contract.RoomRepo
	roomPlayerRepo contract.RoomPlayerRepo
	friendRepo     contract.FriendRepo
	roomStateCache contract.RoomStateCache
	romRepo        contract.RomRepo
	scheduler      contract.Scheduler
	workerRegistry contract.WorkerRegistry
	workerClient   contract.WorkerClient
	minioFunc      contract.MinioFunc
	bucket         string
	saveStateRepo  contract.SaveStateRepo
}

// NewRoomService 创建 RoomService 实例
func NewRoomService(
	roomRepo contract.RoomRepo,
	roomPlayerRepo contract.RoomPlayerRepo,
	friendRepo contract.FriendRepo,
	roomStateCache contract.RoomStateCache,
	romRepo contract.RomRepo,
	scheduler contract.Scheduler,
	workerRegistry contract.WorkerRegistry,
	workerClient contract.WorkerClient,
	minioFunc contract.MinioFunc,
	bucket string,
	saveStateRepo contract.SaveStateRepo,
) *RoomService {
	return &RoomService{
		roomRepo:       roomRepo,
		roomPlayerRepo: roomPlayerRepo,
		friendRepo:     friendRepo,
		roomStateCache: roomStateCache,
		romRepo:        romRepo,
		scheduler:      scheduler,
		workerRegistry: workerRegistry,
		workerClient:   workerClient,
		minioFunc:      minioFunc,
		bucket:         bucket,
		saveStateRepo:  saveStateRepo,
	}
}

// Create 创建房间
// 流程：插入rooms(status=0) → 房主自动加入room_players(role=host, port=0) → invitee_ids中的好友直接加入（默认旁观，无需接受）
func (s *RoomService) Create(ctx context.Context, hostID uuid.UUID, req contract.CreateRoomReq) (*model.Room, error) {
	// 若指定了 RomID，需校验 ROM 存在且模拟器类型匹配；未指定则保持为 nil
	if req.RomID != nil {
		rom, err := s.romRepo.ByID(ctx, *req.RomID)
		if err != nil || rom == nil {
			return nil, apperror.ErrRomNotExist
		}
		if rom.EmulatorType != req.EmulatorType {
			return nil, apperror.ErrRomTypeMismatch
		}
	}

	room := &model.Room{
		ID:           uuid.Must(uuid.NewV7()),
		HostID:       hostID,
		Title:        req.Title,
		EmulatorType: req.EmulatorType,
		RomID:        req.RomID,
		MaxPorts:     req.MaxPorts,
		Status:       0,
	}

	if err := s.roomRepo.Create(ctx, room); err != nil {
		return nil, apperror.ErrInternal
	}

	port := int16(0)
	player := &model.RoomPlayer{
		ID:     uuid.Must(uuid.NewV7()),
		RoomID: room.ID,
		UserID: hostID,
		Role:   0,
		Port:   &port,
	}
	if err := s.roomPlayerRepo.Create(ctx, player); err != nil {
		return nil, apperror.ErrInternal
	}

	// 被邀请的好友直接加入房间（默认旁观）
	for _, inviteeID := range req.InviteeIDs {
		if inviteeID == hostID {
			continue
		}
		if err := s.addPlayerToRoom(ctx, hostID, room, inviteeID); err != nil {
			return nil, err
		}
	}

	return room, nil
}

// List 列出当前用户参与的所有活跃房间
func (s *RoomService) List(ctx context.Context, userID uuid.UUID) ([]model.Room, error) {
	return s.roomRepo.ActiveByUser(ctx, userID)
}

// InviteToRoom 房主邀请好友加入已有房间，好友直接加入（类似微信拉群，无需接受）
func (s *RoomService) InviteToRoom(ctx context.Context, hostID uuid.UUID, roomID uuid.UUID, inviteeIDs []uuid.UUID) error {
	room, err := s.roomRepo.ByID(ctx, roomID)
	if err != nil || room == nil {
		return apperror.ErrRoomNotExist
	}

	if room.HostID != hostID {
		return apperror.ErrNotRoomHost
	}

	for _, inviteeID := range inviteeIDs {
		if inviteeID == hostID {
			continue
		}
		if err := s.addPlayerToRoom(ctx, hostID, room, inviteeID); err != nil {
			return err
		}
	}

	return nil
}

// addPlayerToRoom 将玩家加入房间（内部方法）
// 校验好友关系 → 校验未重复加入 → 插入room_players(role=spectator)
func (s *RoomService) addPlayerToRoom(ctx context.Context, hostID uuid.UUID, room *model.Room, userID uuid.UUID) error {
	// 校验好友关系
	isFriend, err := s.isFriend(ctx, hostID, userID)
	if err != nil {
		return err
	}
	if !isFriend {
		return apperror.ErrNotFriend
	}

	// 校验用户未已在房间中
	existing, err := s.roomPlayerRepo.ByRoomAndUser(ctx, room.ID, userID)
	if err != nil {
		return apperror.ErrInternal
	}
	if existing != nil {
		return apperror.ErrAlreadyInRoom
	}

	sp := &model.RoomPlayer{
		ID:     uuid.Must(uuid.NewV7()),
		RoomID: room.ID,
		UserID: userID,
		Role:   2,
		Port:   nil,
	}
	if err := s.roomPlayerRepo.Create(ctx, sp); err != nil {
		return apperror.ErrInternal
	}

	return nil
}

// isFriend 检查两个用户是否为已接受的好友关系
func (s *RoomService) isFriend(ctx context.Context, userA, userB uuid.UUID) (bool, error) {
	if userA == userB {
		return false, nil
	}
	friends, err := s.friendRepo.AcceptedByUser(ctx, userA)
	if err != nil {
		return false, apperror.ErrInternal
	}
	for _, f := range friends {
		if f.UserID == userB || f.FriendID == userB {
			return true, nil
		}
	}
	return false, nil
}

// SelectRom 房主选择/切换房间的 ROM
func (s *RoomService) SelectRom(ctx context.Context, hostID uuid.UUID, req contract.SelectRomReq) error {
	// RoomID 与 RomID 已经过 notnil_uuid 校验，必非 nil
	room, err := s.roomRepo.ByID(ctx, *req.RoomID)
	if err != nil || room == nil {
		return apperror.ErrRoomNotExist
	}
	if room.HostID != hostID {
		return apperror.ErrNotRoomHost
	}
	if room.Status != 0 {
		return apperror.ErrRoomNotWaiting
	}

	rom, err := s.romRepo.ByID(ctx, *req.RomID)
	if err != nil || rom == nil {
		return apperror.ErrRomNotExist
	}
	if rom.EmulatorType != room.EmulatorType {
		return apperror.ErrRomTypeMismatch
	}

	return s.roomRepo.UpdateRomID(ctx, *req.RoomID, req.RomID)
}

// ChangeRole 房主调整成员角色
// role=1(提升为玩家)：必须传入 port，若端口已被占用则原占有者让出（房主仅清端口保持 role=0，其他玩家降为旁观）→ 目标设为 role=1 并绑定端口 → 更新 Redis → 通知 Worker
// role=2(降为旁观)：后端查询目标已有端口并释放 → 若 port=0 则归还房主 → 通知 Worker
func (s *RoomService) ChangeRole(ctx context.Context, hostID uuid.UUID, req contract.ChangeRoleReq) error {
	roomID := *req.RoomID
	targetUserID := *req.UserID

	room, err := s.roomRepo.ByID(ctx, roomID)
	if err != nil || room == nil {
		return apperror.ErrRoomNotExist
	}

	if room.HostID != hostID {
		return apperror.ErrNotRoomHost
	}

	if room.Status != 1 {
		return apperror.ErrRoomNotPlaying
	}

	targetPlayer, err := s.roomPlayerRepo.ByRoomAndUser(ctx, roomID, targetUserID)
	if err != nil || targetPlayer == nil {
		return apperror.ErrNotInRoom
	}

	if req.Role == 1 {
		// 提升为玩家：port 必传
		if req.Port == nil {
			return apperror.ErrPortInvalid
		}
		if *req.Port < 0 || *req.Port >= room.MaxPorts {
			return apperror.ErrPortInvalid
		}

		// 如果端口被其他玩家占用，原占有者让出
		occupied, err := s.roomPlayerRepo.ActiveByRoom(ctx, roomID)
		if err != nil {
			return apperror.ErrInternal
		}
		for _, p := range occupied {
			if p.Port != nil && *p.Port == *req.Port && p.ID != targetPlayer.ID {
				if p.Role == 0 {
					// 房主仅清端口，角色保持 0
					if err := s.roomPlayerRepo.UpdateRoleAndPort(ctx, p.ID, 0, nil); err != nil {
						return apperror.ErrInternal
					}
				} else {
					if err := s.roomPlayerRepo.UpdateRoleAndPort(ctx, p.ID, 2, nil); err != nil {
						return apperror.ErrInternal
					}
				}
				s.roomStateCache.RemovePort(ctx, roomID, *req.Port)
			}
		}

		// 释放目标玩家原有端口（Redis），避免端口映射残留
		if targetPlayer.Port != nil {
			s.roomStateCache.RemovePort(ctx, roomID, *targetPlayer.Port)
		}

		if err := s.roomPlayerRepo.UpdateRoleAndPort(ctx, targetPlayer.ID, 1, req.Port); err != nil {
			return apperror.ErrInternal
		}
		if err := s.roomStateCache.SetPort(ctx, roomID, *req.Port, targetUserID); err != nil {
			return apperror.ErrInternal
		}
	} else {
		// 降为旁观：查询目标已有端口并释放
		oldPort := targetPlayer.Port
		if oldPort == nil {
			return nil // 已是旁观，无需操作
		}

		if err := s.roomPlayerRepo.UpdateRoleAndPort(ctx, targetPlayer.ID, 2, nil); err != nil {
			return apperror.ErrInternal
		}
		s.roomStateCache.RemovePort(ctx, roomID, *oldPort)

		// port=0 归还房主
		if *oldPort == 0 {
			port0 := int16(0)
			hostPlayer, err := s.roomPlayerRepo.ByRoomAndUser(ctx, roomID, hostID)
			if err != nil || hostPlayer == nil {
				slog.Warn("host player record not found when returning port 0", "room_id", roomID, "host_id", hostID)
			} else {
				if err := s.roomPlayerRepo.UpdateRoleAndPort(ctx, hostPlayer.ID, hostPlayer.Role, &port0); err != nil {
					slog.Warn("failed to return port 0 to host", "room_id", roomID, "error", err)
				}
				s.roomStateCache.SetPort(ctx, roomID, 0, hostID)
			}
		}
	}

	// 通知 Worker 更新 port mapping
	if room.WorkerAddr != "" {
		latest, err := s.roomPlayerRepo.ActiveByRoom(ctx, roomID)
		if err != nil {
			slog.Warn("query active players for port mapping failed", "room_id", roomID, "error", err)
			return nil
		}
		mapping := make(map[int32]string, len(latest))
		for _, p := range latest {
			if (p.Role == 0 || p.Role == 1) && p.Port != nil {
				mapping[int32(*p.Port)] = "player:" + p.UserID.String()
			}
		}
		if err := s.workerClient.UpdatePortMapping(ctx, room.WorkerAddr, roomID, mapping); err != nil {
			slog.Warn("worker UpdatePortMapping failed", "room_id", roomID, "error", err)
		}
	}

	return nil
}

// Start 房主启动游戏
// 流程：校验请求者为房主且房间状态为waiting → 查询ROM路径 → 调度选Worker → gRPC调用Worker.StartGame → 更新房间状态为playing → 返回LiveKit token
func (s *RoomService) Start(ctx context.Context, hostID uuid.UUID, roomID uuid.UUID) (*contract.StartGameResponse, error) {
	room, err := s.roomRepo.ByID(ctx, roomID)
	if err != nil || room == nil {
		return nil, apperror.ErrRoomNotExist
	}

	if room.HostID != hostID {
		return nil, apperror.ErrNotRoomHost
	}

	if room.Status != 0 {
		return nil, apperror.ErrRoomNotWaiting
	}

	// 房间必须已选择 ROM 才能开始游戏
	if room.RomID == nil {
		return nil, apperror.ErrRomNotSelected
	}

	// 查询 ROM 元数据（获取 MinIO 路径和模拟器类型）
	rom, err := s.romRepo.ByID(ctx, *room.RomID)
	if err != nil || rom == nil {
		return nil, apperror.ErrRomNotExist
	}

	// 调度选择最优 Worker
	worker, err := s.scheduler.SelectWorker(ctx, s.workerRegistry)
	if err != nil {
		slog.Error("no available worker", "room_id", roomID, "error", err)
		return nil, apperror.ErrNoAvailableWorker
	}

	slog.Info("worker selected",
		"room_id", roomID,
		"worker_id", worker.ID,
		"worker_addr", worker.Addr,
		"sessions", worker.Sessions,
	)

	// 生成 MinIO 预签名下载 URL（5 分钟有效期，Worker 用于下载 ROM 到本地）
	romURL, err := s.minioFunc.PresignedGetURL(ctx, s.bucket, rom.MinioPath, 5*time.Minute)
	if err != nil {
		slog.Error("failed to generate presigned rom url", "room_id", roomID, "error", err)
		return nil, apperror.ErrInternal
	}

	// gRPC 调用 Worker 启动游戏
	resp, err := s.workerClient.StartGame(ctx, worker.Addr, contract.StartGameRequest{
		RoomID:       roomID,
		RomPath:      rom.MinioPath,
		RomURL:       romURL,
		HostUserID:   hostID,
		EmulatorType: room.EmulatorType,
		MaxPorts:     int32(room.MaxPorts),
	})
	if err != nil {
		slog.Error("worker StartGame gRPC failed", "worker", worker.Addr, "room_id", roomID, "error", err)
		return nil, apperror.ErrWorkerUnavailable
	}

	// 更新房间状态为 playing，记录分配到的 Worker 地址
	if err := s.roomRepo.UpdateStatus(ctx, roomID, 1); err != nil {
		// 状态更新失败，通知 Worker 停止（fire-and-forget）
		go func() {
			_ = s.workerClient.StopGame(context.Background(), worker.Addr, roomID)
		}()
		return nil, apperror.ErrInternal
	}

	// 记录房间分配到的 Worker 地址（用于后续 StopGame）
	s.roomRepo.SetWorkerAddr(ctx, roomID, worker.Addr)

	// 将 LiveKit 地址和房间名存入 Redis，供非房主玩家轮询获取（token 按用户独立生成）
	if err := s.roomStateCache.SetLivekitInfo(ctx, roomID, resp.LivekitURL, resp.LivekitRoom); err != nil {
		slog.Warn("failed to cache livekit info", "room_id", roomID, "error", err)
	}

	slog.Info("game started",
		"room_id", roomID,
		"worker_addr", worker.Addr,
		"livekit_room", resp.LivekitRoom,
		"livekit_url", resp.LivekitURL,
	)

	return &contract.StartGameResponse{
		LivekitToken: resp.HostToken, // 返回房主的专属 player token
		LivekitRoom:  resp.LivekitRoom,
		LivekitURL:   resp.LivekitURL,
	}, nil
}

// Pause 房主暂停游戏
// 流程：校验房主身份 → 校验房间状态为 playing → 通知 Worker → EmuRunner
func (s *RoomService) Pause(ctx context.Context, hostID uuid.UUID, roomID uuid.UUID) error {
	room, err := s.roomRepo.ByID(ctx, roomID)
	if err != nil || room == nil {
		return apperror.ErrRoomNotExist
	}
	if room.HostID != hostID {
		return apperror.ErrNotRoomHost
	}
	if room.Status != 1 {
		return apperror.ErrRoomNotPlaying
	}
	if room.WorkerAddr == "" {
		return apperror.ErrWorkerUnavailable
	}
	if err := s.workerClient.PauseGame(ctx, room.WorkerAddr, roomID); err != nil {
		slog.Error("worker PauseGame failed", "room_id", roomID, "error", err)
		return apperror.ErrWorkerUnavailable
	}
	slog.Info("game paused", "room_id", roomID)
	return nil
}

// Resume 房主继续游戏
// 流程：校验房主身份 → 校验房间状态为 playing → 通知 Worker → EmuRunner
func (s *RoomService) Resume(ctx context.Context, hostID uuid.UUID, roomID uuid.UUID) error {
	room, err := s.roomRepo.ByID(ctx, roomID)
	if err != nil || room == nil {
		return apperror.ErrRoomNotExist
	}
	if room.HostID != hostID {
		return apperror.ErrNotRoomHost
	}
	if room.Status != 1 {
		return apperror.ErrRoomNotPlaying
	}
	if room.WorkerAddr == "" {
		return apperror.ErrWorkerUnavailable
	}
	if err := s.workerClient.ResumeGame(ctx, room.WorkerAddr, roomID); err != nil {
		slog.Error("worker ResumeGame failed", "room_id", roomID, "error", err)
		return apperror.ErrWorkerUnavailable
	}
	slog.Info("game resumed", "room_id", roomID)
	return nil
}

// Stop 房主停止游戏
// 流程：校验房主身份 → 校验房间状态为 playing → 通知 Worker 杀 EmuRunner → 恢复非房主玩家为 spectator → 清除 Redis → 房间回到 waiting 态
func (s *RoomService) Stop(ctx context.Context, hostID uuid.UUID, roomID uuid.UUID) error {
	room, err := s.roomRepo.ByID(ctx, roomID)
	if err != nil || room == nil {
		return apperror.ErrRoomNotExist
	}
	if room.HostID != hostID {
		return apperror.ErrNotRoomHost
	}
	if room.Status != 1 {
		return apperror.ErrRoomNotPlaying
	}
	if room.WorkerAddr != "" {
		if err := s.workerClient.StopGame(ctx, room.WorkerAddr, roomID); err != nil {
			slog.Error("worker StopGame failed", "room_id", roomID, "error", err)
			return apperror.ErrWorkerUnavailable
		}
	}
	// 重置非房主玩家为 spectator，端口分配随旧 EmuRunner 失效
	players, err := s.roomPlayerRepo.ActiveByRoom(ctx, roomID)
	if err != nil {
		slog.Warn("query players for reset failed", "room_id", roomID, "error", err)
	} else {
		for _, p := range players {
			if p.Role != 0 {
				_ = s.roomPlayerRepo.UpdateRoleAndPort(ctx, p.ID, 2, nil)
			}
		}
	}
	// 清除 Redis（ports + livekit 随旧 EmuRunner 失效）
	_ = s.roomStateCache.ClearRoom(ctx, roomID)
	// 回到 waiting 态（0），允许房主重新开始游戏
	if err := s.roomRepo.UpdateStatus(ctx, roomID, 0); err != nil {
		return apperror.ErrInternal
	}
	slog.Info("game stopped, room back to waiting", "room_id", roomID)
	return nil
}

// Delete 房主删除房间
// 仅 waiting 态可删，标记所有玩家离开，清 Redis，房间状态=2
func (s *RoomService) Delete(ctx context.Context, hostID uuid.UUID, roomID uuid.UUID) error {
	room, err := s.roomRepo.ByID(ctx, roomID)
	if err != nil || room == nil {
		return apperror.ErrRoomNotExist
	}
	if room.HostID != hostID {
		return apperror.ErrNotRoomHost
	}
	if room.Status != 0 {
		return apperror.ErrRoomNotWaiting
	}
	players, err := s.roomPlayerRepo.ActiveByRoom(ctx, roomID)
	if err != nil {
		slog.Warn("query players for delete failed", "room_id", roomID, "error", err)
	} else {
		for _, p := range players {
			_ = s.roomPlayerRepo.MarkLeft(ctx, p.ID)
		}
	}
	_ = s.roomStateCache.ClearRoom(ctx, roomID)
	if err := s.roomRepo.UpdateStatus(ctx, roomID, 2); err != nil {
		return apperror.ErrInternal
	}
	slog.Info("room deleted", "room_id", roomID)
	return nil
}

// GetLivekitToken 获取房间的 LiveKit token（按用户独立生成）
// 游戏未开始（status != 1）时返回 Waiting=true
// 游戏进行中则调用 Worker 的 GeneratePlayerToken gRPC 生成该玩家专属 token
func (s *RoomService) GetLivekitToken(ctx context.Context, userID uuid.UUID, roomID uuid.UUID) (*contract.LivekitTokenResp, error) {
	room, err := s.roomRepo.ByID(ctx, roomID)
	if err != nil || room == nil {
		return nil, apperror.ErrRoomNotExist
	}

	if room.Status != 1 {
		return &contract.LivekitTokenResp{Waiting: true}, nil
	}

	if room.WorkerAddr == "" {
		return nil, apperror.ErrWorkerUnavailable
	}

	// 从 Redis 获取 LiveKit 地址和房间名
	livekitURL, livekitRoom, err := s.roomStateCache.GetLivekitInfo(ctx, roomID)
	if err != nil {
		slog.Error("failed to get livekit info from cache", "room_id", roomID, "error", err)
		return nil, apperror.ErrInternal
	}

	// 调用 Worker 为当前用户生成专属 player token
	playerToken, err := s.workerClient.GeneratePlayerToken(ctx, room.WorkerAddr, roomID, userID)
	if err != nil {
		slog.Error("failed to generate player token", "room_id", roomID, "user_id", userID, "error", err)
		return &contract.LivekitTokenResp{Waiting: true}, nil
	}

	return &contract.LivekitTokenResp{
		LivekitToken: playerToken,
		LivekitRoom:  livekitRoom,
		LivekitUrl:   livekitURL,
		Waiting:      false,
	}, nil
}

// GetMembers 获取房间成员列表
// 任何在房间中的玩家都可以查看成员列表
func (s *RoomService) GetMembers(ctx context.Context, userID uuid.UUID, roomID uuid.UUID) ([]contract.RoomMemberInfo, error) {
	if _, err := s.roomRepo.ByID(ctx, roomID); err != nil {
		return nil, apperror.ErrRoomNotExist
	}

	player, err := s.roomPlayerRepo.ByRoomAndUser(ctx, roomID, userID)
	if err != nil || player == nil {
		return nil, apperror.ErrNotInRoom
	}

	return s.roomPlayerRepo.ActiveByRoomWithUser(ctx, roomID)
}

// KickPlayer 房主踢出玩家
// 流程：校验房主身份 → 校验目标非房主且非自己 → 校验目标在房间中 → MarkLeft → 清除 Redis 端口映射
func (s *RoomService) KickPlayer(ctx context.Context, hostID uuid.UUID, req contract.KickPlayerReq) error {
	// RoomID 与 UserID 已经过 notnil_uuid 校验，必非 nil
	roomID := *req.RoomID
	targetUserID := *req.UserID

	room, err := s.roomRepo.ByID(ctx, roomID)
	if err != nil || room == nil {
		return apperror.ErrRoomNotExist
	}

	if room.HostID != hostID {
		return apperror.ErrNotRoomHost
	}

	if hostID == targetUserID {
		return apperror.ErrKickSelf
	}

	if targetUserID == room.HostID {
		return apperror.ErrKickHost
	}

	target, err := s.roomPlayerRepo.ByRoomAndUser(ctx, roomID, targetUserID)
	if err != nil || target == nil {
		return apperror.ErrNotInRoom
	}

	if err := s.roomPlayerRepo.MarkLeft(ctx, target.ID); err != nil {
		return apperror.ErrInternal
	}

	if target.Port != nil {
		s.roomStateCache.RemovePort(ctx, roomID, *target.Port)
	}

	return nil
}

// Leave 玩家离开房间
// 流程：标记left_at → 清除端口绑定 → 如果是房主离开则转移房主给下一位player，无player则移交给spectator，无人则关闭房间
func (s *RoomService) Leave(ctx context.Context, userID uuid.UUID, roomID uuid.UUID) error {
	room, err := s.roomRepo.ByID(ctx, roomID)
	if err != nil || room == nil {
		return apperror.ErrRoomNotExist
	}

	player, err := s.roomPlayerRepo.ByRoomAndUser(ctx, roomID, userID)
	if err != nil || player == nil {
		return apperror.ErrNotInRoom
	}

	if err := s.roomPlayerRepo.MarkLeft(ctx, player.ID); err != nil {
		return apperror.ErrInternal
	}

	if player.Port != nil {
		s.roomStateCache.RemovePort(ctx, roomID, *player.Port)
	}

	if room.HostID == userID {
		active, err := s.roomPlayerRepo.ActiveByRoom(ctx, roomID)
		if err != nil {
			return apperror.ErrInternal
		}

		var nextHost *model.RoomPlayer
		for i := range active {
			if active[i].Role == 1 {
				nextHost = &active[i]
				break
			}
		}
		if nextHost == nil {
			for i := range active {
				if active[i].Role == 2 {
					nextHost = &active[i]
					break
				}
			}
		}

		if nextHost != nil {
			if err := s.roomPlayerRepo.TransferHost(ctx, roomID, nextHost.UserID); err != nil {
				return apperror.ErrInternal
			}
		} else {
			if err := s.roomRepo.UpdateStatus(ctx, roomID, 2); err != nil {
				return apperror.ErrInternal
			}
			s.roomStateCache.ClearRoom(ctx, roomID)

			// 通知 Worker 停止游戏（fire-and-forget，不阻塞用户响应）
			if room.WorkerAddr != "" {
				go func() {
					if err := s.workerClient.StopGame(context.Background(), room.WorkerAddr, roomID); err != nil {
						slog.Warn("stop game on worker failed during leave", "worker", room.WorkerAddr, "room_id", roomID, "error", err)
					}
				}()
			}
		}
	}

	return nil
}

// saveStateMinioPath 构造存档在 MinIO 的存储路径
func saveStateMinioPath(roomID, saveStateID uuid.UUID) string {
	return "savestate/" + roomID.String() + "/" + saveStateID.String() + ".dat"
}

// SaveState 房主保存存档
// 流程：校验房主 + 房间 playing + 已选 ROM → 生成存档 ID 与 MinIO 预签名 PUT URL →
//
//	gRPC 通知 Worker（令 EmuRunner 序列化并上传）→ 落库 save_states 记录
func (s *RoomService) SaveState(ctx context.Context, hostID uuid.UUID, roomID uuid.UUID) (*model.SaveState, error) {
	room, err := s.roomRepo.ByID(ctx, roomID)
	if err != nil || room == nil {
		return nil, apperror.ErrRoomNotExist
	}
	if room.HostID != hostID {
		return nil, apperror.ErrNotRoomHost
	}
	if room.Status != 1 {
		return nil, apperror.ErrRoomNotPlaying
	}
	if room.RomID == nil {
		return nil, apperror.ErrRomNotSelected
	}
	if room.WorkerAddr == "" {
		return nil, apperror.ErrWorkerUnavailable
	}

	saveStateID := uuid.Must(uuid.NewV7())
	minioPath := saveStateMinioPath(roomID, saveStateID)

	// 生成预签名 PUT URL，Worker 用它上传状态二进制
	uploadURL, err := s.minioFunc.PresignedPutURL(ctx, s.bucket, minioPath, 5*time.Minute)
	if err != nil {
		slog.Error("failed to generate presigned put url", "room_id", roomID, "error", err)
		return nil, apperror.ErrInternal
	}

	// 通知 Worker：令 EmuRunner 序列化并上传到 MinIO
	size, err := s.workerClient.SaveState(ctx, room.WorkerAddr, roomID, saveStateID, uploadURL)
	if err != nil {
		slog.Error("worker SaveState failed", "room_id", roomID, "error", err)
		return nil, apperror.ErrSaveStateFailed
	}

	ss := &model.SaveState{
		ID:           saveStateID,
		RoomID:       roomID,
		Name:         time.Now().Format("2006-01-02 15:04:05"),
		EmulatorType: room.EmulatorType,
		RomID:        *room.RomID,
		MinioPath:    minioPath,
		Size:         size,
		CreatedBy:    hostID,
	}
	if err := s.saveStateRepo.Create(ctx, ss); err != nil {
		slog.Error("failed to persist save state", "room_id", roomID, "error", err)
		return nil, apperror.ErrInternal
	}

	slog.Info("save state created", "room_id", roomID, "save_state_id", saveStateID, "size", size)
	return ss, nil
}

// LoadState 房主读取存档
// 流程：校验房主 + 房间 playing → 查存档 → 三要素匹配校验（room_id / emulator_type / rom_id）→
//
//	生成 MinIO 预签名 GET URL → gRPC 通知 Worker（下载并令 EmuRunner 反序列化）
func (s *RoomService) LoadState(ctx context.Context, hostID uuid.UUID, req contract.LoadStateReq) error {
	roomID := *req.RoomID
	saveStateID := *req.SaveStateID

	room, err := s.roomRepo.ByID(ctx, roomID)
	if err != nil || room == nil {
		return apperror.ErrRoomNotExist
	}
	if room.HostID != hostID {
		return apperror.ErrNotRoomHost
	}
	if room.Status != 1 {
		return apperror.ErrRoomNotPlaying
	}
	if room.WorkerAddr == "" {
		return apperror.ErrWorkerUnavailable
	}

	ss, err := s.saveStateRepo.ByID(ctx, saveStateID)
	if err != nil {
		return apperror.ErrInternal
	}
	if ss == nil {
		return apperror.ErrSaveStateNotExist
	}

	// 三要素匹配校验：房间、模拟器类型、ROM 全部一致才允许读档
	if ss.RoomID != roomID || ss.EmulatorType != room.EmulatorType || room.RomID == nil || ss.RomID != *room.RomID {
		return apperror.ErrSaveStateMismatch
	}

	return s.loadStateToWorker(ctx, room, ss)
}

// LoadLatestState 房主加载当前机种+ROM 的最新存档
// 流程：校验房主 + 房间 playing + 已选 ROM → 取最新存档 → 通知 Worker 反序列化
func (s *RoomService) LoadLatestState(ctx context.Context, hostID uuid.UUID, roomID uuid.UUID) error {
	room, err := s.roomRepo.ByID(ctx, roomID)
	if err != nil || room == nil {
		return apperror.ErrRoomNotExist
	}
	if room.HostID != hostID {
		return apperror.ErrNotRoomHost
	}
	if room.Status != 1 {
		return apperror.ErrRoomNotPlaying
	}
	if room.RomID == nil {
		return apperror.ErrRomNotSelected
	}
	if room.WorkerAddr == "" {
		return apperror.ErrWorkerUnavailable
	}

	ss, err := s.saveStateRepo.LatestByRoomRom(ctx, roomID, room.EmulatorType, *room.RomID)
	if err != nil {
		return apperror.ErrInternal
	}
	if ss == nil {
		return apperror.ErrSaveStateNotExist
	}

	return s.loadStateToWorker(ctx, room, ss)
}

// loadStateToWorker 生成预签名 GET URL 并通知 Worker 下载 + 令 EmuRunner 反序列化
func (s *RoomService) loadStateToWorker(ctx context.Context, room *model.Room, ss *model.SaveState) error {
	downloadURL, err := s.minioFunc.PresignedGetURL(ctx, s.bucket, ss.MinioPath, 5*time.Minute)
	if err != nil {
		slog.Error("failed to generate presigned get url", "room_id", room.ID, "error", err)
		return apperror.ErrInternal
	}

	if err := s.workerClient.LoadState(ctx, room.WorkerAddr, room.ID, ss.ID, downloadURL); err != nil {
		slog.Error("worker LoadState failed", "room_id", room.ID, "error", err)
		return apperror.ErrLoadStateFailed
	}

	slog.Info("save state loaded", "room_id", room.ID, "save_state_id", ss.ID)
	return nil
}

// RenameSaveState 房主重命名存档
func (s *RoomService) RenameSaveState(ctx context.Context, hostID uuid.UUID, req contract.RenameSaveStateReq) error {
	roomID := *req.RoomID
	saveStateID := *req.SaveStateID

	room, err := s.roomRepo.ByID(ctx, roomID)
	if err != nil || room == nil {
		return apperror.ErrRoomNotExist
	}
	if room.HostID != hostID {
		return apperror.ErrNotRoomHost
	}

	ss, err := s.saveStateRepo.ByID(ctx, saveStateID)
	if err != nil {
		return apperror.ErrInternal
	}
	if ss == nil || ss.RoomID != roomID {
		return apperror.ErrSaveStateNotExist
	}

	if err := s.saveStateRepo.Rename(ctx, saveStateID, req.Name); err != nil {
		slog.Error("failed to rename save state", "save_state_id", saveStateID, "error", err)
		return apperror.ErrInternal
	}
	return nil
}

// DeleteSaveState 房主删除存档（先删 MinIO 二进制，再删数据库记录）
func (s *RoomService) DeleteSaveState(ctx context.Context, hostID uuid.UUID, req contract.DeleteSaveStateReq) error {
	roomID := *req.RoomID
	saveStateID := *req.SaveStateID

	room, err := s.roomRepo.ByID(ctx, roomID)
	if err != nil || room == nil {
		return apperror.ErrRoomNotExist
	}
	if room.HostID != hostID {
		return apperror.ErrNotRoomHost
	}

	ss, err := s.saveStateRepo.ByID(ctx, saveStateID)
	if err != nil {
		return apperror.ErrInternal
	}
	if ss == nil || ss.RoomID != roomID {
		return apperror.ErrSaveStateNotExist
	}

	// 先删 MinIO 二进制，失败仅记录日志不阻断（避免残留记录导致无法删除）
	if err := s.minioFunc.RemoveFile(ctx, s.bucket, ss.MinioPath); err != nil {
		slog.Warn("failed to remove save state object", "save_state_id", saveStateID, "path", ss.MinioPath, "error", err)
	}

	if err := s.saveStateRepo.Delete(ctx, saveStateID); err != nil {
		slog.Error("failed to delete save state", "save_state_id", saveStateID, "error", err)
		return apperror.ErrInternal
	}

	slog.Info("save state deleted", "room_id", roomID, "save_state_id", saveStateID)
	return nil
}

// ListSaveStates 列出房间存档（房间成员可查，创建时间倒序）
// 仅返回与房间当前模拟器类型、ROM 匹配的存档，避免展示其他游戏的存档
func (s *RoomService) ListSaveStates(ctx context.Context, userID uuid.UUID, roomID uuid.UUID) ([]model.SaveState, error) {
	room, err := s.roomRepo.ByID(ctx, roomID)
	if err != nil || room == nil {
		return nil, apperror.ErrRoomNotExist
	}

	player, err := s.roomPlayerRepo.ByRoomAndUser(ctx, roomID, userID)
	if err != nil || player == nil {
		return nil, apperror.ErrNotInRoom
	}

	// 房间尚未选择 ROM 时无匹配存档
	if room.RomID == nil {
		return []model.SaveState{}, nil
	}

	return s.saveStateRepo.ListByRoomRom(ctx, roomID, room.EmulatorType, *room.RomID)
}
