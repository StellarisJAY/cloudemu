package worker

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

// Session EmuRunner 会话，表示一个运行中的模拟器子进程
// 每个游戏房间对应一个 Session，由 SessionManager 统一管理生命周期
type Session struct {
	RoomID       string    // 房间 ID
	Cmd          *exec.Cmd // 子进程命令
	StartedAt    time.Time // 启动时间
	Status       string    // 运行状态：running / stopped / crashed
	EmulatorType string    // 模拟器类型
	WorkDir      string    // 临时工作目录（ROM 文件所在目录），Stop 时清理
	mu           sync.Mutex
}

// SessionManager EmuRunner 子进程管理器
// 负责启动/停止 EmuRunner 子进程，监控进程状态
type SessionManager struct {
	mu            sync.RWMutex
	sessions      map[string]*Session // key = room_id
	emuRunnerPath string
	livekitHost   string
}

// NewSessionManager 创建 SessionManager 实例
// emuRunnerPath: EmuRunner 可执行文件路径
// livekitHost: LiveKit 服务地址，用于 EmuRunner 连接
func NewSessionManager(emuRunnerPath, livekitHost string) *SessionManager {
	return &SessionManager{
		sessions:      make(map[string]*Session),
		emuRunnerPath: emuRunnerPath,
		livekitHost:   livekitHost,
	}
}

// Start 启动 EmuRunner 子进程
// 流程：创建临时目录 → 下载 ROM（从 MinIO 预签名 URL）→ 启动 EmuRunner 子进程
// 命令：emurunner --publisher-host=... --token=... --room=... --rom=... --backend=... --host-identity=...
// hostUserID: 房主用户 ID，EmuRunner 启动时把房主默认绑定到 Port 0
func (m *SessionManager) Start(roomID, token, romPath, romURL, emulatorType, hostUserID string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	workDir := filepath.Join("/tmp/cloudemu", roomID)
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return nil, fmt.Errorf("create workdir for room %s: %w", roomID, err)
	}

	localRom := filepath.Join(workDir, "rom.dat")
	if err := downloadFile(localRom, romURL); err != nil {
		os.RemoveAll(workDir)
		return nil, fmt.Errorf("download rom for room %s: %w", roomID, err)
	}

	hostIdentity := "player:" + hostUserID
	cmd := exec.Command(m.emuRunnerPath,
		"--publisher-host", m.livekitHost,
		"--token", token,
		"--room", roomID,
		"--rom", localRom,
		"--backend", emulatorType,
		"--host-identity", hostIdentity,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		os.RemoveAll(workDir)
		return nil, fmt.Errorf("start emurunner for room %s: %w", roomID, err)
	}

	session := &Session{
		RoomID:       roomID,
		Cmd:          cmd,
		StartedAt:    time.Now(),
		Status:       "running",
		EmulatorType: emulatorType,
		WorkDir:      workDir,
	}

	m.sessions[roomID] = session

	go m.monitor(session)

	return session, nil
}

// downloadFile 从 URL 下载文件到本地路径
func downloadFile(localPath, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	out, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// Stop 停止指定房间的 EmuRunner 子进程
func (m *SessionManager) Stop(roomID string) error {
	m.mu.Lock()
	session, ok := m.sessions[roomID]
	delete(m.sessions, roomID)
	m.mu.Unlock()

	if !ok {
		return fmt.Errorf("session not found: %s", roomID)
	}

	session.mu.Lock()
	if session.Status != "running" {
		session.mu.Unlock()
		if session.WorkDir != "" {
			os.RemoveAll(session.WorkDir)
		}
		return nil
	}
	session.mu.Unlock()

	if err := session.Cmd.Process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("send SIGTERM to emurunner %s: %w", roomID, err)
	}

	done := make(chan error, 1)
	go func() {
		done <- session.Cmd.Wait()
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		session.Cmd.Process.Kill()
		<-done
	}

	session.mu.Lock()
	session.Status = "stopped"
	session.mu.Unlock()

	if session.WorkDir != "" {
		os.RemoveAll(session.WorkDir)
	}

	return nil
}

// Status 获取指定房间的会话状态
func (m *SessionManager) Status(roomID string) (string, int64) {
	m.mu.RLock()
	session, ok := m.sessions[roomID]
	m.mu.RUnlock()

	if !ok {
		return "stopped", 0
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	uptime := int64(0)
	if session.Status == "running" {
		uptime = int64(time.Since(session.StartedAt).Seconds())
	}

	return session.Status, uptime
}

// List 返回所有活跃会话的 room_id 列表
func (m *SessionManager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids := make([]string, 0, len(m.sessions))
	for id := range m.sessions {
		ids = append(ids, id)
	}
	return ids
}

// Count 返回当前活跃会话数量
func (m *SessionManager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}

// monitor 后台监控子进程，检测异常退出
func (m *SessionManager) monitor(session *Session) {
	_ = session.Cmd.Wait()

	session.mu.Lock()
	if session.Status == "running" {
		session.Status = "crashed"
		slog.Warn("session crashed", "room_id", session.RoomID)
	}
	session.mu.Unlock()
}

// StopAll 停止所有正在运行的 EmuRunner 子进程（用于 Worker 优雅关闭）
func (m *SessionManager) StopAll(ctx context.Context) {
	m.mu.Lock()
	ids := make([]string, 0, len(m.sessions))
	for id := range m.sessions {
		ids = append(ids, id)
	}
	m.mu.Unlock()

	for _, id := range ids {
		_ = m.Stop(id)
	}
}
