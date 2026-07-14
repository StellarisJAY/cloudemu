# CloudEmu 云端模拟器 — 架构设计文档

## 1. 项目概述

在服务端模拟 NES、GB 等游戏机硬件，通过 WebRTC 实时流传输到浏览器展示（云游戏模式）。支持多人远程同玩（多手柄映射）、分布式扩展、低延迟低带宽。优先实现 2D 复古游戏机。

### 核心目标
- 多机种模拟（NES → GB → 逐步扩展）
- 浏览器端零安装，WebRTC 拉流
- 多人游戏：多手柄独立输入，共享游戏画面
- 分布式架构，Worker 节点弹性伸缩

---

## 2. 总体架构

```
┌──────────────────────────────────────────────────────┐
│                    控制面 (Control Plane)              │
│                                                      │
│   User Service  │  ROM Service  │    Room Manager    │
│                  │               │   + Scheduler     │
│   共享: PostgreSQL + Redis + MinIO                    │
│   对外: REST API (Nginx 反向代理)                      │
│   对内: gRPC client → 连接所有 Worker                  │
└──────────────────────┬───────────────────────────────┘
                       │ gRPC (StartGame / StopGame)
           ┌───────────┼────────────┐
           ▼           ▼            ▼
    ┌──────────┐ ┌──────────┐ ┌──────────┐
    │ Worker 1 │ │ Worker 2 │ │ Worker N │  (独立扩缩)
    │          │ │          │ │          │
    │ 创建LiveKit│ │ 创建LiveKit│ │ 创建LiveKit│
    │ 房间+Token│ │ 房间+Token│ │ 房间+Token│
    │ │ spawns  │ │ │ spawns  │ │ │ spawns  │
    │ ▼         │ │ ▼         │ │ ▼         │
    │ EmuRunner │ │ EmuRunner │ │ EmuRunner │  (子进程)
    │ (模拟+编码)│ │ (模拟+编码)│ │ (模拟+编码)│
    └─────┬─────┘ └─────┬─────┘ └─────┬─────┘
          │             │             │
          └─────────────┼─────────────┘
                        │ WebRTC (EmuRunner 推流)
                ┌───────▼───────┐
                │  LiveKit SFU  │  (现成组件)
                └───────┬───────┘
                        │ WebRTC (浏览器拉流 + DataChannel输入)
                ┌───────▼───────┐
                │    Browser    │
                │  (Vue 3 +     │
                │   LiveKit JS) │
                └───────────────┘
```

### 服务拆分原则

| 组件 | 合并 | 原因 |
|------|------|------|
| User Service | ✅ 单体控制面 | 极低频，简单 CRUD |
| ROM Service | ✅ 单体控制面 | 偶尔上传，低频 |
| Room Manager + Scheduler | ✅ 单体控制面 | 房间管理+调度逻辑，轻量级 |
| Worker Agent | ❌ 独立集群 | 随游戏会话数弹性伸缩，管理 LiveKit 房间创建和 EmuRunner 子进程 |
| EmuRunner | ❌ Worker 子进程 | 每个游戏会话一个进程，运行模拟器+视频编码+WebRTC 推流 |
| LiveKit SFU | ❌ 现成组件 | 开箱即用，不需要自研 |

### 请求量级估算

| 操作 | 频率 | 说明 |
|------|------|------|
| 用户注册/登录 | ~1 次/会话 | 极低频 |
| ROM 上传 | 偶尔 | 每位用户上传几次 |
| 创建/加入/离开房间 | ~1 次/游戏 | 低频 |
| 调度 Worker | ~1 次/房间 | 低频 |
| 定时存档写入 Redis | ~1 次/分钟 | 中等，仅 Redis SET |

单体控制面足以承载数千并发房间的控制逻辑。

---

## 3. 技术选型

