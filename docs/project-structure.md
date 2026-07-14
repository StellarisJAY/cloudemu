# CloudEmu 项目结构 & 代码规范

## 技术栈

| 层 | 选型 |
|----|------|
| Web 框架 | `github.com/gin-gonic/gin` |
| ORM | `gorm.io/gorm` + `gorm.io/driver/postgres` |
| JWT | `github.com/golang-jwt/jwt/v5` |
| UUID | `github.com/google/uuid`（UUIDv7） |
| Redis | `github.com/redis/go-redis/v9` |
| 配置 | 环境变量（`os.Getenv`），无 viper |
| 日志 | `log/slog` + `lestrrat-go/file-rotatelogs`（按天轮转） |
| 密码 | `golang.org/x/crypto/bcrypt` |
| 包管理 | `go mod` |

| 前端 | 选型 |
|------|------|
| 框架 | Vue 3 + TypeScript |
| 构建 | Vite |
| HTTP | Axios |
| 状态管理 | Pinia |
| 路由器 | Vue Router 4 |
| UI 组件库 | Naive UI |
| 工具库 | @vueuse/core |
| 包管理 | pnpm |

---

## 目录结构

```
cloudemu/
├── cmd/
│   ├── control-plane/
│   │   └── main.go                     # 主控平面入口 + 依赖注入装配
│   └── worker/                         # Worker Agent 节点
│       └── main.go                     # Worker 入口（Redis 自注册 + 心跳 + gRPC Server 启动）
│
├── proto/                              # gRPC Proto 定义
│   └── worker.proto                    # WorkerAgent 服务定义：StartGame / StopGame / SessionStatus
│
├── internal/
   │   ├── control-plane/                  # 主控平面所有业务代码
   │   │   ├── contract/                   # 集中定义接口 + DTO
   │   │   │   ├── auth.go                 # AuthService, UserRepo, EmailVerificationRepo, RefreshTokenRepo, CaptchaCache, SlideCaptchaData
    │   │   │   ├── room.go                 # RoomService, RoomRepo, RoomPlayerRepo, RoomStateCache
   │   │   │   ├── rom.go                  # RomService, RomRepo, MinioFunc
   │   │   │   ├── friend.go               # FriendService, FriendRepo
   │   │   │   ├── scheduler.go            # Scheduler, WorkerClient, WorkerRegistry 接口 + WorkerInfo/StartGameRequest/StartGameResponse DTO
   │   │   │   └── dto.go                  # 全部请求/响应 DTO（20 个 Req + 11 个 Resp）+ Token TTL 常量
   │   │   │
   │   │   ├── model/                      # 9 张 GORM Model 定义（uuid7 主键，中文注释）
│   │   │   ├── user.go                 # User — 用户表（11 字段）
│   │   │   ├── email_verification.go   # EmailVerification — 邮箱验证码（7 字段）
│   │   │   ├── refresh_token.go        # RefreshToken — 刷新令牌（5 字段）
│   │   │   ├── password_reset.go       # PasswordReset — 密码重置 token（7 字段）
│   │   │   ├── friend.go               # Friend — 好友关系（7 字段，含 rejected 状态）
   │   │   │   ├── room.go                 # Room — 游戏房间（11 字段，含 worker_addr）
    │   │   │   ├── room_player.go          # RoomPlayer — 房间座位（7 字段）
    │   │   │   └── rom.go                  # Rom — ROM 库（12 字段）
   │   │   │
   │   │   ├── repo/                       # 数据访问层（struct，不 import contract）
   │   │   │   ├── user_repo.go            # UserRepo, EmailVerificationRepo, RefreshTokenRepo
    │   │   │   ├── room_repo.go            # RoomRepo, RoomPlayerRepo
   │   │   │   ├── rom_repo.go             # RomRepo
   │   │   │   ├── friend_repo.go          # FriendRepo
   │   │   │   └── minio.go                # MinioAdapter（实现 contract.MinioFunc）
   │   │   │
     │   │   ├── cache/                      # Redis 缓存（struct，不 import contract）
     │   │   │   ├── captcha.go              # Captcha — 图形验证码缓存（含 captcha_verified key）
     │   │   │   ├── room_state.go           # RoomState — 房间端口映射（Hash：room:{id}:ports）
     │   │   │   ├── limiter.go              # Limiter — 频率限制（登录/验证/重发）
     │   │   │   └── worker_registry.go      # WorkerRegistry — Worker 存活列表（Redis SCAN + GET，DB 1）
    │   │   │
    │   │   ├── scheduler/                  # Worker 调度器
    │   │   │   └── scheduler.go            # 加权最低负载优先选择算法
    │   │   │
    │   │   ├── grpc/                       # gRPC 客户端
    │   │   │   └── worker_client.go        # WorkerClient — 连接池 + StartGame/StopGame gRPC 调用
   │   │   │
   │   │   ├── service/                    # 业务逻辑层
   │   │   │   ├── auth_service.go         # AuthService — 注册/登录/验证/刷新Token/更新资料
   │   │   │   ├── room_service.go         # RoomService — 房间创建/邀请/分配/开始/离开
   │   │   │   ├── rom_service.go          # RomService — ROM 上传/列表
   │   │   │   └── friend_service.go       # FriendService — 好友添加/接受/拒绝/列表
   │   │   │
   │   │   ├── handler/                    # gin handler（薄层）
│   │       │   ├── auth.go             # AuthHandler — 13 个认证接口
   │   │   │   ├── room.go                 # RoomHandler — 6 个房间接口
   │   │   │   ├── rom.go                  # RomHandler — 上传 + 列表 + 更新
   │   │   │   ├── admin.go                # AdminHandler — 平台内置 ROM 管理（增删改查）
   │   │   │   ├── friend.go               # FriendHandler — 5 个好友接口
   │   │   │   └── files.go                # FileHandler — MinIO 文件代理
   │   │   │
   │   │   └── router/                     # 路由 + 中间件
│   │     ├── router.go               # gin.Engine 初始化、CORS、路由注册（共 27 个端点）
   │   │       └── middleware.go           # JWTAuth — JWT 解析注入中间件
   │   │
    │   ├── worker/                         # Worker Agent 代码
    │   │   ├── config.go                   # Worker 独立配置（gRPC 地址、Redis、LiveKit、EmuRunner 路径）
    │   │   ├── heartbeat.go                # Redis 心跳注册 + 负载上报
    │   │   ├── grpc/                       # gRPC Server 实现
    │   │   │   └── server.go               # StartGame / StopGame / SessionStatus 实现
    │   │   ├── process.go                  # EmuRunner 子进程管理（启动/停止/监控）
    │   │   └── livekit.go                  # LiveKit 房间创建 + Token 生成
   │   │
   │   ├── proto/                           # 生成的 protobuf 代码
   │   │   └── worker/
   │   │       ├── worker.pb.go             # protobuf 消息序列化（protoc-gen-go 生成）
   │   │       └── worker_grpc.pb.go        # gRPC 客户端/服务端 stub（protoc-gen-go-grpc 生成）
   │   │
   │   └── pkg/                            # 跨组件共享工具包
   │       ├── apperror/
   │       │   └── error.go                # AppError 统一错误类型 + 32+ 个预定义错误
   │       ├── config/
   │       │   └── config.go               # Config 结构体（23 字段，含 SMTP + FrontendBaseURL）+ 环境变量加载
   │       ├── email/
   │       │   └── sender.go               # SMTPSender + NoopSender 邮件发送器（net/smtp stdlib）
   │       ├── jwt/
   │       │   └── jwt.go                  # JWT HS256 签发 + 解析
   │       ├── logging/                    # slog 统一日志方案
   │       │   ├── logger.go               # Logger 工厂（stdout + 按天轮转文件）
   │       │   ├── gin.go                  # Gin 日志中间件
   │       │   └── gorm.go                 # Gorm 日志适配器
   │       └── response/
   │           └── response.go             # Body 统一响应封装（6 个方法）
│
   ├── web/                                # 前端 Vue 3 + Vite + Naive UI
   │   ├── index.html
   │   ├── package.json                    # pnpm 依赖管理
   │   ├── vite.config.ts                  # Vite 配置（@ 别名）
   │   ├── tsconfig.json                   # TypeScript 配置（引用拆分）
   │   ├── tsconfig.app.json               # 应用 TS 配置（@/* 路径别名）
   │   ├── tsconfig.node.json              # Node 侧 TS 配置
   │   ├── .prettierrc.json                # Prettier 格式化
   │   ├── .oxlintrc.json                  # Oxlint 配置
   │   ├── eslint.config.ts                # ESLint 配置
   │   ├── env.d.ts                        # Vite 环境变量类型声明
   │   │
   │   ├── public/
   │   │   └── assets/
   │   │       ├── default-cover-nes.png    # NES 默认封面
   │   │       └── default-cover-gb.png   # GB 默认封面
   │   │
   │   └── src/
   │       ├── main.ts                     # Vue 应用入口
   │       ├── App.vue                     # 根组件
   │       │
   │       ├── api/                        # Axios 封装 + 接口函数
   │       │   ├── client.ts               # axios 实例 + 请求/响应拦截器（token 注入 + 401 自动刷新）
│   │       ├── auth.ts                 # 认证 API（12 个端点：captcha, verifyCaptcha, register, verifyEmail, login, resendCode, refresh, me, updateProfile, updatePassword, forgotPassword, resetPassword）
   │       │   ├── room.ts                 # 房间 API（6 个端点）
   │       │   ├── rom.ts                  # ROM API（3 个端点：list, upload, update）
   │       │   ├── admin.ts                # 管理员 API（4 个端点：listBuiltin, uploadBuiltin, updateBuiltin, deleteBuiltin）
   │       │   └── friend.ts               # 好友 API（6 个端点：list, pending, search, add, accept, reject）
   │       │
   │       ├── types/
   │       │   └── api.ts                  # TypeScript 类型定义（与后端 DTO、Model 一一对应；User.is_admin / Rom.is_builtin）
   │       │
   │       ├── utils/
   │       │   └── token.ts                # localStorage token 读写工具
   │       │
   │       ├── router/
   │       │   └── index.ts                # 路由定义（含 /admin，admin meta 守卫查 is_admin）+ 导航守卫（auth/guest/admin meta）
   │       │
   │       ├── stores/                     # Pinia 状态管理
   │       │   ├── auth.ts                 # useAuthStore — 用户信息、登录状态、isAdmin getter
   │       │   ├── admin.ts                # useAdminStore — 平台内置 ROM 管理状态
   │       │   └── friend.ts               # useFriendStore — 好友列表、待处理、搜索结果
   │       │
   │       ├── views/                      # 页面级组件
   │       │   ├── LoginView.vue           # 登录页（guest only）
   │       │   ├── RegisterView.vue         # 注册页（guest only）
   │       │   ├── ForgotPasswordView.vue    # 忘记密码页（guest only）
   │       │   ├── ResetPasswordView.vue     # 重置密码页（guest only）
   │       │   ├── LobbyView.vue           # 大厅首页（需登录，顶栏含管理员入口）
   │       │   ├── AdminView.vue           # 管理后台（需管理员，平台内置 ROM 管理）
   │       │   └── ProfileView.vue         # 个人设置页（需登录）
   │       │
   │       ├── components/                 # 组件
   │       │   ├── auth/
   │       │   │   └── SlideCaptcha.vue    # 滑块验证码组件
   │       │   ├── common/
   │       │   │   └── AnimatedBackground.vue  # 动态背景
   │       │   └── friend/
   │       │       ├── AddFriendDialog.vue  # 添加好友弹窗
   │       │       └── FriendList.vue       # 好友列表组件
   │       │
   │       ├── composables/               # 组合式函数
   │       │   └── useTheme.ts             # 深浅主题切换（localStorage 持久化）
   │       │
   │       └── styles/                     # 样式
   │           ├── tokens.css              # CSS 自定义属性
   │           └── naive-overrides.ts      # Naive UI 主题覆写（Midnight Arcade / Morning Cartridge）
│
├── docs/                               # 设计文档
│   ├── architecture.md
│   ├── db.md
│   ├── frontend.md
│   ├── deferred.md
│   └── project-structure.md
│
├── AGENTS.md
├── go.mod
└── go.sum
```

