# 玩家游戏操作流程

本文档描述已完成的玩家输入/操作功能的完整实现，涵盖 DataChannel 协议定义、Identity 体系、Port Mapping 流程、控制权转移、按键映射、以及端到端输入数据流。

---

## 1. 总体架构

```
┌──────────────┐    LiveKit DataChannel     ┌─────────────┐    libretro C API    ┌──────────┐
│  浏览器玩家    │ ◄── 4-Byte 二进制包 ──────► │  EmuRunner   │ ◄── input_state() ─► │ libretro │
│  (Vue 3)     │   topic="input" (lossy)    │  (Go)        │                      │  核心     │
│              │                            │              │                      │          │
│              │   topic="control" (reliable)│              │                      └──────────┘
│              │    PORT_MAP 广播             │              │
│              │                            │              │
│              │   topic="ping" (lossy)       │              │
│              │    RTT 延迟探测（双向）       │              │
│              │                            │              │
│   useGameInput │◄── publishInput() ────►  │ InputManager │
│   useLiveKit │                            │ LiveKitPublisher│
└──────────────┘                            └──────┬───────┘
                                                   │ gRPC
                                          ┌────────┴───────┐
                                          │  Worker Agent   │
                                          │  (Go)          │
                                          │  LiveKitManager │
                                          └────────┬───────┘
                                                   │ gRPC
                                          ┌────────┴───────┐
                                          │  Control Plane  │
                                          │  (Go)          │
                                          │  RoomService    │
                                          │  WorkerClient   │
                                          └────────────────┘
```

---

## 2. LiveKit DataChannel 协议定义

### 2.1 Topic 分类

所有玩家和 EmuRunner 通过 LiveKit 的 DataChannel 通信。消息按 `topic` 分为两类：

| Topic | 通道类型 | 用途 | 发送方 | 接收方 |
|-------|---------|------|--------|--------|
| `"input"` | **lossy** (unreliable) | 玩家按键输入 | 浏览器（所有玩家） | EmuRunner |
| `"control"` | **reliable** | 服务端控制消息（Port Mapping） | Worker（服务端） | EmuRunner、浏览器 |
| `"ping"` | **lossy** (unreliable) | 客户端→EmuRunner 延迟探测 + EmuRunner→Client 回复 | 浏览器 & EmuRunner | EmuRunner & 浏览器（双向） |

**可靠性选择原因**：
- `input` 使用 lossy：60Hz 高频发送，丢帧影响极小，最新帧覆盖旧帧，低延迟优先
- `control` 使用 reliable：端口映射是低频关键状态变更，不能丢失

### 2.2 二进制包格式

#### 2.2.1 玩家输入包 (`topic="input"`)

```
 Offset │ Size │ Field
────────┼──────┼─────────────────────────────
  0     │  1   │ type = 0x01 （输入包标识）
  1     │  1   │ buttons_lo （uint16 低字节）
  2     │  1   │ buttons_hi （uint16 高字节）
  3     │  1   │ reserved = 0x00 （保留）
```

**buttons 位定义**（uint16 little-endian，与 libretro `RETRO_DEVICE_ID_JOYPAD_*` 对齐）：

| Bit | 按键名 | libretro 常量 |
|-----|--------|---------------|
| 0   | B      | RETRO_DEVICE_ID_JOYPAD_B |
| 1   | Y      | RETRO_DEVICE_ID_JOYPAD_Y |
| 2   | Select | RETRO_DEVICE_ID_JOYPAD_SELECT |
| 3   | Start  | RETRO_DEVICE_ID_JOYPAD_START |
| 4   | Up     | RETRO_DEVICE_ID_JOYPAD_UP |
| 5   | Down   | RETRO_DEVICE_ID_JOYPAD_DOWN |
| 6   | Left   | RETRO_DEVICE_ID_JOYPAD_LEFT |
| 7   | Right  | RETRO_DEVICE_ID_JOYPAD_RIGHT |
| 8   | A      | RETRO_DEVICE_ID_JOYPAD_A |
| 9   | X      | RETRO_DEVICE_ID_JOYPAD_X |
| 10  | L      | RETRO_DEVICE_ID_JOYPAD_L |
| 11  | R      | RETRO_DEVICE_ID_JOYPAD_R |
| 12  | TurboA | 自定义连发 A（暂未在 EmuRunner 使用） |
| 13  | TurboB | 自定义连发 B（暂未在 EmuRunner 使用） |

