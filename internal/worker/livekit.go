package worker

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
)

// LiveKitManager LiveKit 房间管理，负责创建/删除 LiveKit 房间和生成 access token
// Worker Agent 使用 LiveKit 服务端 SDK 管理房间生命周期
type LiveKitManager struct {
	host      string
	apiKey    string
	apiSecret string
	client    *lksdk.RoomServiceClient
}

// NewLiveKitManager 创建 LiveKitManager 实例
// host: LiveKit 服务地址，e.g. "http://localhost:7880"
func NewLiveKitManager(host, apiKey, apiSecret string) *LiveKitManager {
	slog.Info("livekit api auth", "api_key", apiKey, "api_secret", apiSecret)
	return &LiveKitManager{
		host:      host,
		apiKey:    apiKey,
		apiSecret: apiSecret,
		client:    lksdk.NewRoomServiceClient(host, apiKey, apiSecret),
	}
}

// CreateRoom 创建 LiveKit 房间
// roomName: 房间名（使用 room_id UUIDv7 字符串）
// 设置空房间超时 60s，所有参与者离开后自动清理
func (m *LiveKitManager) CreateRoom(ctx context.Context, roomName string) error {
	_, err := m.client.CreateRoom(ctx, &livekit.CreateRoomRequest{
		Name:            roomName,
		EmptyTimeout:    60, // 空房间 60s 后自动删除
		MaxParticipants: 10, // 最多 10 个参与者（含 EmuRunner + 玩家）
	})
	if err != nil {
		return fmt.Errorf("create livekit room %s: %w", roomName, err)
	}
	return nil
}

// GenerateToken 生成 LiveKit access token
// roomName: 房间名
// identity: 参与者身份标识（emurunner 或 player:{user_id}）
// canPublish: 是否允许发布 track（EmuRunner=true，玩家=false）
func (m *LiveKitManager) GenerateToken(roomName, identity string, canPublish bool) (string, error) {
	at := auth.NewAccessToken(m.apiKey, m.apiSecret)
	at.SetName(identity)
	at.SetIdentity(identity)

	grant := &auth.VideoGrant{
		RoomJoin: true,
		Room:     roomName,
	}

	if canPublish {
		grant.SetCanPublish(true)
		grant.SetCanPublishData(true)
	}

	grant.SetCanSubscribe(true)
	at.SetVideoGrant(grant)

	token, err := at.ToJWT()
	if err != nil {
		return "", fmt.Errorf("generate livekit token: %w", err)
	}

	return token, nil
}

// DeleteRoom 删除 LiveKit 房间
func (m *LiveKitManager) DeleteRoom(ctx context.Context, roomName string) error {
	_, err := m.client.DeleteRoom(ctx, &livekit.DeleteRoomRequest{
		Room: roomName,
	})
	if err != nil {
		return fmt.Errorf("delete livekit room %s: %w", roomName, err)
	}
	return nil
}

// SendDataBroadcast 向房间内所有参与者广播一条 DataPacket
// topic: LiveKit DataPacket topic 字符串（如 "control"），用于接收端按 topic 分发
// reliable=true 走 reliable channel（不丢包，用于控制消息）；false 走 lossy（低延迟，用于实时输入）
func (m *LiveKitManager) SendDataBroadcast(ctx context.Context, roomName, topic string, reliable bool, data []byte) error {
	kind := livekit.DataPacket_RELIABLE
	if !reliable {
		kind = livekit.DataPacket_LOSSY
	}
	topicValue := topic
	_, err := m.client.SendData(ctx, &livekit.SendDataRequest{
		Room:  roomName,
		Data:  data,
		Kind:  kind,
		Topic: &topicValue,
	})
	if err != nil {
		return fmt.Errorf("livekit SendData room=%s topic=%s: %w", roomName, topic, err)
	}
	return nil
}

// HostURL 返回 LiveKit 服务端地址，供 EmuRunner 和前端连接使用
func (m *LiveKitManager) HostURL() string {
	return m.host
}
