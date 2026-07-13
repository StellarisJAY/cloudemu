package emurunner

import (
	"encoding/binary"
	"log/slog"
	"time"

	lksdk "github.com/livekit/server-sdk-go/v2"
	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
)

type LiveKitConfig struct {
	HostURL string
	Token   string
	RoomID  string
}

// DataChannel 协议 type prefix
const (
	packetTypeInput   byte = 0x01 // 玩家手柄输入：[type][buttons_lo][buttons_hi][reserved]
	packetTypePortMap byte = 0x02 // 端口映射更新：[type][count] + entries
	packetTypePing    byte = 0x03 // 延迟探测请求：[type][client_ts:8B LE]
	packetTypePong    byte = 0x04 // 延迟探测回复：[type][client_ts:8B LE][server_ts:8B LE]
	packetTypePause   byte = 0x05 // 暂停模拟器运行
	packetTypeResume  byte = 0x06 // 继续模拟器运行
)

// DataChannel topic 字符串
const (
	topicInput   = "input"   // 玩家输入
	topicControl = "control" // 服务端控制消息（如 PORT_MAP）
	topicPing    = "ping"    // 延迟探测
)

type LiveKitPublisher struct {
	config   LiveKitConfig
	room     *lksdk.Room
	inputMgr *InputManager // 输入管理器，OnDataPacket 收到后写入

	videoTrack *webrtc.TrackLocalStaticSample
	audioTrack *webrtc.TrackLocalStaticSample

	OnMemberConnect    func(*lksdk.RemoteParticipant)
	OnMemberDisconnect func(*lksdk.RemoteParticipant)
	OnPause            func() // handleControlPacket 收到 type=0x05 时调用
	OnResume           func() // handleControlPacket 收到 type=0x06 时调用
}

func NewLiveKitPublisher(config LiveKitConfig, inputMgr *InputManager) *LiveKitPublisher {
	return &LiveKitPublisher{config: config, inputMgr: inputMgr}
}

func (l *LiveKitPublisher) ConnectRoom() error {
	// 房间事件回调
	cb := lksdk.NewRoomCallback()
	// 用户加入
	cb.OnParticipantConnected = func(rp *lksdk.RemoteParticipant) {
		if l.OnMemberConnect != nil {
			l.OnMemberConnect(rp)
		}
	}
	// 用户离开
	cb.OnParticipantDisconnected = func(rp *lksdk.RemoteParticipant) {
		if l.OnMemberDisconnect != nil {
			l.OnMemberDisconnect(rp)
		}
	}
	// DataChannel 包路由：按 topic 分发到 input/control 处理器
	cb.OnDataPacket = func(data lksdk.DataPacket, params lksdk.DataReceiveParams) {
		userData, ok := data.(*lksdk.UserDataPacket)
		if !ok {
			return
		}
		topic := userData.Topic
		if topic == "" {
			topic = params.Topic
		}
		switch topic {
		case topicInput:
			l.handleInputPacket(params.SenderIdentity, userData.Payload)
		case topicControl:
			l.handleControlPacket(params.SenderIdentity, userData.Payload)
		case topicPing:
			l.handlePingPacket(params.SenderIdentity, userData.Payload)
		default:
			// 未知 topic 直接忽略
		}
	}
	// 用token连接到livekit房间
	room, err := lksdk.ConnectToRoomWithToken(l.config.HostURL, l.config.Token, cb)
	if err != nil {
		return err
	}
	l.room = room
	// 创建视频、音频轨道
	videoTrack, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{
		MimeType: webrtc.MimeTypeH264,
	}, "cloudemu-video", "cloudemu-video")
	if err != nil {
		return err
	}
	audioTrack, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{
		MimeType: webrtc.MimeTypeOpus,
	}, "cloudemu-audio", "cloudemu-audio")
	if err != nil {
		return err
	}
	// 添加视频轨道到livekit房间
	_, err = room.LocalParticipant.PublishTrack(videoTrack, &lksdk.TrackPublicationOptions{})
	if err != nil {
		return err
	}
	// 添加音频轨道到livekit房间
	_, err = room.LocalParticipant.PublishTrack(audioTrack, &lksdk.TrackPublicationOptions{})
	if err != nil {
		return err
	}
	l.videoTrack = videoTrack
	l.audioTrack = audioTrack
	return nil
}