| 层 | 技术 | 理由 |
|----|------|------|
| 后端语言 | **Go** | 编译单二进制，并发友好，适合容器化，网络库成熟，代码统一 |
| 模拟器封装 | **Go (cgo)** | 通过 dlopen 动态加载 libretro C 内核，cgo 开销对 NES/GB 可忽略 |
| 模拟器内核 | **libretro** | 统一 API 覆盖 200+ 内核（NES: FCEUmm/Mesen, GB: mGBA） |
| 数据库 | **PostgreSQL** | 持久化用户/ROM 元数据/会话记录 |
| 缓存/会话状态 | **Redis** | 会话元数据、端口锁、存档暂存、调度状态 |
| 对象存储 | **MinIO** | S3 兼容，单机/集群皆可，存 ROM + SaveState |
| WebRTC SFU | **LiveKit** | Go SDK 成熟，开箱即用 SFU，比 Janus 更友好 |
| 容器运行时 | **Docker** → **K8s** | 渐进式部署，开发期 docker-compose，生产 K8s |
| 编码管线 | **x264 (veryfast) + Opus** | 2D 低分辨率，软件编码单核足够，无需 GPU |
| 前端 | **Vue 3 + Vite** | 见 docs/frontend.md |
| 前端流播放 | **LiveKit JS SDK** | 与后端 SFU 统一，WebRTC 自动协商 |
| 服务间通信 | **gRPC + protobuf** | 高性能二进制协议，强类型 |
| API 风格 | **REST (外部)** + **gRPC (内部)** | 外部简单 REST，内部高性能 gRPC |

---

## 4. ROM 上传

- ROM 上传至 MinIO，SHA-256 去重（同用户不可重复上传同一文件）
- **模拟器类型检测**: MVP 阶段仅根据文件扩展名判断（`.nes` / `.gba` / `.gbc`），不做魔数校验
- **上传限制**: NES < 2MB, GB < 32MB
- ROM 审核功能留待后续设计（见 docs/deferred.md）

---

## 5. 模拟器抽象层 — EmuRunner

### 5.1 libretro 调用模型

```
libretro 内核编译为 .so/.dll (如: fceumm_libretro.so, mgba_libretro.so)

Go(cgo) → dlopen 加载内核 → 注册回调 → retro_run() 驱动帧循环
                                │
        ┌───────────────────────┼───────────────────────┐
        ▼                       ▼                       ▼
  video_refresh             audio_sample            input_state
  (每帧 ~60Hz)             (~94次/秒)              (轮询按键)
        │                       │
        ▼                       ▼
   Go channel              Go channel
        │                       │
        └───────┬───────────────┘
                ▼
         编码 goroutine
      (x264 + Opus → WebRTC)
```

### 5.2 cgo 边界性能

| 回调 | 频率 | 每次数据量 | cgo 开销 | 结论 |
|------|------|-----------|---------|------|
| `video_refresh` | 60 次/秒 | 240KB (NES) | ~100ns | 可忽略 |
| `audio_sample` | ~94 次/秒 | ~2KB | ~100ns | 可忽略 |
| `input_state` | 60 次/秒 | 几个字节 | ~100ns | 可忽略 |

每会话每秒 ~150 次 cgo 调用，开销不到 0.001% 单核 CPU。

### 5.3 内核与分辨率

| 机种 | 推荐内核 | 分辨率 | 帧率 | 音频 | 单核心承载量 |
|------|---------|--------|------|------|------------|
| NES | FCEUmm / Mesen | 256×240 | 60fps | 单声道 | ~20~50 会话 |
| GB | mGBA | 240×160 | 60fps | 立体声 | ~20~50 会话 |

### 5.4 编码参数

- **视频**: x264 `veryfast` preset, 自适应码率 NES 100~800kbps / GB 200~1500kbps
- **音频**: Opus, 单声道 (NES) / 立体声 (GB), 64~96kbps
- **自适应**: 丢包 2% 以上触发降帧/降码率

### 5.5 EmuRunner 运行模型

EmuRunner 是 **Worker Agent 通过 `exec.Cmd` 启动的子进程**。每个游戏会话对应一个独立的 EmuRunner 进程，由 Worker 传入 LiveKit 房间 token 和 ROM 路径作为命令行参数。

```
Worker Agent                          EmuRunner 子进程
  │                                       │
  │  gRPC 收到 StartGame 请求              │
  ├─ 创建 LiveKit Room ────────────→ LiveKit SFU
  ├─ 获取 LiveKit Token                  │
  ├─ exec.Command("emurunner",            │
  │    "--room=xxx",                      │
  │    "--token=xxx",                     │
  │    "--rom=/path/to/rom",              │
  │    "--backend=nes")                   │
  │    └─→ cmd.Start() ─────────────────→ 进程启动
  │                                       ├─ 连接 LiveKit Room (用 token)
  │                                       ├─ dlopen 加载 libretro .so
  │                                       ├─ 加载 ROM 文件
  │                                       ├─ 启动帧循环 (retro_run)
  │                                       ├─ 编码 goroutine (x264 + Opus)
  │                                       └─ 发布 Track 到 LiveKit SFU
  │                                       │
  │  gRPC 返回 token 给 Control Plane     │  ← 浏览器用同一 token 加入房间
```