---

## 依赖关系

```
handler ──→ contract  ←── service
               ↑             │
               │             ├──→ repo     (struct, 不 import contract)
               │             ├──→ cache    (struct, 不 import contract)
               │             └──→ limiter  (service 内部接口)
               │
            main.go 在注入时做类型检查，Go 编译期保证匹配
```

| 包 | 可 import | 不可 import |
|----|-----------|-------------|
| handler | contract, model, pkg/response, pkg/apperror | repo, cache, service |
| service | contract, model, pkg/apperror | handler, repo, cache |
| repo | model | handler, service, contract, cache |
| cache | model | handler, service, repo, contract |
| contract | model | handler, service, repo, cache |
| logging | config | handler, service, repo, cache, contract |

---

## Interface 集中在 contract 包

所有接口定义在 `contract/` 中，按业务模块分文件。实际代码含完整中文注释，此处为接口概览。

### contract/auth.go — 6 个接口 + 1 个实用接口

| 接口 | 方法 | 说明 |
|------|------|------|
| `AuthService` | Register, Login, VerifyEmail, ResendCode, RefreshToken, Captcha, VerifyCaptcha, Me, UpdateProfile, UpdatePassword, ForgotPassword, ResetPassword, Search | 认证业务逻辑 |
| `UserRepo` | Create, ByID, ByEmail, ByUsername, UpdateStatus, UpdateLastLogin, UpdateProfile, Search | 用户表操作 |
| `EmailVerificationRepo` | Create, LatestByEmail, MarkVerified | 邮箱验证码操作 |
| `RefreshTokenRepo` | Create, ByHash, DeleteByUser, DeleteByHash | 刷新令牌操作 |
| `PasswordResetRepo` | Create, ByHash, MarkUsed | 密码重置 token 操作 |
| `CaptchaCache` | Set, GetAndDel, SetVerified, IsVerified | 图形验证码缓存 |
| `EmailSender` | Send | 邮件发送接口（SMTPSender / NoopSender）|
| `SlideCaptchaData` | MasterBgBase64, TileBase64, ThumbX, ThumbY, Width, Height | 滑块验证码数据（生成器接口） |