#### 2.2.2 端口映射包 (`topic="control"`)

```
 Offset │ Size │ Field
────────┼──────┼─────────────────────────────
  0     │  1   │ type = 0x02 （控制包标识）
  1     │  1   │ count = N （映射条目数）
  2     │ 变长  │ count 个 entry，每个格式：
        │      │   [port:1B][identity_len:1B][identity:identity_len字节]
```

**示例**（假设有 2 个玩家，Port 0 绑定 `player:user_a`，Port 1 绑定 `player:user_b`）：

```
02 02  00 0E 70 6C 61 79 65 72 3A 75 73 65 72 5F 61  01 0E 70 6C 61 79 65 72 3A 75 73 65 72 5F 62
│  │   │  │  p  l  a  y  e  r  :  u  s  e  r  _  a   │  │  p  l  a  y  e  r  :  u  s  e  r  _  b
│  │   └─── entry 0 ──────────────────────────────── └─── entry 1 ───────────────────────────────
│  └── count=2（共 2 个映射条目）
└── type=0x02（端口映射包）
```

#### 2.2.3 延迟探测包 (`topic="ping"`)

Ping（Client → EmuRunner）：

```
 Offset │ Size │ Field
────────┼──────┼─────────────────────────────
  0     │  1   │ type = 0x03
  1     │  8   │ client_ts （int64 LE, ms，来自 performance.now()）
```

Pong（EmuRunner → Client，定向回复给发起者）：

```
 Offset │ Size │ Field
────────┼──────┼─────────────────────────────
  0     │  1   │ type = 0x04
  1     │  8   │ client_ts （原路回传）
  9     │  8   │ server_ts （time.Now().UnixMilli()）
```

**延迟计算**（全部在客户端完成）：

```
RTT = performance.now() - client_ts   // 同源时钟，准确
server_ts 仅作为参考日志，不与 client_ts 混算
```

**发送策略**：客户端每 3 秒发一次 ping，lossy 通道（与输入通道一致），连接成功后自动开始，断开/暂停时停止。

**常量和 Topic 定义**：

```go
const (
    packetTypePing    byte = 0x03
    packetTypePong    byte = 0x04
)
const (
    topicPing    = "ping"
)
```

**解析常量定义**（`internal/emurunner/publish.go:20-32`）：

```go
const (
    packetTypeInput   byte = 0x01 // 玩家手柄输入
    packetTypePortMap byte = 0x02 // 端口映射更新
    packetTypePing    byte = 0x03 // 延迟探测请求
    packetTypePong    byte = 0x04 // 延迟探测回复
)

const (
    topicInput   = "input"   // 玩家输入
    topicControl = "control" // 服务端控制消息
    topicPing    = "ping"    // 延迟探测
)
```

### 2.3 发送端实现

**前端（浏览器）**：`web/src/composables/useGameInput.ts:86-91`

```ts
const packet = new Uint8Array(4)
packet[0] = 0x01           // type
packet[1] = state & 0xff    // buttons_lo
packet[2] = (state >> 8) & 0xff  // buttons_hi
packet[3] = 0               // reserved
publishInput(packet, 'input')
```

**前端发布函数**：`web/src/composables/useLiveKit.ts:59-63`

```ts
function publishInput(data: Uint8Array, topic?: string) {
    const opts: { reliable: boolean; topic?: string } = { reliable: false }
    if (topic) opts.topic = topic
    room.value?.localParticipant.publishData(data, opts)
}
```

