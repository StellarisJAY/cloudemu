package worker

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
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

	"github.com/StellarisJAY/cloudemu/internal/emurunner"
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

	// stdin/stdout 管道通信（Worker ↔ EmuRunner）
	cmdIn      io.WriteCloser // Worker 写 → EmuRunner stdin（os.Pipe writer）
	cmdOut     *bufio.Scanner // Worker 读 ← EmuRunner stdout（scanner 包裹 os.Pipe reader）
	cmdOutFile *os.File       // stdout pipe reader（用于 Stop 时关闭）
	cmdMu      sync.Mutex     // 串行化命令发送，同一时刻只允许一个命令
}

// SendCommand 向 EmuRunner 发送命令并等待响应
// 串行化：同一时刻只允许一个命令在执行，防止 stdout 响应错乱
func (s *Session) SendCommand(cmd emurunner.Cmd) (emurunner.Resp, error) {
	// 重要，串行化发送命令，避免命令响应混乱
	s.cmdMu.Lock()
	defer s.cmdMu.Unlock()

	if s.cmdIn == nil || s.cmdOut == nil {
		return emurunner.Resp{}, fmt.Errorf("pipe not initialized")
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		return emurunner.Resp{}, fmt.Errorf("marshal command: %w", err)
	}
	if _, err := s.cmdIn.Write(append(data, '\n')); err != nil {
		return emurunner.Resp{}, fmt.Errorf("write command: %w", err)
	}

	if !s.cmdOut.Scan() {
		if scanErr := s.cmdOut.Err(); scanErr != nil {
			return emurunner.Resp{}, fmt.Errorf("read response: %w", scanErr)
		}
		return emurunner.Resp{}, fmt.Errorf("emurunner stdout closed")
	}

	var resp emurunner.Resp
	if err := json.Unmarshal(s.cmdOut.Bytes(), &resp); err != nil {
		return emurunner.Resp{}, fmt.Errorf("unmarshal response: %w", err)
	}
	return resp, nil
}