### contract/room.go — 3 个接口

| 接口 | 方法 | 说明 |
|------|------|------|
| `RoomService` | Create, List, InviteToRoom, AssignPort, Start, Leave | 房间业务逻辑；Start 返回 `*StartGameResponse`（含 LiveKit token）；InviteToRoom 直接加入好友（类似微信拉群） |
| `RoomRepo` | Create, ByID, ActiveByUser, UpdateStatus, SetWorkerAddr | 房间表操作；SetWorkerAddr 记录分配到的 Worker |
| `RoomPlayerRepo` | Create, ActiveByRoom, ActiveByUser, ByRoomAndUser, UpdateRoleAndPort, MarkLeft, TransferHost | 房间座位操作 |
| `RoomStateCache` | SetPort, RemovePort, GetPorts, ClearRoom | 房间端口映射缓存 |

### contract/scheduler.go — 3 个接口

| 接口 | 方法 | 说明 |
|------|------|------|
| `WorkerRegistry` | ListAlive() → ([]WorkerInfo, error) | 从 Redis SCAN 获取所有存活 Worker |
| `Scheduler` | SelectWorker(ctx, registry) → (*WorkerInfo, error) | 加权最低负载优先选择 Worker（已实现） |
| `WorkerClient` | StartGame, StopGame | gRPC 客户端接口，Control Plane 调用 Worker Agent |

### contract/rom.go — 2 个接口

| 接口 | 方法 | 说明 |
|------|------|------|
| `RomService` | Upload, List, Update, UploadBuiltin, ListBuiltin, UpdateBuiltin, DeleteBuiltin | ROM 业务逻辑（含平台内置 ROM 管理） |
| `RomRepo` | Create, Update, Delete, ByID, ListForUser, ListBuiltin, BySHA256, BuiltinBySHA256 | ROM 表操作 |
| `MinioFunc` | UploadFile, GetURL, GetFile, RemoveFile | MinIO 文件操作 |

### contract/friend.go — 1 个接口

| 接口 | 方法 | 说明 |
|------|------|------|
| `FriendService` | Add, Accept, Reject, List, Pending, SearchUsers | 好友业务逻辑 |
| `FriendRepo` | Create, ByPair, AcceptedByUser, PendingByUser, UpdateStatus, SearchUsers | 好友表操作 |

---

## DTO — 请求/响应结构体

定义在 `contract/dto.go`，所有 API 请求/响应均使用统一类型。

### 请求 DTO（20 个）

```go
// 滑块验证码
type VerifyCaptchaReq struct {
    CaptchaKey string `json:"captcha_key" binding:"required"`
    SlideX     int    `json:"slide_x"     binding:"required"`
    SlideY     int    `json:"slide_y"     binding:"required"`
}

// 认证
type RegisterReq struct {
    Username string `json:"username" binding:"required,min=3,max=64"`
    Email    string `json:"email"    binding:"required,email,max=255"`
    Password string `json:"password" binding:"required,min=6,max=128"`
}

type LoginReq struct {
    Account       string `json:"account"        binding:"required"`
    Password      string `json:"password"       binding:"required"`
    CaptchaKey    string `json:"captcha_key"    binding:"required"`
}

type VerifyEmailReq struct {
    Email string `json:"email" binding:"required,email"`
    Code  string `json:"code"  binding:"required,len=6"`
}

type ResendCodeReq struct {
    Email string `json:"email" binding:"required,email"`
}

type RefreshTokenReq struct {
    RefreshToken string `json:"refresh_token" binding:"required"`
}

type UpdateProfileReq struct {
    Nickname string `json:"nickname" binding:"omitempty,max=64"`
    Bio      string `json:"bio"      binding:"omitempty,max=512"`
}

type UpdatePasswordReq struct {
    OldPassword string `json:"old_password" binding:"required,min=6,max=128"`
    NewPassword string `json:"new_password" binding:"required,min=6,max=128"`
}

type ForgotPasswordReq struct {
    Email string `json:"email" binding:"required,email,max=255"`
}

type ResetPasswordReq struct {
    Token       string `json:"token"        binding:"required"`
    NewPassword string `json:"new_password" binding:"required,min=6,max=128"`
}

// 房间
type CreateRoomReq struct {
    Title        string      `json:"title"         binding:"required,max=128"`
    EmulatorType string      `json:"emulator_type" binding:"required,oneof=nes gb dos"`
    RomID        uuid.UUID   `json:"rom_id"        binding:"required"`
    MaxPorts     int16       `json:"max_ports"     binding:"required,min=1,max=4"`
    InviteeIDs   []uuid.UUID `json:"invitee_ids"`
}

type AssignPortReq struct {
    RoomID uuid.UUID `json:"room_id" binding:"required"`
    UserID uuid.UUID `json:"user_id" binding:"required"`
    Port   int16     `json:"port"    binding:"required,min=0"`
}

type InviteToRoomReq struct {
    RoomID     uuid.UUID   `json:"room_id"     binding:"required"`
    InviteeIDs []uuid.UUID `json:"invitee_ids" binding:"required,min=1"`
}

type StartRoomReq struct {
    RoomID uuid.UUID `json:"room_id" binding:"required"`
}

type LeaveRoomReq struct {
    RoomID uuid.UUID `json:"room_id" binding:"required"`
}

// ROM
type UploadRomReq struct {
    Title string `form:"title" binding:"required"`
}

// 好友
type FriendAddReq struct {
    FriendID uuid.UUID `json:"friend_id" binding:"required"`
}

type FriendAcceptReq struct {
    FriendID uuid.UUID `json:"friend_id" binding:"required"`
}

type FriendRejectReq struct {
    FriendID uuid.UUID `json:"friend_id" binding:"required"`
}
```

