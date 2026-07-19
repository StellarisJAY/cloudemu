package emurunner

import (
	"bufio"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
)

// HandleCommand 解析并执行来自 Worker 的管道命令
// 每条命令执行后通过 stdout 回复响应（JSON 行协议）
func (instance *Instance) HandleCommand(raw []byte) {
	var cmd Cmd
	if err := json.Unmarshal(raw, &cmd); err != nil {
		writeResp(Resp{Status: "error", Message: "invalid json: " + err.Error()})
		return
	}

	switch cmd.Cmd {
	case "pause":
		instance.runner.Pause(context.TODO())
		writeResp(Resp{Cmd: "pause", Status: "ok"})

	case "resume":
		instance.runner.Resume(context.TODO())
		writeResp(Resp{Cmd: "resume", Status: "ok"})

	case "port_map":
		entries := make([]PortEntry, 0, len(cmd.Mapping))
		for port, identity := range cmd.Mapping {
			entries = append(entries, PortEntry{Port: port, Identity: identity})
		}
		instance.inputMgr.UpdatePortMapping(entries)
		slog.Info("port mapping updated via pipe", "entries", len(entries))
		writeResp(Resp{Cmd: "port_map", Status: "ok"})

	case "save_state":
		if err := instance.runner.SaveState(instance.workDir); err != nil {
			slog.Error("save state via pipe failed", "error", err)
			writeResp(Resp{Cmd: "save_state", Status: "error", Message: err.Error()})
			return
		}
		statePath := filepath.Join(instance.workDir, saveStateFile)
		fi, err := os.Stat(statePath)
		size := int64(0)
		if err == nil {
			size = fi.Size()
		}
		slog.Info("save state via pipe completed", "size", size)
		writeResp(Resp{Cmd: "save_state", Status: "ok", Size: size})

	case "load_state":
		if err := instance.runner.LoadState(instance.workDir); err != nil {
			slog.Error("load state via pipe failed", "error", err)
			writeResp(Resp{Cmd: "load_state", Status: "error", Message: err.Error()})
			return
		}
		slog.Info("load state via pipe completed")
		writeResp(Resp{Cmd: "load_state", Status: "ok"})

	case "load_rom":
		romPath := cmd.RomPath
		if romPath == "" {
			writeResp(Resp{Cmd: "load_rom", Status: "error", Message: "missing rom_path"})
			return
		}
		slog.Info("reloading rom via pipe", "path", romPath)
		if err := instance.ReloadROM(romPath); err != nil {
			slog.Error("reload rom via pipe failed", "error", err)
			writeResp(Resp{Cmd: "load_rom", Status: "error", Message: err.Error()})
			return
		}
		slog.Info("reload rom via pipe completed", "path", romPath)
		writeResp(Resp{Cmd: "load_rom", Status: "ok"})

	default:
		slog.Warn("unknown pipe command", "cmd", cmd.Cmd)
		writeResp(Resp{Cmd: cmd.Cmd, Status: "error", Message: "unknown command: " + cmd.Cmd})
	}
}

// StartCommandReader 启动后台协程，从 stdin 读取 JSON 行命令并交给 HandleCommand 处理
// 应在 Run() 之前以 goroutine 方式调用
func (instance *Instance) StartCommandReader() {
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			instance.HandleCommand(scanner.Bytes())
		}
		if err := scanner.Err(); err != nil {
			slog.Warn("stdin command reader stopped", "error", err)
		} else {
			slog.Info("stdin command reader closed (pipe disconnected)")
		}
	}()
}

// writeResp 将响应序列化为 JSON 并写入 stdout，每条以 \n 结尾
func writeResp(resp Resp) {
	data, err := json.Marshal(resp)
	if err != nil {
		slog.Error("marshal pipe response failed", "error", err)
		return
	}
	if _, err := os.Stdout.Write(append(data, '\n')); err != nil {
		slog.Error("write pipe response failed", "error", err)
	}
}