// closePipes 关闭 stdin/stdout 管道，通知 EmuRunner 准备退出并释放 FD
func (s *Session) closePipes() {
	if s.cmdIn != nil {
		_ = s.cmdIn.Close()
	}
	if s.cmdOutFile != nil {
		_ = s.cmdOutFile.Close()
	}
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

// Get 按 roomID 获取活跃会话，不存在时返回 false
func (m *SessionManager) Get(roomID string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, ok := m.sessions[roomID]
	return session, ok
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

	// 创建 stdin 管道：Worker 写入 → EmuRunner 读（通过 os.Pipe 跨进程通信）
	stdinReader, stdinWriter, err := os.Pipe()
	if err != nil {
		os.RemoveAll(workDir)
		return nil, fmt.Errorf("create stdin pipe: %w", err)
	}
	cmd.Stdin = stdinReader

	// 创建 stdout 管道：EmuRunner 写入 → Worker 读
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		stdinWriter.Close()
		stdinReader.Close()
		os.RemoveAll(workDir)
		return nil, fmt.Errorf("create stdout pipe: %w", err)
	}
	cmd.Stdout = stdoutWriter

	if err := cmd.Start(); err != nil {
		stdinWriter.Close()
		stdinReader.Close()
		stdoutWriter.Close()
		stdoutReader.Close()
		os.RemoveAll(workDir)
		return nil, fmt.Errorf("start emurunner for room %s: %w", roomID, err)
	}

	// 子进程已持有 stdin reader / stdout writer 的副本，Worker 侧关闭这些 FD
	stdinReader.Close()
	stdoutWriter.Close()

	session := &Session{
		RoomID:       roomID,
		Cmd:          cmd,
		StartedAt:    time.Now(),
		Status:       "running",
		EmulatorType: emulatorType,
		WorkDir:      workDir,
		cmdIn:        stdinWriter,
		cmdOut:       bufio.NewScanner(stdoutReader),
		cmdOutFile:   stdoutReader,
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
// 流程：关闭管道 → SIGTERM → 5s 超时 → SIGKILL → 清理 workDir
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
		// 清理残留管道
		session.closePipes()
		if session.WorkDir != "" {
			os.RemoveAll(session.WorkDir)
		}
		return nil
	}
	session.mu.Unlock()

	// 关闭 stdin 管道，EmuRunner 的 StartCommandReader 协程收到 EOF 后退出
	session.closePipes()

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

// 共享目录中的存档/读档文件名（与 EmuRunner runner.go 保持一致）
const (
	saveStateFile = "state.dat"
	saveDoneFile  = "state.done"
	loadStateFile = "load.dat"
	loadDoneFile  = "load.done"

	stateWaitTimeout  = 10 * time.Second       // 等待 EmuRunner 完成序列化/反序列化的超时
	stateWaitInterval = 100 * time.Millisecond // 轮询完成标志文件的间隔
)

// SaveStateViaPipe 通过管道命令 EmuRunner 序列化存档并上传到 MinIO
// 替代原有的 DataChannel 广播 + state.done 轮询流程
func (m *SessionManager) SaveStateViaPipe(ctx context.Context, roomID, uploadURL string) (int64, error) {
	session, ok := m.Get(roomID)
	if !ok {
		return 0, fmt.Errorf("session not found: %s", roomID)
	}

	resp, err := session.SendCommand(emurunner.Cmd{Cmd: "save_state"})
	if err != nil {
		return 0, fmt.Errorf("save state command: %w", err)
	}
	if resp.Status != "ok" {
		return 0, fmt.Errorf("emurunner save state: %s", resp.Message)
	}

	data, err := os.ReadFile(filepath.Join(session.WorkDir, saveStateFile))
	if err != nil {
		return 0, fmt.Errorf("read state file: %w", err)
	}

	if err := uploadFile(ctx, uploadURL, data); err != nil {
		return 0, fmt.Errorf("upload state: %w", err)
	}

	slog.Info("save state via pipe completed", "room_id", roomID, "size", len(data))
	return int64(len(data)), nil
}

// LoadStateViaPipe 下载存档到 workDir 并通过管道命令 EmuRunner 反序列化
// 替代原有的 DataChannel 广播 + load.done 轮询流程
func (m *SessionManager) LoadStateViaPipe(ctx context.Context, roomID, downloadURL string) error {
	session, ok := m.Get(roomID)
	if !ok {
		return fmt.Errorf("session not found: %s", roomID)
	}

	data, err := downloadBytes(ctx, downloadURL)
	if err != nil {
		return fmt.Errorf("download state: %w", err)
	}
	if err := os.WriteFile(filepath.Join(session.WorkDir, loadStateFile), data, 0644); err != nil {
		return fmt.Errorf("write load file: %w", err)
	}

	resp, err := session.SendCommand(emurunner.Cmd{Cmd: "load_state"})
	if err != nil {
		return fmt.Errorf("load state command: %w", err)
	}
	if resp.Status != "ok" {
		return fmt.Errorf("emurunner load state: %s", resp.Message)
	}

	slog.Info("load state via pipe completed", "room_id", roomID)
	return nil
}

// SwitchRom 下载新 ROM 并通过管道命令 EmuRunner 热切换
// 下载到 shared workDir/rom.dat（覆盖旧 ROM），发送 load_rom 命令，等待 EmuRunner 完成切换
func (m *SessionManager) SwitchRom(roomID, romURL string) error {
	session, ok := m.Get(roomID)
	if !ok {
		return fmt.Errorf("session not found: %s", roomID)
	}

	romPath := filepath.Join(session.WorkDir, "rom.dat")
	if err := downloadFile(romPath, romURL); err != nil {
		return fmt.Errorf("download new rom: %w", err)
	}

	resp, err := session.SendCommand(emurunner.Cmd{Cmd: "load_rom", RomPath: romPath})
	if err != nil {
		return fmt.Errorf("send load_rom command: %w", err)
	}
	if resp.Status != "ok" {
		return fmt.Errorf("emurunner load_rom failed: %s", resp.Message)
	}

	slog.Info("switch rom via pipe completed", "room_id", roomID, "rom_path", romPath)
	return nil
}

// workDirOf 返回房间的共享工作目录
func (m *SessionManager) workDirOf(roomID string) (string, bool) {
	m.mu.RLock()
	session, ok := m.sessions[roomID]
	m.mu.RUnlock()
	if !ok {
		return "", false
	}
	return session.WorkDir, true
}

// PrepareSaveState 清理残留的存档标志文件（在广播 SaveState 指令前调用，避免轮询到旧标志）
func (m *SessionManager) PrepareSaveState(roomID string) (string, error) {
	workDir, ok := m.workDirOf(roomID)
	if !ok {
		return "", fmt.Errorf("session not found: %s", roomID)
	}
	_ = os.Remove(filepath.Join(workDir, saveStateFile))
	_ = os.Remove(filepath.Join(workDir, saveDoneFile))
	return workDir, nil
}

// WaitAndUploadSaveState 轮询 state.done 完成标志，读取 state.dat 并用预签名 PUT URL 上传到 MinIO
// 返回上传的状态字节数
func (m *SessionManager) WaitAndUploadSaveState(ctx context.Context, workDir, uploadURL string) (int64, error) {
	donePath := filepath.Join(workDir, saveDoneFile)
	if err := waitForFile(ctx, donePath); err != nil {
		return 0, fmt.Errorf("wait save done: %w", err)
	}

	data, err := os.ReadFile(filepath.Join(workDir, saveStateFile))
	if err != nil {
		return 0, fmt.Errorf("read state file: %w", err)
	}

	if err := uploadFile(ctx, uploadURL, data); err != nil {
		return 0, fmt.Errorf("upload state: %w", err)
	}

	// 清理标志文件，为下次存档准备
	_ = os.Remove(donePath)
	return int64(len(data)), nil
}

// PrepareLoadState 下载存档二进制到共享目录 load.dat，并清理旧的完成标志
// 在广播 LoadState 指令前调用，确保 EmuRunner 读取到最新数据
func (m *SessionManager) PrepareLoadState(ctx context.Context, roomID, downloadURL string) error {
	workDir, ok := m.workDirOf(roomID)
	if !ok {
		return fmt.Errorf("session not found: %s", roomID)
	}
	_ = os.Remove(filepath.Join(workDir, loadDoneFile))

	data, err := downloadBytes(ctx, downloadURL)
	if err != nil {
		return fmt.Errorf("download state: %w", err)
	}
	if err := os.WriteFile(filepath.Join(workDir, loadStateFile), data, 0644); err != nil {
		return fmt.Errorf("write load file: %w", err)
	}
	return nil
}

// waitForFile 轮询等待文件出现，直到超时或 ctx 取消
func waitForFile(ctx context.Context, path string) error {
	deadline := time.Now().Add(stateWaitTimeout)
	ticker := time.NewTicker(stateWaitInterval)
	defer ticker.Stop()
	for {
		if _, err := os.Stat(path); err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return fmt.Errorf("timeout waiting for %s", path)
			}
		}
	}
}

// uploadFile 用 HTTP PUT 将数据上传到 MinIO 预签名 URL
func uploadFile(ctx context.Context, uploadURL string, data []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.ContentLength = int64(len(data))
	req.Header.Set("Content-Type", "application/octet-stream")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("upload failed with status %d", resp.StatusCode)
	}
	return nil
}

// downloadBytes 从 URL 下载全部内容
func downloadBytes(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed with status %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}