### 5.6 实现方式

Go 通过 cgo 动态加载 libretro 内核（`dlopen`），获取函数符号后：

```go
// 加载 .so/.dll → 获取 retro_init/retro_run/retro_load_game 等符号
// 注册 video_refresh → C 帧缓冲区 copy 到 Go []byte → 发到 VideoCh
// 注册 audio_sample → 累积到缓冲区 → 发到 AudioCh
// 注册 input_state → 从 DataChannel 消费前端发来的按键状态
// 主循环: retro_run() → 编码管线消费 VideoCh/AudioCh → WebRTC track
```

编码管线在独立 goroutine 中运行，不阻塞帧循环。WebRTC 推流使用 LiveKit Go SDK。

---

## 6. 多人手柄映射

```
会话: room_abc (游戏: Contra, NES, 2 手柄)
         │
    ┌────┴────┐
    ▼         ▼
 Port1     Port2    ← 模拟器手柄端口
    ▲         ▲
    │         │
 玩家A      玩家B    ← WebRTC DataChannel
```

- 每个手柄端口一次只绑定一个玩家
- 玩家连接时选择手柄位（Port1/Port2），已被占用的不可选
- DataChannel 消息格式（二进制紧凑协议）:
  ```
  [buttons:2B][dpad:1B]
  ```
  每帧发送一次（60Hz），约 4 bytes × 60 = 240 B/s，带宽可忽略（不含 port 和 frame_id）

  buttons 为 uint16 little-endian，bit 位定义如下（对齐 libretro RETRO_DEVICE_ID_JOYPAD_*）：
  - bit 0: B (=libretro id=0)
  - bit 1: Y (=libretro id=1, NES 未用)
  - bit 2: Select (=libretro id=2)
  - bit 3: Start (=libretro id=3)
  - bit 4: Up (=libretro id=4)
  - bit 5: Down (=libretro id=5)
  - bit 6: Left (=libretro id=6)
  - bit 7: Right (=libretro id=7)
  - bit 8: A (=libretro id=8)
  - bit 9: X (=libretro id=9, NES 未用)
  - bit 10: L (=libretro id=10)
  - bit 11: R (=libretro id=11)
  - bit 12: TurboA (自定义)
  - bit 13: TurboB (自定义)
  - bit 14-15: 保留

  dpad:1B 当前固定填 0，保留给后续扩展（如模拟摇杆）

  数据包格式（含 type prefix，共 4 bytes）：
  ```
  [type:1B=0x01][buttons_lo:1B][buttons_hi:1B][reserved:1B]
  ```
  通过 LiveKit DataChannel topic="input" 发送，reliable=false。

---

## 7. 一次完整房间游戏流程

