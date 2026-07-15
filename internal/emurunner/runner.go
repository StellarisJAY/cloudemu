package emurunner

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/StellarisJAY/cloudemu/internal/emurunner/backend"
)

// Runner 每次游戏会话对应一个 Runner 实例
type Runner struct {
	backend *backend.LibretroBackend
	Type    backend.Type

	pauseChan  chan struct{}
	resumeChan chan struct{}
}

// NewRunner 创建新的 Runner 实例
func NewRunner(backendType backend.Type) *Runner {
	return &Runner{
		backend:    backend.NewBackend(),
		Type:       backendType,
		pauseChan:  make(chan struct{}),
		resumeChan: make(chan struct{}),
	}
}

func (r *Runner) Init() error {
	backendFile, err := backend.GetBackendFilePath(r.Type)
	if err != nil {
		return fmt.Errorf("load backend failed %w", err)
	}
	runtime.LockOSThread()
	if err := r.backend.LoadBackend(backendFile); err != nil {
		return err
	}
	slog.Info("loaded libretro backend", "type", r.Type, "file", backendFile)
	r.backend.Init()
	return nil
}

func (r *Runner) LoadROM(path string) error {
	if ok := r.backend.LoadGameFile(path); !ok {
		return errors.New("load rom failed")
	}
	r.backend.GetSysAVInfo()
	slog.Info("rom loaded", "path", path, "av_width", r.backend.AVInfo.BaseWidth, "av_height", r.backend.AVInfo.BaseHeight, "pixel_format", r.backend.PixelFormat)
	r.backend.GetPixelFormat()
	slog.Info("pixel format", "pf", r.backend.PixelFormat)
	return nil
}

func (r *Runner) Run(ctx context.Context) {
	fps := r.backend.AVInfo.FPS
	interval := time.Duration(1000000000 / fps)
	slog.Info("emurunner running", "frame_interval", interval.Milliseconds())
	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ticker.C:
			r.backend.Run()
		case <-ctx.Done():
			return
		case <-r.pauseChan:
			ticker.Stop()
		case <-r.resumeChan:
			ticker.Reset(interval)
		}
	}
}

func (r *Runner) Pause(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	case r.pauseChan <- struct{}{}:
		return
	}
}

func (r *Runner) Resume(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	case r.resumeChan <- struct{}{}:
		return
	}
}

// AudioChannels 根据模拟器类型返回音频声道数
// NES=1（单声道），GB/DOS=2（立体声）
func (r *Runner) AudioChannels() int {
	switch r.Type {
	case backend.BackEndTypeNES:
		return 1
	default:
		return 2
	}
}

// 共享目录中的存档/读档文件名（Worker 与 EmuRunner 同主机共享 /tmp/cloudemu/{room_id}/）
const (
	saveStateFile = "state.dat" // EmuRunner 序列化写入
	saveDoneFile  = "state.done" // EmuRunner 序列化完成标志（原子 rename）
	loadStateFile = "load.dat"   // Worker 下载存档写入，EmuRunner 读取
	loadDoneFile  = "load.done"  // EmuRunner 反序列化完成标志
)

// stateSerializer 存档序列化接口，便于单元测试注入 fake（真实实现为 *backend.LibretroBackend）
type stateSerializer interface {
	Serialize() ([]byte, error)
	Unserialize(data []byte) error
}

// SaveState 序列化当前状态并写入共享目录，完成后原子写出完成标志文件
// dir: 共享工作目录（/tmp/cloudemu/{room_id}/）
func (r *Runner) SaveState(dir string) error {
	return saveStateTo(r.backend, dir)
}

// LoadState 从共享目录读取存档数据并反序列化，完成后写出完成标志文件
func (r *Runner) LoadState(dir string) error {
	return loadStateFrom(r.backend, dir)
}

// saveStateTo 存档核心逻辑（可测试）：序列化 → 写临时文件 → 原子 rename 为 state.dat → 写 state.done
func saveStateTo(s stateSerializer, dir string) error {
	data, err := s.Serialize()
	if err != nil {
		return fmt.Errorf("serialize failed: %w", err)
	}
	tmp := filepath.Join(dir, saveStateFile+".tmp")
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("write state tmp failed: %w", err)
	}
	if err := os.Rename(tmp, filepath.Join(dir, saveStateFile)); err != nil {
		return fmt.Errorf("rename state failed: %w", err)
	}
	// 原子写出完成标志：先写 .tmp 再 rename，Worker 轮询到 state.done 即视为 state.dat 就绪
	doneTmp := filepath.Join(dir, saveDoneFile+".tmp")
	if err := os.WriteFile(doneTmp, []byte("ok"), 0644); err != nil {
		return fmt.Errorf("write done tmp failed: %w", err)
	}
	if err := os.Rename(doneTmp, filepath.Join(dir, saveDoneFile)); err != nil {
		return fmt.Errorf("rename done failed: %w", err)
	}
	slog.Info("save state written", "dir", dir, "size", len(data))
	return nil
}

// loadStateFrom 读档核心逻辑（可测试）：读 load.dat → 反序列化 → 写 load.done
func loadStateFrom(s stateSerializer, dir string) error {
	data, err := os.ReadFile(filepath.Join(dir, loadStateFile))
	if err != nil {
		return fmt.Errorf("read load state failed: %w", err)
	}
	if err := s.Unserialize(data); err != nil {
		return fmt.Errorf("unserialize failed: %w", err)
	}
	doneTmp := filepath.Join(dir, loadDoneFile+".tmp")
	if err := os.WriteFile(doneTmp, []byte("ok"), 0644); err != nil {
		return fmt.Errorf("write load done tmp failed: %w", err)
	}
	if err := os.Rename(doneTmp, filepath.Join(dir, loadDoneFile)); err != nil {
		return fmt.Errorf("rename load done failed: %w", err)
	}
	slog.Info("load state applied", "dir", dir, "size", len(data))
	return nil
}