**Worker 广播 PORT_MAP**：`internal/worker/grpc/server.go:148-178`

```go
func (s *WorkerServer) UpdatePortMapping(ctx context.Context, req *workerpb.UpdatePortMappingRequest) (*workerpb.UpdatePortMappingResponse, error) {
    // 编码 PORT_MAP: [type=0x02][count:1B][entries...]
    data := make([]byte, totalLen)
    data[0] = 0x02
    data[1] = byte(len(mapping))
    // ...逐 entry 写入 port + identity
    s.livekit.SendDataBroadcast(ctx, roomID, "control", true, data)
}
```

**LiveKit SDK 服务端广播**：`internal/worker/livekit.go:93-109`

```go
func (m *LiveKitManager) SendDataBroadcast(ctx context.Context, roomName, topic string, reliable bool, data []byte) error {
    kind := livekit.DataPacket_RELIABLE  // reliable = true 时
    if !reliable {
        kind = livekit.DataPacket_LOSSY // reliable = false 时
    }
    m.client.SendData(ctx, &livekit.SendDataRequest{
        Room:  roomName,
        Data:  data,
        Kind:  kind,
        Topic: &topic,
    })
}
```

### 2.4 接收端路由

`internal/emurunner/publish.go:61-78` — `ConnectRoom()` 中注册的 `OnDataPacket` 回调：

```go
cb.OnDataPacket = func(data lksdk.DataPacket, params lksdk.DataReceiveParams) {
    userData, ok := data.(*lksdk.UserDataPacket)
    if !ok { return }
    topic := userData.Topic
    if topic == "" {
        topic = params.Topic
    }
    switch topic {
    case "input":
        l.handleInputPacket(params.SenderIdentity, userData.Payload)
    case "control":
        l.handleControlPacket(params.SenderIdentity, userData.Payload)
    }
}
```

---

## 3. Identity 体系

### 3.1 LiveKit Participant Identity

LiveKit 房间中每个参与者有一个唯一 `identity` 字符串，输入数据按此 identity 路由到对应端口。

| 角色 | Identity 格式 | canPublish | 说明 |
|------|--------------|------------|------|
| EmuRunner | `"emurunner"` | `true` | 模拟器进程本身，发布视频/音频 track |
| 玩家 | `"player:{user_id}"` | `false` | 每个玩家独立身份，订阅视频，发送 DataChannel 输入 |

**Token 权限**：`internal/worker/livekit.go:53-77`

```go
func (m *LiveKitManager) GenerateToken(roomName, identity string, canPublish bool) (string, error) {
    at := auth.NewAccessToken(m.apiKey, m.apiSecret)
    at.SetIdentity(identity)
    grant := &auth.VideoGrant{RoomJoin: true, Room: roomName}
    if canPublish {
        grant.SetCanPublish(true)       // EmuRunner 可发布 track
        grant.SetCanPublishData(true)   // EmuRunner 可发 DataChannel（广播 port_map 等场景）
    }
    grant.SetCanSubscribe(true)         // 所有参与者可订阅（接收视频）
    at.SetVideoGrant(grant)
    return at.ToJWT()
}
```

### 3.2 Token 生成时机

| 场景 | Who | 生成方式 |
|------|-----|---------|
| 房主启动游戏 | 房主 | `RoomService.Start()` → Worker `StartGame()` 返回 `HostToken`（identity=`player:{host_id}`） |
| 非房主进入游戏 | 非房主玩家 | 前端 `GET /api/rooms/:id/livekit` → Control Plane → Worker `GeneratePlayerToken()`（identity=`player:{user_id}`） |
| EmuRunner 连接 | EmuRunner 进程 | Worker `StartGame()` 内部生成 EmuRunner 专属 token（identity=`emurunner`），通过 `--token` 传参 |

### 3.3 非房主玩家获取 Token 流程

`internal/control-plane/service/room_service.go:382-395`

