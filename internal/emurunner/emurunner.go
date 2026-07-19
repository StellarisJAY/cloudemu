package emurunner

import (
	"context"
	"fmt"
	"image"
	"log/slog"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/StellarisJAY/cloudemu/internal/emurunner/backend"
	lksdk "github.com/livekit/server-sdk-go/v2"
)

type Instance struct {
	runner           *Runner
	videoEncoder     *X264Encoder
	audioEncoder     *OpusEncoder
	publisher        *LiveKitPublisher
	inputMgr         *InputManager
	connectedMembers map[string]*lksdk.RemoteParticipant
	mutex            sync.Mutex
	upscaleEnabled   bool
	reloading        atomic.Bool // ROM 热切换中为 true，EncoderLoop 检查此标志跳过编码
	workDir          string      // 共享工作目录（ROM 所在目录，= /tmp/cloudemu/{room_id}/），存档/读档文件在此
}

// NewInstance 创建模拟器实例
// hostIdentity: 房主的 LiveKit identity（如 "player:{hostUserId}"），默认绑定到 Port 0
// upscaleEnabled: 是否开启整数倍 nearest-neighbor 放大以保留像素边缘锐度
func NewInstance(config LiveKitConfig, emulatorType backend.Type, hostIdentity string, upscaleEnabled bool) *Instance {
	instance := &Instance{}
	instance.connectedMembers = make(map[string]*lksdk.RemoteParticipant)
	instance.mutex = sync.Mutex{}
	instance.upscaleEnabled = upscaleEnabled
	instance.inputMgr = NewInputManager(hostIdentity)
	instance.publisher = NewLiveKitPublisher(config, instance.inputMgr)
	instance.videoEncoder = NewX264Encoder()
	instance.audioEncoder = NewOpusEncoder()
	instance.runner = NewRunner(emulatorType)
	return instance
}