### 响应 DTO（10 个）

```go
type CaptchaResp struct {
    CaptchaKey     string `json:"captcha_key"`
    MasterBgBase64 string `json:"master_bg_base64"`
    TileBase64     string `json:"tile_base64"`
    ThumbX         int    `json:"thumb_x"`
    ThumbY         int    `json:"thumb_y"`
    TileWidth      int    `json:"tile_width"`
    TileHeight     int    `json:"tile_height"`
}

type LoginResp struct {
    AccessToken  string `json:"access_token"`
    RefreshToken string `json:"refresh_token"`
    ExpiresIn    int64  `json:"expires_in"`
}

type TokenPair struct {
    AccessToken  string `json:"access_token"`
    RefreshToken string `json:"refresh_token"`
    ExpiresIn    int64  `json:"expires_in"`
}

// 好友列表项（JOIN User 表）
type FriendWithUser struct {
    ID         uuid.UUID  `json:"id"`
    UserID     uuid.UUID  `json:"user_id"`
    FriendID   uuid.UUID  `json:"friend_id"`
    Status     int16      `json:"status"`
    AcceptedAt *time.Time `json:"accepted_at"`
    CreatedAt  time.Time  `json:"created_at"`
    Username   string     `json:"username"`
    Nickname   *string    `json:"nickname"`
    Avatar     *string    `json:"avatar"`
}

type FriendListResp struct {
    Friends []FriendWithUser `json:"friends"`
}

// 用户搜索
type UserSearchItem struct {
    ID       uuid.UUID `json:"id"`
    Username string    `json:"username"`
    Nickname *string   `json:"nickname"`
    Avatar   *string   `json:"avatar"`
}

type UserSearchResp struct {
    Users []UserSearchItem `json:"users"`
}

// 待处理好友请求
type FriendPendingItem struct {
    ID        uuid.UUID `json:"id"`
    UserID    uuid.UUID `json:"user_id"`
    FriendID  uuid.UUID `json:"friend_id"`
    CreatedAt time.Time `json:"created_at"`
    Username  string    `json:"username"`
    Nickname  *string   `json:"nickname"`
    Avatar    *string   `json:"avatar"`
}

type FriendPendingListResp struct {
    Pending []FriendPendingItem `json:"pending"`
}

// 房间
type StartRoomResp struct {
    LivekitToken string `json:"livekit_token"` // LiveKit access token
    LivekitRoom  string `json:"livekit_room"`  // LiveKit 房间名（= room_id）
    LivekitUrl   string `json:"livekit_url"`   // LiveKit 服务端 WebSocket 地址
}

type LivekitTokenResp struct {
    LivekitToken string `json:"livekit_token,omitempty"` // LiveKit access token
    LivekitRoom  string `json:"livekit_room,omitempty"`  // LiveKit 房间名
    LivekitUrl   string `json:"livekit_url,omitempty"`   // LiveKit 服务端地址
    Waiting      bool   `json:"waiting"`                  // 游戏未开始则返回 true
}

const (
    AccessTokenTTL  = 24 * time.Hour
    RefreshTokenTTL = 7 * 24 * time.Hour
)
```

---

## Gorm Model

### 通用规则

- **主键**: `gorm:"type:uuid;primaryKey"` — UUIDv7，由应用层 `uuid.Must(uuid.NewV7())` 生成，不依赖 DB 默认值
- **时间**: `gorm:"type:timestamptz;not null;autoCreateTime"` — 创建时间；`autoUpdateTime` — 更新时间
- **枚举**: SMALLINT 类型，状态值见中文注释
- **JSON 标签**: 直接返回给前端的 model 必须加 `json:"snake_case"` 标签（与 DB 字段名一致）；敏感字段用 `json:"-"` 排除（如 `PasswordHash`、`UploaderID`）
- **仅内部使用的 model**（`EmailVerification`、`RefreshToken`、`Friend`、`RoomPlayer`）可不加 json 标签，因为它们通过 DTO 返回
- 显式声明 `TableName()`
- 不设外键约束（`DisableForeignKeyConstraintWhenMigrating: true`）

**Model 代码示例**（前端可见的 model）：

```go
type Room struct {
    ID           uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
    HostID       uuid.UUID  `gorm:"type:uuid;not null;index" json:"host_id"`
    Title        string     `gorm:"type:varchar(128);not null" json:"title"`
    Status       int16      `gorm:"type:smallint;not null;default:0;index" json:"status"`
    CreatedAt    time.Time  `gorm:"type:timestamptz;not null;autoCreateTime" json:"created_at"`
    UpdatedAt    time.Time  `gorm:"type:timestamptz;not null;autoUpdateTime" json:"updated_at"`
}
```

### 所有 Model 一览

| Struct | 表名 | 说明 | JSON 标签 |
|--------|------|------|-----------|
| `User` | users | 用户 | 有（`PasswordHash` 排除） |
| `EmailVerification` | email_verifications | 邮箱验证码 | 无需（内部使用） |
| `RefreshToken` | refresh_tokens | Refresh Token | 无需（内部使用） |
| `PasswordReset` | password_resets | 密码重置 token | 无需（内部使用） |
| `Friend` | friends | 好友关系 | 无需（通过 DTO 返回） |
| `Room` | rooms | 游戏房间 | 有 |
| `RoomPlayer` | room_players | 房间座位 | 无需（内部使用） |
| `Rom` | roms | ROM 库 | 有（敏感字段排除） |

---

## 分层编码规范

### Repo 层