```go
func (s *RoomService) GetLivekitToken(ctx context.Context, userID uuid.UUID, roomID uuid.UUID) (*contract.LivekitTokenResp, error) {
    room, err := s.roomRepo.ByID(ctx, roomID)
    // 游戏未开始（status != 1）→ 返回 Waiting=true，前端继续轮询
    if room.Status != 1 {
        return &contract.LivekitTokenResp{Waiting: true}, nil
    }
    // 游戏进行中 → 调用 Worker GeneratePlayerToken gRPC
    token, err := s.workerClient.GeneratePlayerToken(ctx, room.WorkerAddr, roomID, userID)
    // 返回 token 给前端
}
```

前端轮询逻辑：`web/src/views/PlayView.vue` — 非房主进入 `/play/:roomId` 后，每 2 秒调用 `GET /api/rooms/:id/livekit` 直到拿到 token。

---

## 4. InputManager 状态机

`internal/emurunner/input.go` — EmuRunner 内的核心输入管理组件。

### 4.1 数据结构

```go
type InputManager struct {
    mu              sync.RWMutex
    portToIdentity  map[int]string    // port → LiveKit identity
    identityToState map[string]uint16 // identity → 当前按键 bitset
}
```

双 map 设计确保：
- **写入路径**（DataChannel 回调）：按 `identity` 更新 `identityToState`，只需一次 map 查找
- **读取路径**（libretro 帧循环）： `port → identity → state`，两次 map 查找
- 同一玩家可被重新映射到不同 port（控制权转移），状态保留

### 4.2 初始化

```go
func NewInputManager(hostIdentity string) *InputManager {
    m := &InputManager{...}
    m.portToIdentity[0] = hostIdentity  // 房主默认绑定 Port 0
    return m
}
```

### 4.3 UpdateInput — 玩家输入写入

DataChannel 收到 `topic="input"` 包时调用：

```go
func (m *InputManager) UpdateInput(identity string, state uint16) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.identityToState[identity] = state  // 直接覆盖，不需要 port 查找
}
```

### 4.4 UpdatePortMapping — 端口映射整体替换

DataChannel 收到 `topic="control"` 的 PORT_MAP 包时调用：

```go
func (m *InputManager) UpdatePortMapping(entries []PortEntry) {
    m.mu.Lock()
    defer m.mu.Unlock()
    // 1. 整体替换 portToIdentity
    newMap := make(map[int]string, len(entries))
    for _, e := range entries {
        newMap[e.Port] = e.Identity
    }
    m.portToIdentity = newMap
    // 2. 清理不再映射的 identity 的输入状态（避免状态泄漏）
    activeIdentities := make(map[string]bool)
    for _, e := range entries {
        activeIdentities[e.Identity] = true
    }
    for id := range m.identityToState {
        if !activeIdentities[id] {
            delete(m.identityToState, id)
        }
    }
}
```

### 4.5 GetButton — libretro 读取

每帧被 libretro `input_state` 回调调用（每个 port × 12 个按键）：

```go
func (m *InputManager) GetButton(port int, id int) int16 {
    m.mu.RLock()
    defer m.mu.RUnlock()
    identity, ok := m.portToIdentity[port]
    if !ok { return 0 }              // port 未绑定 → 无输入
    state, ok := m.identityToState[identity]
    if !ok { return 0 }              // 该玩家尚未发送过输入
    return int16((state >> id) & 1)  // 返回 1（按下）或 0（释放）
}
```

---

## 5. Port Mapping 完整流程

### 5.1 初始映射（游戏启动时）

```
Worker.StartGame()
  │
  ├─ 创建 LiveKit 房间 → 生成 EmuRunner token（identity="emurunner"）
  ├─ 生成房主 token（identity="player:{host_id}"）
  └─ SessionManager.Start() → 启动 EmuRunner 子进程
       │ 命令行参数：--host-identity=player:{host_id}
       ▼
     EmuRunner main()
       │ NewInstance(hostIdentity) → NewInputManager(hostIdentity)
       ▼
     InputManager.portToIdentity[0] = "player:{host_id}"
     ↓ 房主默认绑定 Port 0，其余 port 空
```