```
 1. 房主创建房间: POST /api/rooms/create { title, emulator_type, rom_id, invitee_ids }
 2. 被邀请者接受邀请: POST /api/rooms/accept { invitation_id }
 3. 房主分配手柄: POST /api/rooms/assign { room_id, user_id, port }
  4. 房主开始游戏: POST /api/rooms/start { room_id }
     → Control Plane: Scheduler 选择最优 Worker (加权最低负载)
     → Control Plane: 从 MinIO 生成 ROM 文件预签名下载 URL（5 分钟有效期）
     → Control Plane: gRPC 调用 Worker.StartGame(rom_url, rom_path, emulator_type, max_ports)
  5. Worker Agent 处理 StartGame:
     a. 通过 LiveKit SDK 创建 LiveKit 房间
     b. 生成 LiveKit token（含 room 权限 + player 身份）
     c. 返回 {livekit_token, livekit_room, livekit_url} 给 Control Plane
     d. 从 MinIO 预签名 URL 下载 ROM 文件到 /tmp/cloudemu/{room_id}/rom.dat
     e. 启动 EmuRunner 子进程（传入本地 ROM 路径 + token + backend_type）
  6. 房主收到响应 → Redis 存入 {token, url, room} → 前端跳转 PlayView
     a. 用 token 连接 LiveKit 房间
     b. dlopen 加载 libretro 内核（NES/GB）
     c. 通过 os.Open() 加载本地 ROM 文件
     d. 启动帧循环 + 编码管线 → 视频/音频 Track 发布到 LiveKit SFU
     e. 订阅 DataChannel 接收玩家手柄输入 → retro_input_state 回调
 7. 所有房间玩家收到 LiveKit token → 前端建立 WebRTC 连接
    → 视频轨(接收游戏画面) + 音频轨 + DataChannel(发送手柄输入)
 8. 存档（房主手动）: 房主点击「存档」→ CP 生成 MinIO 预签名 PUT URL + gRPC Worker.SaveState
    → Worker 经 control DataChannel(0x07) 令 EmuRunner retro_serialize 写共享目录 → Worker 轮询完成标志后上传 MinIO → CP 落库 save_states
    （MVP 仅手动存档，自动定时存档见 deferred.md）
 9. 玩家离开 / 房主关闭:
    → Control Plane: gRPC 调用 Worker.StopGame(room_id)
    → Worker: kill EmuRunner 子进程 → 清理 LiveKit 房间
```

---

## 8. Worker Agent 职责

Worker Agent 是部署在 Worker 节点上的常驻服务进程（`cmd/worker/main.go`）。它本身**不运行模拟器**，而是作为会话管理器。

### 8.1 核心职责

| 职责 | 说明 |
|------|------|
| **Redis 自注册** | 启动时向 Redis **DB 1** 注册自身（工作地址、硬件权重），每 15s 心跳续期（TTL 30s）。DB 1 与业务数据（DB 0）隔离 |
| **gRPC Server** | 暴露 `StartGame` / `StopGame` / `SessionStatus` 等 RPC，供 Control Plane 调用 |
| **LiveKit 房间管理** | 收到 StartGame 后调用 LiveKit SDK 创建房间 + 生成 token |
| **EmuRunner 子进程生命周期** | 通过 `exec.Cmd` 启动/停止 emurunner 进程，监控进程状态 |
| **负载上报** | 在心跳中携带当前 `sessions` 计数（运行的 EmuRunner 子进程数） |

### 8.2 StartGame 处理流程

```
Worker.StartGame(room_id, rom_path, rom_url, emulator_type, max_ports):
  1. 通过 LiveKit SDK 创建房间 (server-sdk-go)
     → 设置房间名 = room_id
     → 设置空房间超时 = 60s（所有参与者离开后自动清理）
  2. 生成 LiveKit token:
     → 身份: emurunner（视频/音频发布权限 + DataChannel 订阅权限）
     → 返回此 token 给 Control Plane（Control Plane 再分发给玩家）
  3. 创建临时工作目录 /tmp/cloudemu/{room_id}/
  4. 从 rom_url（MinIO 预签名 URL）下载 ROM 文件到本地:
     http.Get(rom_url) → /tmp/cloudemu/{room_id}/rom.dat
  5. 启动 EmuRunner 子进程:
     exec.Command("emurunner",
       "--publisher-host", livekitHost,
       "--token", token,
       "--room", roomID,
       "--rom", "/tmp/cloudemu/{room_id}/rom.dat",  // 本地文件路径
       "--backend", emulatorType)
     → 进程后台运行，stdout/stderr 接入 Worker 日志
  6. 返回 { livekit_token, livekit_room, livekit_url } 给 Control Plane
```

### 8.3 StopGame 处理流程

```
Worker.StopGame(room_id):
  1. 找到对应 EmuRunner 子进程 → SIGTERM
  2. 超时 5s 未退出 → SIGKILL
  3. 删除 LiveKit 房间
  4. 清理临时工作目录: rm -rf /tmp/cloudemu/{room_id}/
  5. 更新 Redis 心跳中的 sessions 计数
```

### 8.4 进程监控

Worker Agent 定期检查所有子进程是否存活。若 EmuRunner 异常退出（crash），Worker 通过 gRPC 通知 Control Plane（`OnSessionCrashed` 回调），Control Plane 关闭对应房间。

### 8.5 ROM 文件下载机制

