# AGENTS.md — CloudEmu 项目指南

本文档面向 AI Agent 和开发者，提供项目基本信息、文档索引和开发注意事项。

---

## 项目概述

CloudEmu 是一个分布式云端模拟器系统。在服务端运行 NES/GB 模拟器，通过 WebRTC 实时将游戏画面流式传输到浏览器，支持多人远程同玩（多手柄映射、邀请制房间）。

- **语言**: Go (后端) + Vue 3 / TypeScript (前端)
- **当前阶段**: 后端 Control Plane 已编码完成（30 个 Go 文件），前端 Axios + 类型封装已完成、Pinia stores + 4 个视图 + 4 个组件已开发（23 个 TS/Vue 文件）
- **数据库**: 所有表以 snake_case 命名，UUIDv7 PK（应用层 `uuid.Must(uuid.NewV7())` 生成），Gorm AutoMigrate，TIMESTAMPTZ，无 FK 约束。7 张表：users、email_verifications、refresh_tokens、friends、rooms、room_players、roms
- **接口约定**: 所有接口集中定义在 `contract/` 包
- **日志**: `log/slog` + `lestrrat-go/file-rotatelogs`（按天轮转 + stdout 双输出）

---

## 文档索引

| 文档 | 内容 | 何时查阅 |
|------|------|----------|
| [docs/architecture.md](./docs/architecture.md) | 总体架构、EmuRunner 设计、多人手柄映射、完整游戏流程、分布式调度、部署形态 | 理解系统全貌、新增核心组件 |
| [docs/db.md](./docs/db.md) | 7 张 PostgreSQL 表 DDL、Redis 键设计、认证流程、好友流程、房间流程 | 修改数据库结构、编写 SQL/迁移 |
| [docs/project-structure.md](./docs/project-structure.md) | Go 工程目录、contract 接口定义、全部 DTO、API 路由表、依赖注入、slog 日志方案、前端架构 | 新增接口、修改路由、添加模块 |
| [docs/frontend.md](./docs/frontend.md) | Vue 3 架构、路由、Pinia stores、composables、页面布局、Steam-dark 主题 | 前端开发、页面修改 |
| [docs/deferred.md](./docs/deferred.md) | 延后功能清单：调度器、DataChannel 流向、邀请过期、ROM 审核、聊天、实时通知 | 讨论新功能是否需要延后 |

---

## 技术栈速查

| 层 | 技术 | 关键依赖 |
|----|------|----------|
| 后端框架 | Go | Gin, Gorm, golang-jwt/v5, go-redis/v9, google/uuid（UUIDv7）, bcrypt, file-rotatelogs |
| WebRTC SFU | LiveKit | LiveKit Go SDK, LiveKit JS SDK |
| 存储 | PostgreSQL + Redis + MinIO (S3) | Gorm (PG), go-redis (Redis), minio-go |
| 模拟器 | libretro (C) | cgo + dlopen 加载 .so 内核 |
| 视频编码 | x264 + Opus | pion/webrtc + LiveKit SDK |
| 前端 | Vue 3 + TypeScript + Vite | Naive UI, Pinia, Axios, @vueuse/core, LiveKit JS SDK（待集成） |
| Worker 通信 | gRPC | protobuf (proto/worker.proto) |

---

## 关键约定

### 命名与风格
- **Go 目录**: 全小写，`control-plane`、`emurunner`
- **Go 包名**: 与目录名一致，全小写，单数（`model` 不叫 `models`）
- **Go 接口**: 集中在 `contract/` 包，不在实现包内定义 `type XxxService interface{}`
- **数据库表名**: `snake_case` 复数（`users`、`room_players`）
- **数据库字段**: `snake_case`（`created_at`、`emulator_type`）
- **JSON field tag**: `snake_case`，与 DB 字段一致
- **API 路由**: `/api/` 前缀，REST 风格（`/api/auth/register`）

### 三层架构（Control Plane）
```
handler → service → repo
   ↓         ↓        ↓
  HTTP    业务逻辑  数据访问
```
- **handler**: 参数校验、调用 service、返回 JSON（用 `response.Body`）
- **service**: 业务逻辑、调用 repo 和 cache、事务管理
- **repo**: 纯数据库操作（Gorm）
- **错误**: 统一用 `apperror.AppError`（code + message + httpStatus）

### 术语表
| 术语 | 含义 | 不要用 |
|------|------|--------|
| rooms | 房间/游戏会话 | sessions |
| Control Plane | 控制面 | master, controller |
| Worker | 工作节点（运行模拟器） | node, slave |
| EmuRunner | 模拟器运行时（libretro wrapper） | emulator |
| port | 模拟器手柄端口 | controller, gamepad |

### 状态枚举
| 域 | 值 |
|----|-----|
| 用户状态 | 0=pending, 1=active, 2=disabled |
| 好友状态 | 0=pending, 1=accepted, 2=blocked, 3=rejected |
| 房间状态 | 0=waiting, 1=playing, 2=closed |
| 房间玩家角色 | 0=host, 1=player, 2=spectator |
| ROM 状态 | 0=pending, 1=approved, 2=rejected |

### JWT 方案
- 签名算法: HS256
- Access Token: 24h, payload 含 `user_id`
- Refresh Token: 7d, SHA-256 hash 后存入 `refresh_tokens` 表，轮换制

---

## 开发注意事项