### 5.2 控制权转移（AssignPort）完整流程

房主在游戏中通过 UI 让其他玩家获得控制权（从 spectator 升级到 player，绑定指定 port）：

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│ 1. 前端触发                                                                      │
│    MemberPanel → 房主点击成员角色下拉 → 选择 "Player" → handleRoleChange()        │
│                                                                                 │
│ 2. Pinia Action                                                                 │
│    roomStore.assignPort({ room_id, user_id, port })                              │
│    → POST /api/rooms/assign                                                      │
│                                                                                 │
│ 3. Control Plane Handler                                                        │
│    RoomHandler.AssignPort(c) → parseUUIDParam + binding 校验                     │
│                                                                                 │
│ 4. Control Plane Service                                                        │
│    RoomService.AssignPort(ctx, hostID, req):                                     │
│      ├─ 校验：hostID 是房主 + room.Status==1 (playing) + port 合法               │
│      ├─ 如端口已被 old player 占用：old → spectator (role=2, port=NULL)         │
│      ├─ target player → role=1, port=req.Port                                   │
│      ├─ Redis room:{id}:ports hash: SET port → user_id                         │
│      ├─ 查询所有 role=1 且 port!=NULL 的玩家 → 构建全量 mapping:                 │
│      │   map[port] = "player:{user_id}"                                         │
│      └─ WorkerClient.UpdatePortMapping(workerAddr, roomID, mapping)              │
│           │                                                                      │
│           │ gRPC call                                                            │
│           ▼                                                                      │
│ 5. Worker gRPC Server                                                           │
│    WorkerServer.UpdatePortMapping():                                             │
│      ├─ 编码 PORT_MAP 二进制包:                                                  │
│      │   [0x02][count][port][idLen][identity]...                                 │
│      └─ LiveKitManager.SendDataBroadcast(roomID, "control", reliable=true, data) │
│           │                                                                      │
│           │ LiveKit Server SDK                                                   │
│           ▼                                                                      │
│ 6. LiveKit SFU → 广播到房间所有参与者                                              │
│                                                                                 │
│ 7. EmuRunner 接收                                                                │
│    LiveKitPublisher.OnDataPacket → topic="control"                              │
│      → handleControlPacket() → 解析 PORT_MAP entries                            │
│      → InputManager.UpdatePortMapping(entries)                                   │
│         ├─ 整体替换 portToIdentity                                                │
│         └─ 清理不再映射的 identity 的输入状态                                      │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### 5.3 关键边界条件

1. **端口抢占**：如果 Port 1 已被玩家 A 占有，房主分配 Port 1 给玩家 B 时，A 自动降级为 spectator（`port=NULL`），其 identity 从映射中移除
2. **全量 Mapping**：PORT_MAP 包总是包含当前所有 `role=1` 玩家的**全量映射**（非增量），因为 EmuRunner 做整体替换，确保与 Control Plane 最终一致
3. **Redis 缓存**：`room:{id}:ports` hash 记录每个 port→user_id，与 DB room_players 互为镜像
4. **PostgreSQL**：`room_players` 表的 `role` 和 `port` 字段实时更新（`UpdateRoleAndPort`）

---

## 6. 按键映射系统

### 6.1 架构

```
keyMapping.ts
  ├── BUTTON_BITS    按钮名 → uint16 bit 位置（0~13）
  ├── DEFAULT_KEY_MAPPING  按钮名 → KeyboardEvent.code（默认键盘映射）
  ├── loadMapping()  localStorage → 用户自定义映射（合并默认值）
  ├── saveMapping()  映射 → localStorage
  └── buildReverseMap()  code → 按钮名（反向索引，O(1) 查找）
```

