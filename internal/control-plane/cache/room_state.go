package cache

import (
	"context"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// RoomState 房间实时状态缓存，基于 Redis Hash，实现 contract.RoomStateCache 接口
// Key 格式:
//
//	room:{room_id}:ports    -> Hash {port: user_id} 端口绑定
//	room:{room_id}:livekit  -> Hash {url, room}      LiveKit 连接信息
type RoomState struct {
	cli *redis.Client
}

func NewRoomState(cli *redis.Client) *RoomState {
	return &RoomState{cli: cli}
}

func (r *RoomState) portsKey(roomID uuid.UUID) string {
	return fmt.Sprintf("room:%s:ports", roomID.String())
}

func (r *RoomState) livekitKey(roomID uuid.UUID) string {
	return fmt.Sprintf("room:%s:livekit", roomID.String())
}

// SetPort 设置某端口绑定的玩家
func (r *RoomState) SetPort(ctx context.Context, roomID uuid.UUID, port int16, userID uuid.UUID) error {
	return r.cli.HSet(ctx, r.portsKey(roomID), strconv.Itoa(int(port)), userID.String()).Err()
}

// RemovePort 移除某端口绑定
func (r *RoomState) RemovePort(ctx context.Context, roomID uuid.UUID, port int16) error {
	return r.cli.HDel(ctx, r.portsKey(roomID), strconv.Itoa(int(port))).Err()
}

// GetPorts 获取房间所有端口映射 {port: user_id}
func (r *RoomState) GetPorts(ctx context.Context, roomID uuid.UUID) (map[int16]uuid.UUID, error) {
	result, err := r.cli.HGetAll(ctx, r.portsKey(roomID)).Result()
	if err != nil {
		return nil, err
	}
	ports := make(map[int16]uuid.UUID, len(result))
	for k, v := range result {
		p, err := strconv.Atoi(k)
		if err != nil {
			continue
		}
		uid, err := uuid.Parse(v)
		if err != nil {
			continue
		}
		ports[int16(p)] = uid
	}
	return ports, nil
}

// ClearRoom 清除房间所有缓存数据（房间关闭时调用）
func (r *RoomState) ClearRoom(ctx context.Context, roomID uuid.UUID) error {
	return r.cli.Del(ctx, r.portsKey(roomID), r.livekitKey(roomID)).Err()
}

// SetLivekitInfo 存储 LiveKit 地址和房间名
func (r *RoomState) SetLivekitInfo(ctx context.Context, roomID uuid.UUID, livekitUrl string, room string) error {
	return r.cli.HSet(ctx, r.livekitKey(roomID), map[string]string{
		"url":  livekitUrl,
		"room": room,
	}).Err()
}

// GetLivekitInfo 获取 LiveKit 地址和房间名
func (r *RoomState) GetLivekitInfo(ctx context.Context, roomID uuid.UUID) (string, string, error) {
	result, err := r.cli.HGetAll(ctx, r.livekitKey(roomID)).Result()
	if err != nil {
		return "", "", err
	}
	return result["url"], result["room"], nil
}