EmuRunner 使用 `os.Open()` 加载本地文件，不支持直接从 MinIO 读取。因此 Worker 需要在启动 EmuRunner 之前将 ROM 从 MinIO 下载到本地临时目录。

**下载流程：**

```
Control Plane                          Worker
    │                                       │
    │ 1. 查 DB 获取 rom.MinioPath            │
    │   (如 "rom/{user_id}/{rom_id}.nes")    │
    │                                       │
    │ 2. 调用 minioAdapter.PresignedGetURL()  │
    │   生成 5 分钟有效期的预签名下载 URL       │
    │                                       │
    │ 3. gRPC StartGame {                    │
    │      rom_url: "https://minio:9000/..." │
    │      rom_path: "rom/{user_id}/...nes"  │
    │    } ───────────────────────────────→  │
    │                                       │
    │                                    4. mkdir -p /tmp/cloudemu/{room_id}/
    │                                    5. http.Get(rom_url)
    │                                       │
    │                                    6. 写入 /tmp/cloudemu/{room_id}/rom.dat
    │                                       │
    │                                    7. exec emurunner --rom /tmp/cloudemu/.../rom.dat
    │                                       │
    │                                  StopGame 时:
    │                                    8. rm -rf /tmp/cloudemu/{room_id}/
```

**关键设计决策：**

| 决策 | 选择 | 理由 |
|------|------|------|
| 谁来下载 | **Worker** | Worker 是 EmuRunner 的父进程，有网络能力，可以控制下载时机和进度 |
| 下载方式 | **MinIO 预签名 URL** | Worker 不需要 MinIO SDK，仅用标准 `net/http`；URL 带临时认证，无需配置 MinIO 凭证 |
| URL 有效期 | **5 分钟** | Worker 收到 gRPC 后会立即下载，5 分钟足够，过期快安全性好 |
| 临时文件位置 | `/tmp/cloudemu/{room_id}/` | 每个房间独立目录，清理时整体删除，不会残留 |
| 文件清理时机 | **StopGame 时** | Worker 的 StopGame 方法中会 `os.RemoveAll()` 整个房间目录 |

### 8.6 LiveKit 地址动态获取

前端**不硬编码** LiveKit 地址。LiveKit 地址只配置在 Worker 节点上（`LiveKitHost`），通过 API 响应动态返回给前端。

**Token 策略：每个参与者独立 identity**

LiveKit 要求每个连接的 `identity` 唯一。因此 Worker 生成两类 token：
- **EmuRunner token**：`identity="emurunner"`，`canPublish=true` — 仅 EmuRunner 进程使用
- **玩家 token**：`identity="player:{user_id}"`，`canPublish=false` — 每个玩家独立一个

**数据流：**

```
Worker.StartGame:
  → emu_token (emurunner)   → EmuRunner 进程使用
  → host_token (player:{id}) → 房主在前端使用

Worker.GeneratePlayerToken(room, user_id):
  → player_token (player:{id}) → 非房主在前端使用
```

```
Worker.LiveKitManager.host ──gRPC──→ Control Plane ──REST──→ 前端
                                            │
                                     Redis: room:{id}:livekit
                                     Hash { url, room }（无 token）
```

- `POST /api/rooms/start` 返回 `{livekit_token: host_token, livekit_room, livekit_url}`
- `GET /api/rooms/:id/livekit` → CP 调 Worker `GeneratePlayerToken` → 返回该用户专属 token
- 变更 LiveKit 地址只需重启 Worker，Control Plane 和前端无需任何配置变更

### 8.7 存档 / 读档（SaveState / LoadState）

EmuRunner 只能被动接收 LiveKit control 指令，无 DB / 对象存储能力，且房间关闭后即销毁。
因此存档数据经 **Worker 中转 + 共享目录**：EmuRunner 与 Worker 同主机，共享 `/tmp/cloudemu/{room_id}/`。

**一个存档 = room_id + emulator_type + rom_id + 序列化数据**；读档时三要素须与当前房间/机种/加载的 ROM 全部匹配。
存档元数据存 PostgreSQL `save_states` 表，序列化二进制存 MinIO `savestate/{room_id}/{id}.dat`，
不随房间关闭或 EmuRunner 销毁而删除。MVP 仅房主手动存档/读档；读档仅在房间 `status=1 (playing)` 时允许。

**control DataChannel packet**：`0x07`=SaveState、`0x08`=LoadState（payload 仅 1 byte type）。