```go
// repo/user_repo.go

type UserRepo struct {
    db *gorm.DB
}

func NewUserRepo(db *gorm.DB) *UserRepo {
    return &UserRepo{db: db}
}

func (r *UserRepo) Create(ctx context.Context, user *model.User) error {
    return r.db.WithContext(ctx).Create(user).Error
}

func (r *UserRepo) ByEmail(ctx context.Context, email string) (*model.User, error) {
    var user model.User
    err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return nil, nil
    }
    return &user, err
}
```

### Service 层

```go
// service/auth_service.go

type AuthService struct {
    userRepo     contract.UserRepo
    captchaCache contract.CaptchaCache
    jwtSecret    []byte
}

func NewAuthService(userRepo contract.UserRepo, captchaCache contract.CaptchaCache, jwtSecret []byte) *AuthService {
    return &AuthService{userRepo: userRepo, captchaCache: captchaCache, jwtSecret: jwtSecret}
}

func (s *AuthService) Register(ctx context.Context, req contract.RegisterReq) (*model.User, error) {
    existing, _ := s.userRepo.ByEmail(ctx, req.Email)
    if existing != nil {
        return nil, apperror.ErrUserExists
    }
    // ...
}
```

### Handler 层

```go
// handler/auth.go

type AuthHandler struct {
    svc contract.AuthService
}

func NewAuthHandler(svc contract.AuthService) *AuthHandler {
    return &AuthHandler{svc: svc}
}

func (h *AuthHandler) Register(c *gin.Context) {
    var req contract.RegisterReq
    if err := c.ShouldBindJSON(&req); err != nil {
        response.BadRequest(c, "参数错误: "+err.Error())
        return
    }
    user, err := h.svc.Register(c.Request.Context(), req)
    if err != nil {
        response.Error(c, err)
        return
    }
    response.Created(c, user)
}
```

---

## 统一响应封装

```go
// pkg/response/response.go

type Body struct {
    Code    int         `json:"code"`            // 0=成功，非0=错误码
    Message string      `json:"message"`
    Data    interface{} `json:"data,omitempty"`
}

func OK(c *gin.Context, data interface{})           // 200
func Created(c *gin.Context, data interface{})  // 201
func NoContent(c *gin.Context)                  // 204
func Error(c *gin.Context, err error)           // AppError → 对应状态码，否则 500
func BadRequest(c *gin.Context, msg string)     // 400
func Unauthorized(c *gin.Context, msg string)   // 401
```

---

## 错误处理

错误码按模块分段：0xxx=通用，1xxx=认证，2xxx=房间，3xxx=ROM，4xxx=好友/通用，5xxx=服务器。

```go
// pkg/apperror/error.go

type AppError struct {
    Code    int    // 业务错误码
    Message string
    HTTP    int    // HTTP 状态码
}

// 通用
var ErrBadRequest    = &AppError{4000, "请求参数错误", 400}
var ErrUnauthorized  = &AppError{4001, "未登录或 token 无效", 401}

// 认证模块
var ErrUserExists         = &AppError{1001, "用户名或邮箱已存在", 409}
var ErrInvalidCaptcha     = &AppError{1002, "验证码错误", 401}
var ErrCaptchaExpired     = &AppError{1003, "验证码已过期", 401}
var ErrInvalidCredentials = &AppError{1004, "用户名或密码错误", 401}
var ErrUserNotActive      = &AppError{1005, "账户未激活", 403}
var ErrTooManyAttempts    = &AppError{1006, "尝试次数过多", 429}
var ErrResendCooldown     = &AppError{1007, "发送过于频繁", 429}
var ErrInvalidCode        = &AppError{1008, "验证码错误或已过期", 400}
var ErrRefreshTokenExp    = &AppError{1009, "refresh token 无效或已过期", 401}
var ErrUserNotFound       = &AppError{1010, "用户不存在", 404}
var ErrCaptchaNotVerified = &AppError{1011, "验证码未验证", 401}
var ErrForbiddenAdmin     = &AppError{1014, "需要管理员权限", 403}
var ErrResetTokenInvalid = &AppError{1012, "重置链接无效或已过期", 400}
var ErrResetTokenUsed    = &AppError{1013, "该重置链接已被使用", 400}

// 房间模块
var ErrRoomNotExist  = &AppError{2001, "房间不存在", 404}
var ErrNotRoomHost   = &AppError{2002, "仅房主可执行此操作", 403}
var ErrPortOccupied  = &AppError{2003, "该手柄已被占用", 409}
var ErrRoomFull      = &AppError{2005, "房间已满", 409}
var ErrRoomNotWaiting = &AppError{2006, "房间不在等待状态", 400}
var ErrRoomClosed    = &AppError{2007, "房间已关闭", 410}
var ErrAlreadyInRoom = &AppError{2008, "你已经在该房间中", 409}
var ErrNotInRoom     = &AppError{2009, "你不在该房间中", 403}
var ErrNotFriend     = &AppError{2010, "只能邀请好友", 403}
var ErrPortInvalid   = &AppError{2011, "无效的手柄端口", 400}

// ROM 模块
var ErrRomNotExist      = &AppError{3001, "ROM 不存在", 404}
var ErrRomDuplicate     = &AppError{3002, "该 ROM 文件已上传", 409}
var ErrRomTooLarge      = &AppError{3003, "ROM 文件过大", 400}
var ErrRomInvalidFormat = &AppError{3004, "ROM 文件格式不正确", 400}

// 好友模块
var ErrFriendSelf          = &AppError{4001, "不能添加自己为好友", 400}
var ErrFriendExists        = &AppError{4002, "好友关系已存在", 409}
var ErrFriendNotFound      = &AppError{4003, "好友关系不存在", 404}
var ErrFriendAlreadyHandled = &AppError{4004, "该好友请求已被处理", 400}

// Worker
var ErrNoAvailableWorker  = &AppError{5001, "暂无可用游戏节点", 503}
var ErrWorkerUnavailable  = &AppError{5002, "游戏节点不可用", 502}

// 服务器
var ErrInternal = &AppError{5000, "服务器内部错误", 500}
```

---

## JWT 设计

- 算法: HS256
- Access Token: 24h，payload 含 `user_id` + `username`
- Refresh Token: 7d，SHA-256 hash 后存入 `refresh_tokens` 表，轮换制

```go
// pkg/jwt/jwt.go

type Claims struct {
    UserID   uuid.UUID `json:"user_id"`
    Username string    `json:"username"`
    jwt.RegisteredClaims
}

func Generate(userID uuid.UUID, username string, secret []byte, ttl time.Duration) (string, error)
func Parse(tokenStr string, secret []byte) (*Claims, error)
```