### 6.2 默认按键映射

`web/src/utils/keyMapping.ts:44-60`

| 手柄按钮 | 键盘按键 (code) | 说明 |
|---------|----------------|------|
| A       | `KeyZ`         | NES/GB 确认键 |
| B       | `KeyX`         | NES/GB 取消键 |
| Start   | `Enter`        | 开始/暂停 |
| Select  | `ShiftLeft`    | 选择 |
| Up      | `ArrowUp`      | 方向：上 |
| Down    | `ArrowDown`    | 方向：下 |
| Left    | `ArrowLeft`    | 方向：左 |
| Right   | `ArrowRight`   | 方向：右 |
| L       | `KeyQ`         | 左扳机 |
| R       | `KeyE`         | 右扳机 |
| Y       | *空*           | 未默认绑定（多数复古游戏只用 A/B） |
| X       | *空*           | 未默认绑定 |
| TurboA  | *空*           | 连发 A（EmuRunner 侧暂未使用） |
| TurboB  | *空*           | 连发 B（EmuRunner 侧暂未使用） |

### 6.3 按键映射 UI

`web/src/components/play/KeyMappingDialog.vue`

- 点击游戏手柄图示上的按钮 → 进入"监听"模式
- 按下键盘按键 → `KeyboardEvent.code` 写入映射（物理键位，跨键盘布局稳定）
- 保存到 `localStorage`，持久化生效
- 支持"恢复默认"
- 与 `useGameInput` 通过 `reloadMapping()` 联动

### 6.4 实时输入采集

`web/src/composables/useGameInput.ts`

```
keydown ──→ reverseMap[e.code] → ButtonName → BUTTON_BITS → state |= 1<<bit
keyup   ──→ reverseMap[e.code] → ButtonName → BUTTON_BITS → state &= ~(1<<bit)
blur    ──→ state = 0  （失焦释放所有按键，防止卡键）
e.repeat → ignore （忽略按键重复）

RAF loop (60Hz) → 构建 4-byte packet → publishInput(packet, 'input')
```

**自动启停**：当 `enabled` ref 为 `false` 时（LiveKit 未连接或模拟器未运行），自动停止监听键盘和 RAF 循环。

---

## 7. 端到端输入数据流

```
┌──────────────────────────────────────────────────────────────────────────────────┐
│                             完整输入帧数据流                                       │
└──────────────────────────────────────────────────────────────────────────────────┘

[浏览器] 玩家按下键盘
  │ keydown → onKeyDown(e)
  │ reverseMap[e.code] = ButtonName (如 "ArrowUp")
  │ BUTTON_BITS["Up"] = 4
  │ state |= (1 << 4)  →  state = 0b00000000_00010000
  ▼
[useGameInput.ts] RAF loop @ 60Hz
  │ while (enabled.value) {
  │   if (30ms elapsed) {
  │     packet = Uint8Array[0x01, 0x10, 0x00, 0x00]
  │     publishInput(packet, "input")
  │   }
  │   requestAnimationFrame(loop)
  │ }
  ▼
[LiveKit JS SDK] room.localParticipant.publishData(data, { reliable: false, topic: "input" })
  │ WebRTC DataChannel (lossy, low-latency)
  │ identity = "player:{user_id}" （浏览器加入 LiveKit 时的身份）
  ▼
[LiveKit SFU] 路由到房间内所有参与者
  │ DataPacket → EmuRunner (identity="emurunner")
  ▼
[EmuRunner LiveKitPublisher] OnDataPacket callback
  │ rawData.(*lksdk.UserDataPacket) → userData.Topic = "input"
  │ switch topic:
  │   case "input":
  │     handleInputPacket(params.SenderIdentity, userData.Payload)
  │     │ senderIdentity = "player:{user_id}"
  │     │ payload = [0x01, 0x10, 0x00, 0x00]
  │     ▼
  │     ┌─ 验证: len(payload) >= 3 && payload[0] == 0x01
  │     ├─ state = uint16(payload[1]) | (uint16(payload[2]) << 8) = 0x0010
  │     └─ InputManager.UpdateInput("player:{user_id}", 0x0010)
  │          identityToState["player:{user_id}"] = 0x0010
  ▼
[InputManager] GetButton(port=0, id=4) ← libretro 帧循环查询
  │ m.mu.RLock()
  │ identity = portToIdentity[0] = "player:{user_id}"  ← 当前 Port 0 的玩家
  │ state = identityToState["player:{user_id}"] = 0x0010
  │ return (0x0010 >> 4) & 1 = 1  → 按下了
  │ m.mu.RUnlock()
  ▼
[goInputStateCB] (backend/backend.go)
  │ return InputProvider.GetButton(port, 4) = 1
  ▼
[libretro 核心]
  │ retro_input_state(port=0, device=1, index=0, id=RETRO_DEVICE_ID_JOYPAD_UP) → 1
  │ 核心处理 UP 方向键按下 → 游戏角色向上移动
  ▼
[libretro] retro_run() 渲染新帧
  │ video_refresh(data, width, height, pitch)
  ▼
[X264Encoder] YCbCr → H.264 编码
  ▼
[LiveKitPublisher] 发布 video track → WebRTC → 浏览器 <video> 播放
```

