package emurunner

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"

	"github.com/StellarisJAY/cloudemu/internal/emurunner/backend"
	lksdk "github.com/livekit/server-sdk-go/v2"
)

type Instance struct {
	runner           *Runner
	videoEncoder     *X264Encoder
	publisher        *LiveKitPublisher
	inputMgr         *InputManager
	connectedMembers map[string]*lksdk.RemoteParticipant
	mutex            sync.Mutex
	upscaleEnabled   bool
	workDir          string // 共享工作目录（ROM 所在目录，= /tmp/cloudemu/{room_id}/），存档/读档文件在此
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
	instance.runner = NewRunner(emulatorType)
	return instance
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
	instance.publisher.OnPause = func() {
		instance.runner.Pause(context.TODO())
	}
	instance.publisher.OnResume = func() {
		instance.runner.Resume(context.TODO())
	}
	instance.publisher.OnSaveState = func() {
		if err := instance.runner.SaveState(instance.workDir); err != nil {
			slog.Error("save state failed", "error", err)
		}
	}
	instance.publisher.OnLoadState = func() {
		if err := instance.runner.LoadState(instance.workDir); err != nil {
			slog.Error("load state failed", "error", err)
		}
	}

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
	return nil
}

func (instance *Instance) Run(ctx context.Context) {
	// 分离模拟器线程和视频流编码线程
	go instance.EncoderLoop(ctx)

	instance.runner.Run(ctx)
}

func (instance *Instance) EncoderLoop(ctx context.Context) {
	encW := instance.runner.backend.AVInfo.BaseWidth * instance.runner.backend.ScaleFactor
	encH := instance.runner.backend.AVInfo.BaseHeight * instance.runner.backend.ScaleFactor
	for {
		select {
		case <-ctx.Done():
			return
		case frame := <-instance.runner.backend.FrameChan(): // 接受模拟器线程的画面帧
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