---

## 路由注册

```go
// router/router.go
func New(cfg *config.Config, h *Handlers) *gin.Engine {
    r := gin.New()
    r.Use(logging.GinLogger(slog.Default()))  // slog 请求日志
    r.Use(gin.Recovery())                      // panic 恢复
    r.Use(cors.New(...))

    api := r.Group("/api")
    {
        // 公开接口（无需登录）
        api.GET("/auth/captcha",          h.Auth.Captcha)
        api.POST("/auth/captcha/verify",  h.Auth.VerifyCaptcha)
        api.POST("/auth/register",        h.Auth.Register)
        api.POST("/auth/verify-email",    h.Auth.VerifyEmail)
        api.POST("/auth/login",           h.Auth.Login)
        api.POST("/auth/resend-code",     h.Auth.ResendCode)
        api.POST("/auth/refresh",         h.Auth.RefreshToken)
        api.POST("/auth/forgot-password", h.Auth.ForgotPassword)
        api.POST("/auth/reset-password",   h.Auth.ResetPassword)

        // 需登录的接口
        auth := api.Group("", JWTAuth(cfg.JWTSecret))
        {
            // 用户信息
            auth.GET("/auth/me",          h.Auth.Me)
            auth.PUT("/auth/profile",     h.Auth.UpdateProfile)
            auth.PUT("/auth/password",    h.Auth.UpdatePassword)

            // 用户搜索
            auth.GET("/users/search",     h.Auth.Search)

            // 好友
            auth.GET("/friends",          h.Friend.List)
            auth.GET("/friends/pending",  h.Friend.Pending)
            auth.POST("/friends/add",     h.Friend.Add)
            auth.POST("/friends/accept",  h.Friend.Accept)
            auth.POST("/friends/reject",  h.Friend.Reject)

            // 房间
            auth.GET("/rooms",            h.Room.List)
            auth.POST("/rooms/create",    h.Room.Create)
            auth.POST("/rooms/invite",    h.Room.InviteToRoom)
            auth.POST("/rooms/assign",    h.Room.AssignPort)
            auth.POST("/rooms/start",     h.Room.Start)
            auth.POST("/rooms/leave",     h.Room.Leave)

            // ROM
            auth.GET("/roms",             h.Rom.List)    // 返回：自有 ROM + 全部平台内置 ROM
            auth.POST("/roms/upload",     h.Rom.Upload)
            auth.PUT("/roms/:id",         h.Rom.Update)   // 仅可改自有非内置 ROM

            // 管理员：平台内置 ROM 管理（JWTAuth + AdminAuth 查库校验 is_admin）
            admin := auth.Group("/admin", AdminAuth(userRepo))
            {
                admin.GET("/roms",        h.Admin.ListBuiltin)
                admin.POST("/roms/upload", h.Admin.UploadBuiltin)
                admin.PUT("/roms/:id",    h.Admin.UpdateBuiltin)
                admin.DELETE("/roms/:id", h.Admin.DeleteBuiltin)
            }
        }

        // 文件代理（公开）
        api.GET("/files/*path",           h.Files.Proxy)
    }
    return r
}
```

### JWT 中间件

从 `Authorization: Bearer <token>` 提取 JWT，解析后将 `user_id` 和 `username` 注入 `gin.Context`。

### AdminAuth 中间件

`AdminAuth(userRepo)` 必须在 `JWTAuth` 之后使用：读取 `user_id`，查库校验 `users.is_admin`，非管理员返回 403（错误码 1014）。权限改动实时生效，无需重新登录（is_admin 不写入 JWT）。

---

## Gorm 配置（main.go）

```go
func main() {
    cfg := config.MustLoad()

    // 初始化 slog 日志
    slog.SetDefault(logging.MustNew(cfg))

    // Gorm Logger：开发模式(LOG_LEVEL=debug)输出所有 SQL，生产仅慢查询
    devMode := cfg.LogLevel == "debug"
    gormLog := logging.NewGormLogger(slog.Default(), devMode)

    db, err := gorm.Open(postgres.Open(cfg.DSN), &gorm.Config{
        DisableForeignKeyConstraintWhenMigrating: true,
        Logger: gormLog,
    })
    if err != nil {
        slog.Error("failed to connect database", "error", err)
        os.Exit(1)
    }
    rds := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
    workerRds := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr, DB: cfg.WorkerRedisDB})  // DB 1: Worker 注册数据

    db.AutoMigrate(
        &model.User{}, &model.EmailVerification{}, &model.RefreshToken{},
        &model.Friend{}, &model.Room{}, &model.RoomPlayer{},
        &model.RoomPlayer{}, &model.Rom{},
    )

    minioAdapter, _ := repo.NewMinioAdapter(...)

    // --- 依赖注入 ---
    userRepo    := repo.NewUserRepo(db)
    roomRepo    := repo.NewRoomRepo(db)
    romRepo     := repo.NewRomRepo(db)
    friendRepo  := repo.NewFriendRepo(db)
    // ... 其他 repo

    captchaCache   := cache.NewCaptcha(rds)
    roomState      := cache.NewRoomState(rds)
    limiterCache   := cache.NewLimiter(rds)
    workerRegistry := cache.NewWorkerRegistry(workerRds)  // Worker 注册中心（Redis DB 1）

    // Scheduler + gRPC Client
    scheduler     := scheduler.New()                        // 加权最低负载优先
    workerClient  := grpcclient.NewWorkerClient(cfg.WorkerGRPCTimeout)  // gRPC 连接池

    authSvc  := service.NewAuthService(
        userRepo, emailVerificationRepo, refreshTokenRepo,
        passwordResetRepo,
        captchaCache, slideCaptcha, limiterCache, cfg.JWTSecret,
        minioAdapter, cfg.MinioBucket,
        emailSender, cfg.FrontendBaseURL,
    )
    roomSvc   := service.NewRoomService(roomRepo, roomPlayerRepo, friendRepo,
                     roomState, romRepo, scheduler, workerRegistry, workerClient)
    romSvc    := service.NewRomService(romRepo, minioAdapter, cfg.MinioBucket)
    friendSvc := service.NewFriendService(friendRepo, userRepo)

    handlers := &router.Handlers{
        Auth:     handler.NewAuthHandler(authSvc),
        Room:     handler.NewRoomHandler(roomSvc),
        Rom:      handler.NewRomHandler(romSvc),
        Admin:    handler.NewAdminHandler(romSvc),   // 平台内置 ROM 管理，复用 RomService
        Friend:   handler.NewFriendHandler(friendSvc),
        Files:    handler.NewFileHandler(minioAdapter, cfg.MinioBucket),
        UserRepo: userRepo,                          // 供 AdminAuth 中间件查库校验 is_admin
    }

    r := router.New(cfg, handlers)
    slog.Info("control-plane starting", "addr", cfg.Addr)
    if err := r.Run(cfg.Addr); err != nil {
        slog.Error("server error", "error", err)
        os.Exit(1)
    }
}
```