### 不要做的事
- **不要**在 `model/` 或 `service/` 包中定义接口 — 接口在 `contract/` 包
- **不要**添加 Foreign Key 约束 — Gorm `DisableForeignKeyConstraintWhenMigrating: true`
- **不要**使用 WebSocket（当前阶段）— 实时通知留待后续设计
- **不要**实现聊天功能（MVP 阶段）— 已列入 `deferred.md`
- **不要**实现 ROM 管理员审核 — 用户上传后直接可见
- **不要**创建独立的 migration SQL 脚本 — 开发阶段用 AutoMigrate
- **不要**在 DTO 中使用 `multipart.File` — 文件由 handler 通过 `c.FormFile()` 处理
- **注释**: 全部使用中文注释，model 字段含义、interface 方法作用、service 业务流程
- **包管理**: 后端用 `go mod`，前端用 `pnpm`

### 必须做的事
- 新增 API 路由前先更新 `docs/project-structure.md` 的 DTO 和路由表
- 新增数据库表/字段前先更新 `docs/db.md`
- 新增 contract 接口后同步更新依赖注入图
- 处理 `UploadRomReq` 时，handler 用 `c.FormFile()` 获取文件，不在 DTO 结构体放文件字段
- 所有 API 响应通过 `response.Body{Code, Msg, Data}` 统一封装
- ROM 封面图片路径 `cover_path` 可为 NULL，NULL 时前端用模拟器类型默认封面
- **DTO 中所有 ID 字段一律用 `*uuid.UUID`**，必填项加 `binding:"required,notnil_uuid"`（自定义校验器在 `router/validator.go` 中注册，拒绝 nil 指针和 `uuid.Nil`，防止前端传 "00000000-..." 绕过校验）
- **model 中主键和业务必填的外键 ID 用 `uuid.UUID` 值类型**；**业务上允许为空的外键 ID**（如 `Room.RomID`，房主创建房间时可暂不选 ROM）必须用 `*uuid.UUID` 并去掉 gorm 的 `not null` 标签
- Handler 中解析 URL Path 上的 UUID 一律用 `parseUUIDParam(c, key)`，禁止直接 `uuid.Parse(c.Param(...))`

### 关键边界条件
- 前端路由: `/login`, `/register`, `/`(Lobby), `/profile`, `/room/:id`, `/play/:roomId`
- ROM 文件通过 `POST /api/roms/upload` 上传后存 MinIO，路径记录 `minio_path`
- ROM 文件/封面图片通过 `GET /api/files/*path` 代理获取（经过 Control Plane，不直接暴露 MinIO）
- 房间邀请接受前检查 `current_active_players < max_ports`（满员拒绝）
- 房间创建时 host 自动加入 `room_players` (role=host)，invitee_ids 中的好友直接加入（role=spectator，无需接受）
- 房间邀请好友加入已有房间：房主调用 `POST /api/rooms/invite`，好友直接加入 room_players（类似微信拉群）
- 房间开始游戏后 `rooms.status = 1 (playing)`
- 定时存档频率: 每 60s 调用 `retro_serialize()` → Redis
- Worker 心跳: 每 15s `SET worker:{id} {json} EX 30` → Redis **DB 1**（TTL 自动过期即视为宕机）
- Redis DB 隔离: DB 0 = 业务数据（captcha/room state/limiter），DB 1 = Worker 注册与调度数据
- **Worker 职责**: Worker Agent 不运行模拟器 — 它通过 `exec.Cmd` 启动 EmuRunner 子进程，并负责创建 LiveKit 房间 + 生成 token
- **EmuRunner 职责**: EmuRunner 是 Worker 的子进程 — 连接 LiveKit、加载 ROM、运行模拟器 + 视频编码 + WebRTC 推流
- **LiveKit 房间**: 由 Worker Agent 通过 LiveKit SDK 创建，token 由 Worker 生成返回给 Control Plane，EmuRunner 和浏览器用同一 token 加入房间
- Worker 调度: Control Plane 从 Redis 读取存活 Worker 列表，加权最低负载优先（score = sessions / weight，weight = CPU核心数 × 30）
- Worker 故障: TTL 过期即消失，MVP 不做会话恢复

### libretro 回调频率
| 回调 | 频率 | 数据量 | 备注 |
|------|------|--------|------|
| video_refresh | 60 Hz | ~240KB (NES) | 每帧一次 |
| audio_sample | ~94 次/秒 | ~2KB/次 | 累积后编码 |
| input_state | 60 Hz | 几个字节 | 轮询当前按键 |

---

## 已决策事项（不要重新讨论）

1. **前端使用 Vue 3 + Naive UI**（不是 React）
2. **术语用 "rooms"**（不是 sessions）
3. **MVP 无 WebSocket**（实时通知用 polling 或 LiveKit DataChannel）
4. **MVP 无聊天**
5. **ROM 审核暂不实现**（上传即用）
6. **Worker 注册与调度**：Redis 自注册（SET key TTL=30s，每 15s 心跳），加权最低负载优先（score = sessions / weight，weight = CPU核心数 × 30），见 architecture.md §8
7. **邀请过期机制延后**（见 deferred.md）
8. **每玩家独立 DataChannel 拓扑**（不是共享通道）
9. **二进制输入协议**: `[buttons:2B][dpad:1B]`（3 bytes/帧，无 port、无 frame_id）
10. **Gorm AutoMigrate** 开发阶段建表
11. **UUIDv7** 应用层 `uuid.Must(uuid.NewV7())` 生成主键，不依赖 DB 默认值
12. **ROM 模拟器类型检测** 仅用文件扩展名（MVP），不做魔数校验