**存档流程**：

```
房主点「存档」
  → POST /api/rooms/save-state {room_id}
  → CP: 校验房主+playing+已选ROM → 生成 save_state_id + MinIO 预签名 PUT URL
       → gRPC Worker.SaveState(room_id, save_state_id, upload_url)
  → Worker.SaveState:
       1. 清理残留 state.dat / state.done
       2. control DataChannel 广播 [0x07] → EmuRunner
       3. 轮询 /tmp/cloudemu/{room_id}/state.done（间隔 100ms，超时 10s）
       4. 读 state.dat → HTTP PUT 到 MinIO 预签名 URL → 返回 size
  → EmuRunner 收到 0x07: retro_serialize → 写 state.dat → 原子 rename state.done
  → CP 落库 save_states(room_id, emulator_type, rom_id, minio_path, size) → 返回记录
```

> **存档列表过滤**：`GET /api/rooms/:id/save-states` 仅返回与房间**当前** `emulator_type` + `rom_id` 匹配的存档，避免展示同一房间切换过的其他 ROM/机种的存档；房间未选 ROM 时返回空列表。

**读档流程**：

```
房主选存档点「读取」
  → POST /api/rooms/load-state {room_id, save_state_id}
  → CP: 校验房主+playing → 查存档 → 三要素匹配校验 → 生成 MinIO 预签名 GET URL
       → gRPC Worker.LoadState(room_id, save_state_id, download_url)
  → Worker.LoadState:
       1. 清理残留 load.done → HTTP GET 下载状态二进制到 load.dat
       2. control DataChannel 广播 [0x08] → EmuRunner
  → EmuRunner 收到 0x08: 读 load.dat → retro_unserialize → 写 load.done
```

**关键设计决策**：

| 决策 | 选择 | 理由 |
|------|------|------|
| EmuRunner↔外部 状态传输 | 共享文件 + Worker 中转 | EmuRunner 无 DB/对象存储能力，复用 Worker 的网络能力与同主机共享目录 |
| 完成回执 | 共享目录标志文件轮询 | 零新增网络链路，最简单；原子 rename 避免读到半成品 |
| 每房间存档数 | 多存档槽（时间戳列表） | 房主可回溯任意历史存档 |
| 自动定时存档 | MVP 不做（见 deferred.md） | 先交付手动存档闭环 |
| 读档时机 | 仅 playing | 对活动 EmuRunner 调用 retro_unserialize |
| MinIO 上传 | Worker net/http PUT 预签名 URL | Worker 无需 MinIO SDK，与 ROM 下载同风格 |



---

## 9. 分布式调度

### 9.1 Worker 注册中心：Redis

Worker 不在 Control Plane 内存中注册，而是通过 Redis **DB 1** 自注册。Control Plane 从 Redis DB 1 读取 Worker 列表进行调度。这使 Control Plane 成为无状态服务，未来可任意扩缩。

```
Worker ──SET worker:{id} {"addr":"...","weight":120,...} EX 30──→ Redis
ControlPlane-1 ──SCAN worker:*──→ Redis
ControlPlane-2 ──SCAN worker:*──→ Redis  （多实例时零差异）
```

### 9.2 Worker 心跳数据结构

WorkerAgent 每 **15s** 向 Redis 上报心跳（TTL=30s），数据为 JSON：

```json
{
  "addr": "10.0.0.5:9090",
  "weight": 120,
  "sessions": 3,
  "max_sessions": 120,
  "cpu_percent": 45.2,
  "mem_percent": 60.1,
  "started_at": "2026-06-09T10:00:00Z"
}
```

| 字段 | 说明 |
|------|------|
| `addr` | gRPC 地址 `IP:Port` |
| `weight` | 调度权重 = CPU核心数 × 30 |
| `sessions` | 当前运行的 EmuRunner 会话数 |
| `max_sessions` | 权重（即硬上限），sessions 达到此值后不再分配 |
| `cpu_percent` | 当前 CPU 使用率（仅监控告警，不参与调度评分） |
| `mem_percent` | 当前内存使用率（仅监控告警，不参与调度评分） |
| `started_at` | Worker 启动时间 |

### 9.3 调度策略：加权最低负载优先

**评分公式**: `score = sessions / weight`

