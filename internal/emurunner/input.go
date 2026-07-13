package emurunner

import (
	"sync"
)

// PortEntry 表示一个端口绑定条目，用于 PORT_MAP 控制包的解析
type PortEntry struct {
	Port     int    // 模拟器手柄端口号（0-based）
	Identity string // LiveKit participant identity（如 "player:{user_id}"）
}

// InputManager 管理玩家输入状态和端口映射
// 线程安全，支持 DataChannel 回调（写入）和 libretro 帧循环（读取）并发访问
type InputManager struct {
	mu              sync.RWMutex
	portToIdentity  map[int]string    // port → LiveKit identity
	identityToState map[string]uint16 // identity → 当前按键状态 bitset
}

// NewInputManager 创建 InputManager 实例
// hostIdentity 是房主的 LiveKit participant identity（如 "player:{hostUserId}"），默认绑定到 Port 0
func NewInputManager(hostIdentity string) *InputManager {
	m := &InputManager{
		portToIdentity:  make(map[int]string),
		identityToState: make(map[string]uint16),
	}
	m.portToIdentity[0] = hostIdentity
	return m
}

// UpdateInput 更新指定 identity 的按键状态
// 用于 DataChannel 收到 input topic 包时调用
func (m *InputManager) UpdateInput(identity string, state uint16) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.identityToState[identity] = state
}

// UpdatePortMapping 整体替换端口映射表
// 用于 DataChannel 收到 control topic 的 PORT_MAP 包时调用
// 当前映射表中不在新列表中的 port 会被清除
func (m *InputManager) UpdatePortMapping(entries []PortEntry) {
	m.mu.Lock()
	defer m.mu.Unlock()

	newMap := make(map[int]string, len(entries))
	for _, e := range entries {
		newMap[e.Port] = e.Identity
	}
	m.portToIdentity = newMap

	// 清除不再映射的 identity 的输入状态
	activeIdentities := make(map[string]bool, len(entries))
	for _, e := range entries {
		activeIdentities[e.Identity] = true
	}
	for id := range m.identityToState {
		if !activeIdentities[id] {
			delete(m.identityToState, id)
		}
	}
}

// GetButton 查询指定端口上某个按钮的按下状态
// port: 模拟器手柄端口号（0-based）
// id: libretro RETRO_DEVICE_ID_JOYPAD_* 常量（0=B, 1=Y, 2=Select, 3=Start, 4=Up, 5=Down, 6=Left, 7=Right, 8=A, 9=X, 10=L, 11=R）
// 返回值：1=按下，0=释放
// 如果端口未绑定任何玩家，返回 0
func (m *InputManager) GetButton(port int, id int) int16 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	identity, ok := m.portToIdentity[port]
	if !ok {
		return 0
	}

	state, ok := m.identityToState[identity]
	if !ok {
		return 0
	}

	return int16((state >> id) & 1)
}