---

## 8. 多人手柄映射数据流

```
游戏已运行中 (status=1)
┌─────────────────────────────────────────────────────────────────┐
│ 房主操作：把旁观玩家升级为 Player，绑定到 Port 1                    │
└─────────────────────────────────────────────────────────────────┘

POST /api/rooms/assign { room_id, user_id, port: 1 }
  │
  ▼
RoomService.AssignPort()
  │ 校验 + DB UpdateRoleAndPort
  │ Redis room:{id}:ports → SET port 1 = user_id
  │
  │ 构建 mapping: { 0: "player:{host_id}", 1: "player:{user_id}" }
  │
  │ gRPC → WorkerClient.UpdatePortMapping()
  ▼
Worker gRPC Server → WorkerServer.UpdatePortMapping()
  │ 编码 PORT_MAP 二进制包 → SendDataBroadcast("control", reliable)
  ▼
LiveKit SFU → 广播到房间
  │
  ▼
EmuRunner: handleControlPacket()
  │ 解析: [0x02][02][00][0E]"player:host" [01][0E]"player:user"
  ▼
InputManager.UpdatePortMapping([
  {Port: 0, Identity: "player:{host_id}"},
  {Port: 1, Identity: "player:{user_id}"}
])
  │ portToIdentity = { 0→host, 1→user }
  ▼
下次 GetButton(port=1, id=X):
  identity = "player:{user_id}"  → state 转为目标玩家按键 → 控制权生效
```

---

## 9. 延迟探测数据流

```
客户端每 3s 自动发送 ping，连接成功后启动，断开/暂停时停止

[浏览器] sendPing() @ 3s interval
  │ ts = performance.now()
  │ packet = [0x03][ts:8B LE]
  │ publishData(packet, 'ping')  →  lossy, topic="ping"
  ▼
[LiveKit SFU] 路由到 EmuRunner (identity="emurunner")
  ▼
[EmuRunner] OnDataPacket: topic="ping", senderIdentity="player:{id}"
  │ handlePingPacket(senderIdentity, payload)
  │   ├─ 解析 clientTs = payload[1:9]
  │   ├─ serverTs = time.Now().UnixMilli()
  │   ├─ 构建 pong: [0x04][clientTs][serverTs]
  │   └─ LocalParticipant.PublishData(pong, lossy, destination=["player:{id}"], topic="ping")
  │
  │    定向回复给发起者（非广播），避免噪声
  ▼
[浏览器] RoomEvent.DataReceived: topic="ping", payload=[0x04][clientTs][serverTs]
  │ handlePong(payload, topic)
  │   ├─ clientTs = Number(view.getBigInt64(1, true))
  │   └─ latencyMs = Math.round(performance.now() - clientTs)  // RTT
  ▼
[HUD] GameScreen: "延迟: 32ms"  /  GameToolbar: "32ms"
```