- `score` 越小越优先
- `sessions >= max_sessions` 的 Worker 被过滤掉
- 全部满员则返回"暂无可用节点"

**权重计算**（Worker 启动时自报，不在 Control Plane 侧计算）：

```
weight = CPU核心数 × 30

单核容量系数取 30（架构文档 §5.3 单核承载量 20~50，取中偏保守）
```

示例：4 核 → weight=120，8 核 → weight=240。

### 9.4 调度流程

```
RoomService.Start(roomID):
  1. 读 ROM 元数据 → 获取 minio_path, emulator_type
  2. SCAN worker:* → 获取所有存活 Worker
  3. 过滤: sessions < max_sessions
  4. 排序: score = sessions/weight 升序
  5. 选中第一个 → INCR worker:{id}:sessions
  6. 生成 MinIO 预签名 ROM 下载 URL
  7. gRPC 调用 Worker: StartGame(room_id, rom_path, rom_url, emulator_type, max_ports)
     → Worker 创建 LiveKit room + 生成 token + 下载 ROM → 返回 {token, room, url}
     → Worker 启动 EmuRunner 子进程
  8. gRPC 超时（5s）→ DECR worker:{id}:sessions → fallback 到下一个 Worker
  9. Redis: HSET room:{id}:livekit {token, url, room}
  10. 返回 {livekit_token, livekit_room, livekit_url} 给前端
```

### 9.5 心跳与故障处理

| 场景 | 处理 |
|------|------|
| Worker 正常心跳 | 每 15s `SET worker:{id} {...} EX 30` |
| Worker 宕机/断网 | TTL 30s 后 key 自动过期，Worker 从可用列表中消失 |
| Worker 恢复后重连 | 重新 SET，sessions 从 0 开始（之前的会话已随 Worker 丢失） |
| Worker 心跳延迟 | 只要 TTL 未到期都视为存活 |

**当前不做故障会话恢复**：Worker 宕机后其上运行的所有 EmuRunner 会话丢失，依赖定时 SaveState 的故障恢复留待 Phase 3 设计。

### 9.6 设计约束

- **不做会话迁移/重调度**: 已运行的 EmuRunner 不移动，只调度新会话
- **不按模拟器类型分权**: NES 与 GB 的资源消耗差异不够大，MVP 不区分
- **CPU/内存仅作监控**: 不参与调度评分，避免启动初期 CPU 虚高误导调度器
- **权重由 Worker 自报**: Control Plane 不假设 Worker 硬件配置

---

## 10. 目录结构

详见 [docs/project-structure.md](./project-structure.md)。

---

## 11. 部署形态（渐进式）

### 阶段 1: 单一主机开发验证
- `docker-compose up` 启动所有组件
- 控制面 + Worker + SFU + DB 全在同一台机器
- 验证完整闭环: 上传ROM → 启动游戏 → WebRTC游玩

### 阶段 2: 小规模多节点
- 1 台控制节点 (单体 + PostgreSQL + Redis + MinIO + LiveKit)
- N 台 Worker 节点 (WorkerAgent + EmuRunner 容器)
- 仅 Worker 水平扩展

### 阶段 3: 正式分布式（可选）
- Kubernetes 编排，控制面 2~3 副本，Worker HPA 自动伸缩
- 边缘多区域部署（降低延迟）
- 会话状态热迁移（基于定时 SaveState）

---

## 12. 低延迟与低带宽策略

| 策略 | 说明 |
|------|------|
| WebRTC UDP | 避免 TCP 重传抖动 |
| 自适应码率 | 根据丢包/BWE 动态调整分辨率/帧率 |
| 帧跳过 | 丢包严重时果断跳帧，保证操作跟手 |
| 边缘部署 | Worker 节点部署在离用户近的区域 |
| 软件编码 | x264 veryfast + Opus，对 2D 低分辨率已足够 |
| 数据通道压缩 | 二进制协议，含 type prefix 共 4 bytes/帧，240 B/s |

---

## 13. 未来扩展点

- 更多机种: SNES、MD、PS1（PS1 需 3D 渲染 + GPU 编码）
- 回放系统: 记录每帧输入 + 定时存档，支持录像回放
- 更多部署区域: 边缘多区域部署降低延迟

> 调度器、ROM 审核、聊天等功能的细节设计见 [docs/deferred.md](./deferred.md)。