---

## slog 日志方案

### 架构概览

使用 Go 标准库 `log/slog` 替代原生的 `log` 包，`internal/pkg/logging/` 提供统一日志方案：

```
log/slog (标准库)
     │
     ├── TextHandler / JSONHandler           ← 通过环境变量 LOG_JSON 切换
     │
     └── io.MultiWriter
           ├── os.Stdout                     ← 始终输出到控制台
           └── rotatelogs (按天轮转)          ← 写入 logs/cloudemu-YYYY-MM-DD.log
                  └── 保留 30 天自动清理
```

slog 通过 `slog.SetDefault()` 设为全局默认 Logger，所有代码通过 `slog.Info()` / `slog.Error()` 等方法输出日志。

### 文件说明

| 文件 | 职责 |
|------|------|
| `internal/pkg/logging/logger.go` | `MustNew(cfg)` 创建 Logger：解析日志级别、构造 MultiWriter、每日轮转文件 |
| `internal/pkg/logging/gin.go` | `GinLogger(logger)` 中间件，替代 `gin.Logger()` |
| `internal/pkg/logging/gorm.go` | `NewGormLogger(logger, devMode)` 实现 `gorm.io/gorm/logger.Interface` |

### 依赖

```
github.com/lestrrat-go/file-rotatelogs v2.4.0  # 按天轮转日志文件
```

### 配置项（`config.Config`）

| 字段 | 环境变量 | 默认值 | 说明 |
|------|----------|--------|------|
| `RedisDB` | `REDIS_DB` | `0` | Redis DB 编号（业务数据） |
| `WorkerRedisDB` | `WORKER_REDIS_DB` | `1` | Redis DB 编号（Worker 注册与调度专用） |
| `LogDir` | `LOG_DIR` | `"logs"` | 日志文件输出目录，启动时自动创建 |
| `LogLevel` | `LOG_LEVEL` | `"info"` | 日志级别：`debug` / `info` / `warn` / `error` |
| `LogJSON` | `LOG_JSON` | `false` | `true` 时输出 JSON 格式，默认 Text 格式（`key=value`） |
| `WorkerGRPCTimeout` | `WORKER_GRPC_TIMEOUT` | `5s` | Worker gRPC 调用超时，支持 "5s" / "10s" / "1m" 等格式 |
| `SMTPHost` | `SMTP_HOST` | — | SMTP 服务器地址（空则不发送邮件） |
| `SMTPPort` | `SMTP_PORT` | `587` | SMTP 端口号 |
| `SMTPUser` | `SMTP_USER` | — | SMTP 登录账号 |
| `SMTPPass` | `SMTP_PASS` | — | SMTP 密码或授权码 |
| `SMTPFrom` | `SMTP_FROM` | — | 发件人显示地址 |
| `SMTPUseTLS` | `SMTP_USE_TLS` | `true` | SMTP 是否使用 TLS |
| `FrontendBaseURL` | `FRONTEND_BASE_URL` | `http://localhost:5173` | 前端基础 URL，用于生成密码重置链接 |

### Worker 配置（`worker.Config`）

| 字段 | 环境变量 | 默认值 | 说明 |
|------|----------|--------|------|
| `Addr` | `WORKER_ADDR` | `:9090` | gRPC 监听地址 |
| `RedisAddr` | `REDIS_ADDR` | `localhost:6379` | Redis 地址 |
| `RedisPass` | `REDIS_PASS` | — | Redis 密码（可选） |
| `RedisDB` | `WORKER_REDIS_DB` | `1` | Redis DB 编号（Worker 注册专用） |
| `LogLevel` | `LOG_LEVEL` | `info` | 日志级别 |
| `LiveKitHost` | `LIVEKIT_HOST` | `http://localhost:7880` | LiveKit 服务地址 |
| `LiveKitAPIKey` | `LIVEKIT_API_KEY` | — | LiveKit API Key |
| `LiveKitAPISecret` | `LIVEKIT_API_SECRET` | — | LiveKit API Secret |
| `EmuRunnerPath` | `EMURUNNER_PATH` | `./emurunner` | EmuRunner 二进制路径 |

### Gin 集成

`router.New()` 中 `gin.Default()` 已替换为 `gin.New()` + 手动注册中间件：

```go
r := gin.New()
r.Use(logging.GinLogger(slog.Default()))  // slog 请求日志
r.Use(gin.Recovery())                      // panic 恢复
```

`GinLogger` 记录的字段：`method`、`path`、`status`、`latency`、`ip`、`user_id`（从 context 提取）。日志级别按 HTTP 状态码分级：

| status | 日志级别 |
|--------|---------|
| >= 500 | `ERROR` |
| >= 400 | `WARN` |
| 其他 | `INFO` |

### Gorm 集成

Gorm 通过 `gorm.Config.Logger` 注入自定义 Logger：

```go
devMode := cfg.LogLevel == "debug"
gormLog := logging.NewGormLogger(slog.Default(), devMode)

db, err := gorm.Open(postgres.Open(cfg.DSN), &gorm.Config{
    Logger: gormLog,
})
```

`GormLogger` 完整实现 `gorm.io/gorm/logger.Interface`：

| 日志类型 | 级别 | 条件 |
|----------|------|------|
| 所有 SQL | `INFO` | `devMode=true`（`LOG_LEVEL=debug`） |
| 慢查询 (>200ms) | `WARN` | 始终记录 |
| SQL 错误 | `ERROR` | 始终记录（`ErrRecordNotFound` 除外） |

`Trace` 方法记录的字段：`sql`、`latency`(ms)、`rows`。

### 使用示例

```go
import "log/slog"

// 结构化日志
slog.Info("user registered", "user_id", user.ID, "email", user.Email)
slog.Warn("rate limit triggered", "ip", clientIP, "count", 5)
slog.Error("failed to send email", "error", err, "to", email)

// 致命错误后退出
slog.Error("failed to connect database", "error", err)
os.Exit(1)
```

