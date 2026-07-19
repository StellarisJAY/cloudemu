// protocol.go — Worker ↔ EmuRunner 管道通信协议定义
// Worker 通过子进程 stdin 发送 Cmd，EmuRunner 通过 stdout 回复 Resp
// 格式: JSONL（每行一个完整 JSON，以 \n 结尾）

package emurunner

// Cmd Worker 发送给 EmuRunner 的命令
// 通过子进程 stdin 传输，JSON 行协议
type Cmd struct {
	Cmd     string         `json:"cmd"`              // 命令名: pause / resume / port_map / save_state / load_state / load_rom
	Mapping map[int]string `json:"mapping,omitempty"` // port_map 时的端口映射: port → LiveKit identity
	RomPath string         `json:"rom_path,omitempty"` // load_rom 时的新 ROM 文件路径
}

// Resp EmuRunner 回复给 Worker 的响应
// 通过 stdout 传输，JSON 行协议
type Resp struct {
	Cmd     string `json:"cmd"`               // 回显命令名，用于命令-响应匹配
	Status  string `json:"status"`            // "ok" 表示成功，"error" 表示失败
	Message string `json:"message,omitempty"` // 失败时的错误描述
	Size    int64  `json:"size,omitempty"`    // save_state 时返回序列化字节数
}