// handleInputPacket 解析玩家输入包并写入 InputManager
// 协议：[type=0x01][buttons_lo][buttons_hi][reserved]，共 4 bytes
// buttons 是 uint16 little-endian，bit i 对应 libretro RETRO_DEVICE_ID_JOYPAD_*
func (l *LiveKitPublisher) handleInputPacket(senderIdentity string, payload []byte) {
	if l.inputMgr == nil {
		return
	}
	if len(payload) < 3 {
		return
	}
	if payload[0] != packetTypeInput {
		return
	}
	// 按 little-endian 解析 buttons
	state := uint16(payload[1]) | (uint16(payload[2]) << 8)
	l.inputMgr.UpdateInput(senderIdentity, state)
}

// handleControlPacket 解析服务端下发的控制包，按 type byte 分发
// 该包由 Worker 通过 LiveKit Server SDK SendData 广播，sender 是服务端，无 identity
// 支持的 type：
//
//	0x02 = PORT_MAP 端口映射更新
//	0x05 = Pause 暂停模拟器
//	0x06 = Resume 继续模拟器
func (l *LiveKitPublisher) handleControlPacket(senderIdentity string, payload []byte) {
	if len(payload) < 1 {
		return
	}
	switch payload[0] {
	case packetTypePortMap:
		l.handlePortMapPacket(payload)
	case packetTypePause:
		slog.Info("pause command received via control channel")
		if l.OnPause != nil {
			l.OnPause()
		}
	case packetTypeResume:
		slog.Info("resume command received via control channel")
		if l.OnResume != nil {
			l.OnResume()
		}
	default:
		slog.Warn("unknown control packet type", "type", payload[0])
	}
}

// handlePortMapPacket 解析 PORT_MAP 控制包并更新 InputManager
// 协议：[type=0x02][count] + count 个 [port][identity_len][identity_bytes]
func (l *LiveKitPublisher) handlePortMapPacket(payload []byte) {
	if l.inputMgr == nil {
		return
	}
	if len(payload) < 2 {
		return
	}
	count := int(payload[1])
	entries := make([]PortEntry, 0, count)
	offset := 2
	for i := 0; i < count; i++ {
		if offset+2 > len(payload) {
			slog.Warn("port_map packet truncated", "offset", offset, "len", len(payload))
			return
		}
		port := int(payload[offset])
		offset++
		idLen := int(payload[offset])
		offset++
		if offset+idLen > len(payload) {
			slog.Warn("port_map identity truncated", "offset", offset, "id_len", idLen, "len", len(payload))
			return
		}
		identity := string(payload[offset : offset+idLen])
		offset += idLen
		entries = append(entries, PortEntry{Port: port, Identity: identity})
	}
	l.inputMgr.UpdatePortMapping(entries)
	slog.Info("port mapping updated", "entries", len(entries))
}

// handlePingPacket 处理玩家发出的延迟探测请求
// 协议：[type=0x03][client_ts:8B int64 LE]
// 回复 pong 包定向回复给发起者，避免对其他玩家造成噪声
func (l *LiveKitPublisher) handlePingPacket(senderIdentity string, payload []byte) {
	if len(payload) < 9 {
		return
	}
	if payload[0] != packetTypePing {
		return
	}
	clientTs := int64(binary.LittleEndian.Uint64(payload[1:9]))
	serverTs := time.Now().UnixMilli()

	pong := make([]byte, 17)
	pong[0] = packetTypePong
	binary.LittleEndian.PutUint64(pong[1:9], uint64(clientTs))
	binary.LittleEndian.PutUint64(pong[9:17], uint64(serverTs))

	if err := l.room.LocalParticipant.PublishData(pong,
		lksdk.WithDataPublishTopic(topicPing),
		lksdk.WithDataPublishReliable(false),
		lksdk.WithDataPublishDestination([]string{senderIdentity}),
	); err != nil {
		slog.Warn("failed to send pong", "identity", senderIdentity, "error", err)
	}
}

func (l *LiveKitPublisher) WriteVideoSample(sample media.Sample) error {
	return l.videoTrack.WriteSample(sample)
}

func (l *LiveKitPublisher) WriteAudioSample(sample media.Sample) error {
	return l.audioTrack.WriteSample(sample)
}

func (l *LiveKitPublisher) Disconnect() {
	if l.room != nil {
		l.room.Disconnect()
	}
}