---

## 10. Player Token 管理完整流程

```
┌─ 房主启动游戏 ─────────────────────────────────────────────────────┐
│                                                                     │
│  RoomView.vue: handleStartGame()                                    │
│    → POST /api/rooms/:id/start  (no body)                           │
│                                                                     │
│  RoomService.Start()                                                │
│    → Scheduler.SelectWorker() → 选 Worker                            │
│    → WorkerClient.StartGame(gRPC)                                   │
│       → Worker: CreateRoom + GenerateToken("emurunner")             │
│        + GenerateToken("player:{host_id}") → hostToken              │
│        + SessionManager.Start() → 启动 EmuRunner 子进程               │
│                                                                     │
│    返回: { livekit_token: hostToken, livekit_room, livekit_url }    │
│                                                                     │
│  前端: livekit.connect(url, hostToken)  → 房东接入 LiveKit           │
└─────────────────────────────────────────────────────────────────────┘

┌─ 非房主加入游戏 ────────────────────────────────────────────────────┐
│                                                                     │
│  PlayView.vue: pollForToken()                                       │
│    → GET /api/rooms/:id/livekit  (每 2 秒轮询)                       │
│                                                                     │
│  RoomService.GetLivekitToken()                                      │
│    if room.Status != 1 → { waiting: true }  (继续轮询)               │
│    if room.Status == 1 → WorkerClient.GeneratePlayerToken(gRPC)     │
│       → Worker: GenerateToken("player:{user_id}") → playerToken     │
│                                                                     │
│  前端: livekit.connect(url, playerToken)  → 玩家接入 LiveKit         │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 11. 关键文件索引

| 文件 | 职责 |
|------|------|
| `web/src/utils/keyMapping.ts` | 按键名定义、默认映射、localStorage 持久化 |
| `web/src/components/play/KeyMappingDialog.vue` | 按键映射 UI 弹窗 |
| `web/src/composables/useGameInput.ts` | 前端输入采集：keydown/keyup → RAF 60Hz 发送 |
| `web/src/composables/useLiveKit.ts` | LiveKit 连接管理 + `publishInput()` + DataReceived 接收 |
| `web/src/composables/useLatencyMeasurer.ts` | 延迟探测：定时 ping → 接收 pong → 计算 RTT |
| `web/src/views/PlayView.vue` | 游戏页面：组配 useLiveKit + useGameInput + useLatencyMeasurer |
| `web/src/views/RoomView.vue` | 房间页面：房主启动游戏入口 |
| `web/src/stores/room.ts` | Pinia store：assignPort/startGame/getLivekitToken |
| `web/src/api/room.ts` | Axios API 封装 |
| `internal/emurunner/input.go` | InputManager：port↔identity↔state 状态机 |
| `internal/emurunner/publish.go` | LiveKitPublisher：DataChannel 路由 + input/control 解析 |
| `internal/emurunner/emurunner.go` | EmuRunner 编排：组装 InputManager + Publisher + Runner |
| `internal/emurunner/backend/backend.go` | libretro 回调：goInputStateCB 查询 InputManager |
| `internal/worker/grpc/server.go` | Worker gRPC：StartGame/GeneratePlayerToken/UpdatePortMapping |
| `internal/worker/livekit.go` | LiveKitManager：CreateRoom/GenerateToken/SendDataBroadcast |
| `internal/worker/process.go` | SessionManager：EmuRunner 子进程启动/停止 |
| `internal/control-plane/service/room_service.go` | RoomService：AssignPort/Start/GetLivekitToken |
| `internal/control-plane/grpc/worker_client.go` | Control Plane → Worker gRPC 客户端 |
| `proto/worker.proto` | Protobuf 定义 |
