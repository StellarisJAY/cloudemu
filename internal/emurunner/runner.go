package emurunner

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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