### 文件轮转策略

- 文件名格式：`cloudemu-%Y-%m-%d.log`（如 `cloudemu-2026-06-04.log`）
- 轮转间隔：每天（`WithRotationTime(24h)`）
- 保留期限：30 天（`WithMaxAge(30*24h)`），超时自动删除
- 位置：`{LogDir}/` 目录下（默认 `logs/`）

---

## 前端 Axios 封装

### client.ts — 核心设计

```
请求拦截 ──→ 自动注入 Authorization: Bearer <access_token>
    ↓
响应拦截 ──→ 401? → 排队刷新 token（isRefreshing 并发保护）
    │                  ├─ 成功 → 重试所有等待请求
    │                  └─ 失败 → 清除 token → 跳转 /login
    └──→ 非 401 → 正常返回/抛出
```

### API 模块一览

| 文件 | 函数 | 对应端点 |
|------|------|----------|
| `auth.ts` | captcha, verifyCaptcha, register, verifyEmail, login, resendCode, refresh, me, updateProfile, updatePassword, forgotPassword, resetPassword | `/auth/*` |
| `room.ts` | list, create, inviteToRoom, assignPort, start, leave | `/rooms/*` |
| `rom.ts` | list, upload | `/roms/*` |
| `friend.ts` | list, pending, search, add, accept, reject | `/friends/*`, `/users/search` |

### 前端路由

| 路径 | 视图 | 元信息 | 说明 |
|------|------|--------|------|
| `/login` | `LoginView.vue` | `guest: true` | 仅在未登录时可见 |
| `/register` | `RegisterView.vue` | `guest: true` | 仅在未登录时可见 |
| `/forgot-password` | `ForgotPasswordView.vue` | `guest: true` | 仅在未登录时可见 |
| `/reset-password` | `ResetPasswordView.vue` | `guest: true` | 仅在未登录时可见 |
| `/` | `LobbyView.vue` | `auth: true` | 需要登录的大厅首页 |
| `/profile` | `ProfileView.vue` | `auth: true` | 需要登录的个人设置 |

### Pinia Stores

| Store | 状态 | 操作 |
|-------|------|------|
| `useAuthStore` | `user`, `isLoggedIn` | `fetchUser()`, `login()`, `updateProfile()`, `updatePassword()`, `logout()` |
| `useFriendStore` | `friends`, `pendingList`, `pendingCount`, `searchResults`, `loading`, `searchLoading` | `fetchFriends()`, `fetchPending()`, `searchUsers()`, `addFriend()`, `acceptFriend()`, `rejectFriend()` |

### 组件概览

| 目录 | 组件 | 说明 |
|------|------|------|
| `auth/` | `SlideCaptcha.vue` | 滑块验证码（加载底图+拼图，回传 slide_x/slide_y） |
| `common/` | `AnimatedBackground.vue` | 全屏动态粒子背景 |
| `friend/` | `AddFriendDialog.vue` | 添加好友弹窗（搜索 → 发送请求） |
| `friend/` | `FriendList.vue` | 好友列表（含在线状态、房间邀请入口） |

### Composables

| 文件 | 导出 | 功能 |
|------|------|------|
| `useTheme.ts` | `useTheme()` | `isDark` 响应式变量 + 深色/浅色切换（localStorage 持久化） |

### 样式

| 文件 | 说明 |
|------|------|
| `tokens.css` | CSS 自定义属性（颜色、阴影、圆角等） |
| `naive-overrides.ts` | Naive UI 主题覆写：深色 "Midnight Arcade" + 浅色 "Morning Cartridge" |

### 类型定义

前端 `src/types/api.ts` 与后端 DTO/Model 类型一一对应：

| 前端类型 | 后端类型 | 说明 |
|-----------|----------|------|
| `User` | `model.User` | 用户信息 |
| `Rom` | `model.Rom` (精简) | ROM 列表项 |
| `Room` | `model.Room` | 房间信息 |
| `FriendWithUser` | `contract.FriendWithUser` | 好友列表项（含用户字段） |
| `FriendPendingItem` | `contract.FriendPendingItem` | 待处理好友请求 |
| `UserSearchItem` | `contract.UserSearchItem` | 用户搜索结果 |
| `LoginReq/Resp`, `RegisterReq` 等 | `contract.*Req/Resp` | 请求/响应 DTO |
| `UpdateProfileReq`, `UpdatePasswordReq` | `contract.*Req` | 更新资料 DTO |
| `VerifyCaptchaReq`, `FriendRejectReq` | `contract.*Req` | 新增 DTO |
| `UserStatus`, `RoomStatus`, `FriendStatus` 等 | SMALLINT 枚举 | 字面量联合类型（已含 rejected=3） |

---

## 编码约定

| 约定 | 规则 |
|------|------|
| Go 目录 | 全小写，`control-plane`、`emurunner` |
| Go 包名 | 与目录名一致，全小写，单数（`model` 不叫 `models`） |
| Go 接口 | 集中在 `contract/` 包，不在实现包内定义 interface |
| DB 表名 | `snake_case` 复数（`users`、`room_players`） |
| JSON tag | `snake_case`，与 DB 字段名一致 |
| API 路由 | `/api/` 前缀，REST 风格 |
| 主键 | UUIDv7，应用层 `uuid.Must(uuid.NewV7())` 生成 |
| handler 职责 | 只做参数绑定和响应序列化，不写业务逻辑 |
| service 职责 | 编排业务逻辑，管理事务边界 |
| repo 职责 | 只做数据存取，不包含任何业务判断 |
| cache 职责 | Redis 操作，Key 格式统一（如 `captcha:{key}`） |
| 错误处理 | 使用 `pkg/apperror` 预定义错误，service 返回，handler 转 HTTP |
| context | 每层透传 `context.Context`，作为第一个参数 |
| 指针 | 返回 `*Struct` 表示可 nil；`[]Struct` 不用指针 |
| import 别名 | 不使用别名，除非冲突（如 `gojwt`） |
| 注释 | 全中文注释：model 字段含义、interface 方法作用、service 业务流程 |
| 前端 文件名 | kebab-case（`room-card.vue`, `use-auth.ts`） |
| 前端 组件 | PascalCase 多词（`<RoomCard>`, `<PlayerSlot>`） |
| 前端 组合式函数 | `useXxx` 前缀 |
| 前端 Store | `useXxxStore` 命名 |
| 前端 包管理 | pnpm |
