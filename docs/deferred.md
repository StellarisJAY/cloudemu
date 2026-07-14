# CloudEmu 待讨论 / 延后设计项

本文档汇总当前阶段搁置、留待后续设计的功能点。

---

## 1. Worker 调度器

**来源**: `architecture.md` §9 分布式调度

Control Plane 的 Room Manager 将游戏启动请求分配给最优 Worker 节点，调度器和 gRPC 通信已实现。

### 已决策（2026-06-09）

| 决策项 | 结论 |
|--------|------|
| 注册中心 | **Redis**（复用已有，Worker SET key TTL=30s 自注册，Control Plane SCAN 发现） |
| 调度策略 | **加权最低负载优先**（score = sessions / weight），weight = CPU核心数 × 30 |
| Worker 注册方式 | WorkerAgent 启动时主动向 Redis 注册（push），Control Plane 被动发现 |
| 心跳机制 | 每 15s SET，TTL 30s 自动过期 |
| 故障恢复 | **MVP 不做会话恢复**，Worker 宕机后其上 EmuRunner 全部丢失；定时 SaveState 恢复留待 Phase 3 |
| 代码位置 | `internal/control-plane/scheduler/`（Scheduler）+ `internal/control-plane/grpc/`（WorkerClient） |

### 已实现（2026-06-09）

- [x] Worker 数据模型（`contract/scheduler.go` WorkerInfo + WorkerRegistry + Scheduler + WorkerClient 接口）
- [x] Worker 注册与心跳（`internal/worker/heartbeat.go` — 每 15s SET，TTL 30s）
- [x] WorkerAgent 进程（`cmd/worker/main.go` — Redis DB 1 自注册 + 心跳循环 + gRPC Server）
- [x] Control Plane WorkerRegistry（`cache/worker_registry.go` — Redis SCAN + GET）
- [x] Control Plane 双 Redis 客户端（DB 0 业务数据 + DB 1 Worker 调度数据）
- [x] Worker 独立配置（`internal/worker/config.go`）
- [x] Proto 定义（`proto/worker.proto` — StartGame / StopGame / SessionStatus）
- [x] Worker gRPC Server（`internal/worker/grpc/` — 3 个 RPC 实现）
- [x] Worker LiveKit 管理（`internal/worker/livekit.go` — 房间创建 + Token 生成）
- [x] Worker EmuRunner 子进程管理（`internal/worker/process.go` — 启动/停止/监控）
- [x] Control Plane Scheduler 实现（`internal/control-plane/scheduler/` — 加权最低负载优先）
- [x] Control Plane gRPC 客户端（`internal/control-plane/grpc/worker_client.go` — 连接池 + 调用）
- [x] RoomService.Start() 集成调度 + gRPC 全流程
- [x] RoomService.Leave() 房间关闭时 fire-and-forget 调用 Worker.StopGame()
- [x] Room model 新增 `worker_addr` 字段（记录分配到的 Worker）
- [x] `POST /api/rooms/start` 返回 `StartRoomResp`（含 livekit_token + livekit_room）
- [x] 错误码：ErrNoAvailableWorker (503) / ErrWorkerUnavailable (502)

---

## 2. LiveKit DataChannel 数据流向

**来源**: `architecture.md` §5.1 EmuRunner 设计, `frontend.md` PlayView

当前设计的输入路径模糊：
- 浏览器通过 LiveKit DataChannel 发送手柄输入 → LiveKit SFU
- Worker 的 EmuRunner 需要从 LiveKit **接收** DataChannel 数据 → `input_state` 回调

### 待决策点
- **Worker 如何订阅 DataChannel**: EmuRunner 作为 LiveKit participant 加入房间并订阅其他 participant 的 data track？
- **输入延迟估算**: DataChannel (UDP) 下的端到端延迟 + cgo 调用开销
- **多人输入路由**: 多 player → 多 DataChannel → EmuRunner 内部多 port 分发逻辑
- **二进制协议格式**: 是否需要 port + timestamp 前缀以保证时序？

### 相关文件
- `architecture.md` §5.1, §6
- `internal/worker/emurunner/runner.go`
- `internal/worker/livekit/publisher.go`
- `frontend.md` useLiveKit composable

---

---

## 3. ROM 审核系统 + 魔数校验

**来源**: `architecture.md` §4（已移除）, `db.md` roms 表

ROM 的 `status` 字段支持 `0=pending, 1=approved, 2=rejected`，但管理审核功能被暂缓。

### 待决策点
- **是否需要审核**: MVP 阶段所有 ROM 默认 `approved`（status=1），上传即用
- **审核面板**: Web 管理后台还是 CLI 工具？
- **管理员角色**: 是否需要独立的 `admin` 用户角色和权限系统？
- **ROM 去重策略**: 基于 SHA-256 的用户级去重（当前设计）vs 全局去重共享
- **非法内容检测**: 是否需要额外机制防止用户上传非游戏文件？
- **魔数校验**: NES 需检测 `0x4E 0x45 0x53 0x1A`，GB 需检测头部标识，MVP 仅用扩展名

### 相关文件
- `db.md` roms 表

---

## 4. 聊天系统（文本/语音）

**来源**: `frontend.md` RoomView/PlayView（已移除）

聊天面板最初出现在前端布局草图中，但在 MVP 中暂不实现。

### 待决策点
- **文本聊天**: 是否需要独立的 WebSocket 服务？还是复用 LiveKit DataChannel？
- **语音聊天**: LiveKit 原生支持音频轨道，是否开启房间内语音？
- **消息持久化**: 聊天记录是否持久化到 PostgreSQL？还是仅 Redis 实时传递？
- **WebSocket 架构**: 集成到 Control Plane 还是独立服务？

### 相关文件
- (暂无相关后端表/接口设计)

---

## 5. 实时通知机制

**来源**: 从 `project-structure.md` 和 `frontend.md` 移除的 WebSocket

当前房间邀请、手柄分配等实时事件的通知机制未确定。

### 待决策点
- **WebSocket 接入方式**: 直连 Control Plane（`gorilla/websocket`）还是通过 LiveKit DataChannel？
- **轮询降级**: 如果暂不实现 WebSocket，是否用短轮询（`GET /api/rooms/:id/state`）作为替代？
- **断线重连**: WebSocket 断开后的重连策略 + 状态恢复

### 相关文件
- `frontend.md` useWebSocket composable (已移除)
- `project-structure.md` ws/ hub (已移除)

---

## 6. 其他待讨论

- **房间人数上限**: 统一最大4人，房间 `max_ports` 是否允许更多旁观者？
- **Gorm AutoMigrate 索引**: 部分唯一索引（如 `uk_friends_pair`、`uk_rp_active`）依赖表达式和 WHERE 子句，Gorm AutoMigrate 是否自动创建？
- **SaveState 持久化**: 定时存档从 Redis 持久化到 MinIO 的时机和策略
- **前端 Gamepad API 集成**: 浏览器端是否支持真实手柄（`navigator.getGamepads()`）作为输入源替代键盘映射？