// normalizeOpusSampleRate 将 libretro 核心报告的采样率标准化为 Opus 支持的合法值
// Opus 编码器要求: 8000, 12000, 16000, 24000, 48000 Hz
// 部分 libretro 核心返回 0 或 44100 等非标准值，需映射到最近的合法值
func normalizeOpusSampleRate(rate int) int {
	valid := []int{8000, 12000, 16000, 24000, 48000}
	if rate <= 0 {
		return 48000
	}
	best := valid[0]
	for _, v := range valid {
		if abs(rate-v) < abs(rate-best) {
			best = v
		}
	}
	return best
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func (instance *Instance) InitRunner(path string) error {
	// 共享工作目录 = ROM 文件所在目录（Worker 会把存档/读档文件放在此处）
	instance.workDir = filepath.Dir(path)
	// 初始化模拟器后端
	if err := instance.runner.Init(); err != nil {
		return fmt.Errorf("emurunner init backend error: %w", err)
	}
	// 注入输入管理器，供 libretro input_state 回调查询按键状态
	instance.runner.backend.SetInputProvider(instance.inputMgr)
	// 加载rom文件后，拿到视频输出宽高等信息，再初始化视频编码器
	if err := instance.runner.LoadROM(path); err != nil {
		return err
	}
	return nil
}

func (instance *Instance) InitPublisher() error {
	instance.publisher.OnMemberConnect = instance.OnMemberConnect
	instance.publisher.OnMemberDisconnect = instance.OnMemberDisconnect

	if err := instance.publisher.ConnectRoom(); err != nil {
		return fmt.Errorf("emurunner connect room error: %w", err)
	}

	if instance.upscaleEnabled {
		instance.runner.backend.SetScaleFactor(backend.ScaleFactorForType(instance.runner.Type))
	} else {
		instance.runner.backend.SetScaleFactor(1)
	}
	encW := instance.runner.backend.AVInfo.BaseWidth * instance.runner.backend.ScaleFactor
	encH := instance.runner.backend.AVInfo.BaseHeight * instance.runner.backend.ScaleFactor
	if err := instance.videoEncoder.Init(encW, encH); err != nil {
		return fmt.Errorf("emurunner init video encoder error: %w", err)
	}
	// 初始化音频编码器
	rawRate := int(instance.runner.backend.AVInfo.SampleRate)
	sampleRate := normalizeOpusSampleRate(rawRate)
	channels := instance.runner.AudioChannels()
	latencyMs := 20 // 20ms Opus 帧
	slog.Info("init audio encoder", "libretroRate", rawRate, "normalizedRate", sampleRate, "channels", channels)
	instance.runner.backend.SetAudioConfig(float64(sampleRate), channels, latencyMs)
	if err := instance.audioEncoder.Init(sampleRate, channels); err != nil {
		return fmt.Errorf("emurunner init audio encoder error: %w", err)
	}
	return nil
}

func (instance *Instance) Run(ctx context.Context) {
	// 分离模拟器线程和视频流编码线程
	go instance.EncoderLoop(ctx)

	instance.runner.Run(ctx)
}

// ReloadROM 热切换 ROM 文件，保持 LiveKit 连接不断
// 流程：暂停模拟器 → 排空编码队列 → 关闭旧编码器 → 卸载旧 ROM → 加载新 ROM → 重新初始化编码器 → 恢复模拟
func (instance *Instance) ReloadROM(newRomPath string) error {
	instance.reloading.Store(true)
	defer instance.reloading.Store(false)

	// 1. 暂停模拟器 tick，停止生成新的帧/音频数据
	instance.runner.Pause(context.TODO())
	// 等待当前帧完成渲染
	time.Sleep(50 * time.Millisecond)

	// 2. 排空 channel 中残留的帧/音频数据
	drainFrameChan(instance.runner.backend.FrameChan())
	drainAudioChan(instance.runner.backend.AudioChan())

	// 3. 关闭旧编码器
	if instance.videoEncoder.enc != nil {
		_ = instance.videoEncoder.enc.Close()
	}
	if instance.audioEncoder.enc != nil {
		_ = instance.audioEncoder.enc.Close()
	}

	// 4. 卸载旧 ROM（调用 retro_unload_game）
	instance.runner.UnloadROM()

	// 5. 加载新 ROM
	if err := instance.runner.LoadROM(newRomPath); err != nil {
		return fmt.Errorf("load new rom: %w", err)
	}

	// 6. 重新初始化视频编码器（新 ROM 分辨率可能不同）
	if instance.upscaleEnabled {
		instance.runner.backend.SetScaleFactor(backend.ScaleFactorForType(instance.runner.Type))
	} else {
		instance.runner.backend.SetScaleFactor(1)
	}
	encW := instance.runner.backend.AVInfo.BaseWidth * instance.runner.backend.ScaleFactor
	encH := instance.runner.backend.AVInfo.BaseHeight * instance.runner.backend.ScaleFactor
	if err := instance.videoEncoder.Init(encW, encH); err != nil {
		return fmt.Errorf("reinit video encoder: %w", err)
	}

	// 7. 重新初始化音频编码器
	rawRate := int(instance.runner.backend.AVInfo.SampleRate)
	sampleRate := normalizeOpusSampleRate(rawRate)
	channels := instance.runner.AudioChannels()
	instance.runner.backend.SetAudioConfig(float64(sampleRate), channels, 20)
	if err := instance.audioEncoder.Init(sampleRate, channels); err != nil {
		return fmt.Errorf("reinit audio encoder: %w", err)
	}
	slog.Info("audio encoder reinitialized", "libretroRate", rawRate, "normalizedRate", sampleRate, "channels", channels)

	// 8. 更新工作目录为新 ROM 所在目录
	instance.workDir = filepath.Dir(newRomPath)

	// 9. 恢复模拟器运行
	instance.runner.Resume(context.TODO())

	slog.Info("rom reloaded", "path", newRomPath, "encW", encW, "encH", encH)
	return nil
}

// drainFrameChan 排空帧通道中的残留数据
func drainFrameChan(ch <-chan *image.YCbCr) {
	for {
		select {
		case <-ch:
		default:
			return
		}
	}
}

// drainAudioChan 排空音频通道中的残留数据
func drainAudioChan(ch <-chan []int16) {
	for {
		select {
		case <-ch:
		default:
			return
		}
	}
}

func (instance *Instance) EncoderLoop(ctx context.Context) {
	encW := instance.runner.backend.AVInfo.BaseWidth * instance.runner.backend.ScaleFactor
	encH := instance.runner.backend.AVInfo.BaseHeight * instance.runner.backend.ScaleFactor
	sampleRate := normalizeOpusSampleRate(int(instance.runner.backend.AVInfo.SampleRate))
	audioChannels := instance.runner.AudioChannels()
	for {
		select {
		case <-ctx.Done():
			return
		case frame := <-instance.runner.backend.FrameChan(): // 接受模拟器线程的画面帧
			// ROM 热切换中，跳过编码避免使用已关闭的编码器
			if instance.reloading.Load() {
				continue
			}
			// 视频流编码
			sample, err := instance.videoEncoder.Encode(frame, encW, encH)
			if err != nil {
				slog.Error("encode video frame failed", "error", err)
				continue
			}
			// 发布到房间的视频轨道
			if err := instance.publisher.WriteVideoSample(sample); err != nil {
				slog.Error("write video sample failed", "error", err)
			}
		case pcm := <-instance.runner.backend.AudioChan(): // 接受模拟器线程的音频帧
			if instance.reloading.Load() {
				continue
			}
			sample, err := instance.audioEncoder.Encode(pcm, sampleRate, audioChannels)
			if err != nil {
				slog.Error("encode audio failed", "error", err)
				continue
			}
			if err := instance.publisher.WriteAudioSample(sample); err != nil {
				slog.Error("write audio sample failed", "error", err)
			}
		}
	}
}

// OnMemberConnect 玩家连接到房间，如果房间之前无人，则需要启动之前暂停的模拟器
func (instance *Instance) OnMemberConnect(member *lksdk.RemoteParticipant) {
	instance.mutex.Lock()
	defer instance.mutex.Unlock()
	instance.connectedMembers[member.Identity()] = member
	slog.Info("member connected to room", "id", member.Identity())
	if len(instance.connectedMembers) > 0 {
		instance.runner.Resume(context.TODO())
	}
}

// OnMemberDisconnect 玩家离开房间，如果离开后房间无人，则需要暂停模拟器循环
func (instance *Instance) OnMemberDisconnect(member *lksdk.RemoteParticipant) {
	instance.mutex.Lock()
	defer instance.mutex.Unlock()
	delete(instance.connectedMembers, member.Identity())
	slog.Info("member disconnected", "id", member.Identity())
	if len(instance.connectedMembers) == 0 {
		instance.runner.Pause(context.TODO())
	}
}
