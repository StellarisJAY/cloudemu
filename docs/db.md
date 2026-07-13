# CloudEmu 数据库设计

## 存储策略

| 数据类型 | 存储 | 理由 |
|---------|------|------|
| 用户数据、验证记录 | PostgreSQL | 持久化，需可追溯 |
| 图形验证码 | Redis DB 0 + TTL | 纯临时，一次性使用，TTL 自动清理 |
| 频率限制（防爆破） | Redis DB 0 + TTL | 临时计数器，自动过期 |
| 会话元数据、端口锁、存档暂存 | Redis DB 0 | 读写频繁，TTL 管理 |
| Worker 注册与调度 | Redis DB 1 | 数据隔离，防止 Worker 数据与业务缓存混淆 |

---

## PostgreSQL 表设计

### 命名规范
- snake_case
- 主键 UUIDv7（应用层 `uuid.Must(uuid.NewV7())` 生成，不依赖 DB 默认值）
- 不使用数据库外键约束，应用层保证一致性
- 时间类型使用 `TIMESTAMPTZ`

---

### 1. `users` — 用户

```sql
CREATE TABLE users (
    id            UUID PRIMARY KEY,
    username      VARCHAR(64)  NOT NULL,
    email         VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    nickname      VARCHAR(64),              -- 昵称，展示用，空时前端展示 username
    avatar        VARCHAR(512),             -- 头像URL/路径，空时前端展示默认头像
    bio           VARCHAR(512),             -- 个人简介
    status        SMALLINT     NOT NULL DEFAULT 0,  -- 0:pending, 1:active, 2:disabled
    last_login_at TIMESTAMPTZ,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX uk_users_username ON users (username);
CREATE UNIQUE INDEX uk_users_email    ON users (email);
CREATE INDEX        ix_users_status   ON users (status);
```

| 列 | 类型 | 说明 |
|----|------|------|
| id | UUID | 主键，应用层 UUIDv7 生成 |
| username | VARCHAR(64) | 唯一，登录凭据之一 |
| email | VARCHAR(255) | 唯一，注册必填，接收验证码 |
| password_hash | VARCHAR(255) | bcrypt 哈希 |
| nickname | VARCHAR(64) | 昵称，展示用，NULL 时前端展示 username |
| avatar | VARCHAR(512) | 头像 URL/路径，NULL 时前端展示默认头像 |
| bio | VARCHAR(512) | 个人简介 |
| status | SMALLINT | 0=待激活, 1=已激活(可游戏), 2=已禁用 |
| last_login_at | TIMESTAMPTZ | 最近一次登录时间 |
| created_at | TIMESTAMPTZ | 注册时间 |
| updated_at | TIMESTAMPTZ | 更新时间 |

#### 状态机

```
注册 → status=0 (pending)
  → 邮箱验证成功 → status=1 (active)  → 可登录/游戏
  → 管理员禁用   → status=2 (disabled) → 不可登录/游戏
```

---

### 2. `email_verifications` — 邮件验证码

```sql
CREATE TABLE email_verifications (
    id                UUID PRIMARY KEY,
    user_id           UUID         NOT NULL,
    email             VARCHAR(255) NOT NULL,
    verification_code VARCHAR(8)   NOT NULL,
    expires_at        TIMESTAMPTZ  NOT NULL,
    verified_at       TIMESTAMPTZ,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX ix_ev_user_id ON email_verifications (user_id);
CREATE INDEX ix_ev_email   ON email_verifications (email);
```

| 列 | 类型 | 说明 |
|----|------|------|
| id | UUID | 主键 |
| user_id | UUID | 逻辑关联 users.id |
| email | VARCHAR(255) | 发送到的邮箱，验证时查询用 |
| verification_code | VARCHAR(8) | 6 位随机数字/字母 |
| expires_at | TIMESTAMPTZ | 有效期（15 分钟） |
| verified_at | TIMESTAMPTZ | NULL=未验证, 非NULL=已验证时间 |
| created_at | TIMESTAMPTZ | 发送时间 |

#### 清理

```sql
-- 定期清理过期的未验证记录（定时任务，如每小时）
DELETE FROM email_verifications
WHERE expires_at < NOW() - INTERVAL '1 hour'
  AND verified_at IS NULL;
```

---

### 3. `refresh_tokens` — 刷新令牌

```sql
CREATE TABLE refresh_tokens (
    id         UUID PRIMARY KEY,
    user_id    UUID         NOT NULL,
    token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMPTZ  NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX ix_rt_user_id ON refresh_tokens (user_id);
CREATE UNIQUE INDEX uk_rt_token_hash ON refresh_tokens (token_hash);
```

| 列 | 类型 | 说明 |
|----|------|------|
| id | UUID | 主键 |
| user_id | UUID | 逻辑关联 users.id |
| token_hash | VARCHAR(255) | Refresh Token 的 SHA-256 哈希 |
| expires_at | TIMESTAMPTZ | 过期时间（7 天） |

> 每次刷新 token 时，旧记录作废（DELETE），插入新记录。

---

### 4. `password_resets` — 密码重置 token

```sql
CREATE TABLE password_resets (
    id         UUID PRIMARY KEY,
    user_id    UUID         NOT NULL,
    email      VARCHAR(255) NOT NULL,
    token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMPTZ  NOT NULL,
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX ix_pr_user_id    ON password_resets (user_id);
CREATE INDEX ix_pr_email      ON password_resets (email);
CREATE UNIQUE INDEX uk_pr_token_hash ON password_resets (token_hash);
```

| 列 | 类型 | 说明 |
|----|------|------|
| id | UUID | 主键 |
| user_id | UUID | 逻辑关联 users.id |
| email | VARCHAR(255) | 冗余邮箱，加速查询 |
| token_hash | VARCHAR(255) | 重置 token 的 SHA-256 哈希 |
| expires_at | TIMESTAMPTZ | 有效期（15 分钟） |
| used_at | TIMESTAMPTZ | NULL=未使用, 非NULL=已使用时间 |

> 与 `refresh_tokens` 设计一致：原始 token 不落库，仅存 SHA-256 哈希。token 通过邮件发送给用户（MVP 阶段暂不实际发送邮件）。

---

## Redis Key 设计

### DB 分配

Redis 通过不同 database index 隔离数据，避免 key 冲突：

| DB | 用途 | 使用者 | Key 前缀 |
|----|------|--------|----------|
| 0 | 业务数据（验证码、频率限制、房间状态、手柄锁） | Control Plane | `captcha:` `login_attempt:` `verify_attempt:` `resend_cooldown:` `room:` `port_lock:` |
| 1 | Worker 注册与调度数据 | Control Plane + WorkerAgent | `worker:` |

### 图形验证码

| Key | Value | TTL | 说明 |
|-----|-------|-----|------|
| `captcha:{captcha_key}` | 验证码答案 | 300s (5 min) | 登录验证，验证成功后 DEL |

**流程**:

```
GET  /api/auth/captcha
  → 生成随机 captcha_key (UUID格式) + answer (4-6位字母数字)
  → Redis SET captcha:{key} {answer} EX 300
  → 根据 answer 绘制图片（Go image 库）
  → 返回 { captcha_key, image_base64 }

POST /api/auth/login  {account, password, captcha_key, captcha_answer}
  → Redis GET captcha:{key}
      ├─ 不存在 → "验证码已过期"
      └─ 存在 → 比较 answer (忽略大小写)
            ├─ 错误 → "验证码错误"
            └─ 正确 → DEL captcha:{key} → 继续校验账号密码
```

### 频率限制（防爆破）

| Key | Value | TTL | 说明 |
|-----|-------|-----|------|
| `login_attempt:{ip}` | 失败次数 (INCR) | 900s (15 min) | 5 次错误锁定 15 分钟 |
| `verify_attempt:{email}` | 尝试次数 (INCR) | 300s (5 min) | 3 次失败暂时禁止 |
| `resend_cooldown:{email}` | `1` (SET) | 60s (1 min) | 60 秒内禁止重复发送 |
| `forgot_pwd:{ip}` | 失败次数 (INCR) | 300s (5 min) | 3 次错误锁定 5 分钟 |
| `forgot_cooldown:{email}` | `1` (SET) | 60s (1 min) | 60 秒内禁止重复发送 |

---

## 认证流程

### 用户注册

```
1. POST /api/auth/register  {email, username, password}
2. 校验 username / email 不重复
3. bcrypt 加密密码
4. INSERT users (status=0)
5. 生成 6 位验证码
6. INSERT email_verifications
7. 异步发送邮件
8. 返回 "请查收验证码邮件"
```

### 邮箱激活

```
1. POST /api/auth/verify-email  {email, code}
2. SELECT * FROM email_verifications
   WHERE email=? ORDER BY created_at DESC LIMIT 1
3. 校验 code 匹配 && NOW() < expires_at && verified_at IS NULL
4. UPDATE users SET status=1 WHERE id=?
5. UPDATE email_verifications SET verified_at=NOW() WHERE id=?
6. 返回 "激活成功"
```

### 登录

```
1. GET /api/auth/captcha → 返回 captcha_key + image_base64

2. POST /api/auth/login  {account, password, captcha_key, captcha_answer}
   a. Redis: GET captcha:{captcha_key}
       → 不存在/不匹配 → 返回错误
       → 匹配 → DEL captcha:{captcha_key}
   b. 查 users WHERE (username=? OR email=?) AND status=1
   c. bcrypt.Compare(password_hash, password)
   d. 生成 JWT (Access Token + Refresh Token)
   e. INSERT refresh_tokens (token_hash, user_id, expires_at)
   f. UPDATE last_login_at
   g. 返回 { access_token, refresh_token, expires_in, user: { id, username, email } }
```

### 刷新 Token

```
1. POST /api/auth/refresh  {refresh_token}
2. 对 refresh_token 做 SHA-256 得到 token_hash
3. 查 refresh_tokens WHERE token_hash=? AND expires_at > NOW()
4. 校验通过 → DELETE 旧记录 → 生成新 token pair → INSERT 新 refresh_token
5. 返回 { access_token, refresh_token, expires_in }
```

### 获取当前用户信息

```
1. GET /api/auth/me
   (JWT 中间件解析出 user_id)
2. SELECT * FROM users WHERE id=? AND status=1
3. 返回 { id, username, email, created_at }
```

### 重发验证码

```
1. POST /api/auth/resend-code  {email}
2. Redis: 检查 resend_cooldown:{email} → 冷却中则拒绝
3. 查 users WHERE email=? AND status=0
4. 使旧的 email_verifications 作废（或标记无效）
5. 新验证码 → INSERT email_verifications
6. 异步发送邮件
7. Redis: SET resend_cooldown:{email} 1 EX 60
```

### 忘记密码

```
1. POST /api/auth/forgot-password  {email}
2. 查 users WHERE email=? AND status=1
   → 不存在或未激活 → 统一返回成功（防止用户枚举）
3. Redis: 检查 forgot_cooldown:{email} → 冷却中则拒绝
4. Redis: 检查 forgot_pwd:{ip} 频率限制 → 超限则拒绝
5. 生成 32 字节随机 token → SHA-256 → token_hash
6. INSERT password_resets (token_hash, user_id, email, expires_at=NOW()+15min)
7. 异步发送密码重置邮件（包含重置链接 {FRONTEND_BASE_URL}/reset-password?token=xxx）
8. Redis: SET forgot_cooldown:{email} 1 EX 60
9. 返回成功
```

### 密码重置

```
1. POST /api/auth/reset-password  {token, new_password}
2. SHA-256(token) → hash
3. SELECT * FROM password_resets WHERE token_hash=?
4. 校验: record 存在 AND used_at IS NULL AND NOW() < expires_at
5. bcrypt 加密 new_password
6. UPDATE users SET password_hash=? WHERE id=record.user_id
7. UPDATE password_resets SET used_at=NOW() WHERE id=record.id
8. 返回成功
```

---

## 房间 + ROM 系统 — 表设计

### 隐私规则

| 实体 | 可见性 |
|------|--------|
| ROM | **仅上传者自己可见**，其他用户无法浏览/搜索 |
| 房间列表 | 仅显示**自己创建或已加入**的房间，无公开大厅 |
| 好友 | 双向确认后可见，未确认不可见 |

### 核心关系

```
                    ┌──────────────┐
                    │   friends    │  用户双向好友关系
                    └──────┬───────┘
                           │
    ┌──────────┐    ┌──────▼───────┐
    │   roms   │◄───│    rooms     │
    └──────────┘    └──────┬───────┘
                           │
                    ┌──────▼───────┐
                    │ room_players │  玩家座位 + 手柄映射
                    └──────────────┘
```

---

### 3. `friends` — 好友关系

```sql
CREATE TABLE friends (
    id            UUID PRIMARY KEY,
    user_id       UUID        NOT NULL,
    friend_id     UUID        NOT NULL,
    status        SMALLINT    NOT NULL DEFAULT 0,  -- 0:pending, 1:accepted, 2:blocked, 3:rejected
    accepted_at   TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX uk_friends_pair    ON friends (LEAST(user_id, friend_id), GREATEST(user_id, friend_id));
CREATE INDEX        ix_friends_user    ON friends (user_id);
CREATE INDEX        ix_friends_friend  ON friends (friend_id);
CREATE INDEX        ix_friends_status  ON friends (status);
```

| 列 | 类型 | 说明 |
|----|------|------|
| user_id | UUID | 发起方 |
| friend_id | UUID | 接收方 |
| status | SMALLINT | 0=待接受, 1=好友, 2=已拉黑, 3=已拒绝 |

> **uk_friends_pair**: 用 `LEAST/GREATEST` 保证 (A,B) 和 (B,A) 视为同一对，重复好友申请不插入。

---

### 5. `rooms` — 游戏房间

```sql
CREATE TABLE rooms (
    id             UUID PRIMARY KEY,
    host_id        UUID         NOT NULL,
    title          VARCHAR(128) NOT NULL,
    emulator_type  VARCHAR(32)  NOT NULL,           -- 'nes', 'gba', 'dos'
    rom_id         UUID,                            -- 关联 ROM，NULL=房主尚未选择 ROM
    max_ports      SMALLINT     NOT NULL DEFAULT 4, -- 最大手柄端口数
    status         SMALLINT     NOT NULL DEFAULT 0, -- 0:waiting, 1:playing, 2:closed
    started_at     TIMESTAMPTZ,
    closed_at      TIMESTAMPTZ,
    worker_addr    VARCHAR(64),                     -- 分配到的 Worker gRPC 地址（用于 StopGame）
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX ix_rooms_host    ON rooms (host_id);
CREATE INDEX ix_rooms_status  ON rooms (status);
CREATE INDEX ix_rooms_rom     ON rooms (rom_id);
CREATE INDEX ix_rooms_emu     ON rooms (emulator_type);
```

| 列 | 类型 | 说明 |
|----|------|------|
| host_id | UUID | 房主，只有房主能启动/分配手柄 |
| title | VARCHAR(128) | 房间名（房主自定义） |
| emulator_type | VARCHAR(32) | 决定了可用的 libretro 核心 |
| rom_id | UUID | 逻辑关联 roms.id；NULL=房主尚未选择 ROM（必须选择后才能开始游戏） |
| max_ports | SMALLINT | 最大4人 |
| status | SMALLINT | 0=等待中, 1=游戏中, 2=已关闭 |
| worker_addr | VARCHAR(64) | 游戏启动时记录分配的 Worker 地址，用于 StopGame 时知道发往哪个 Worker |

```
房间状态流:
创建 → status=0 (waiting)
  → 房主启动游戏 → status=1 (playing)
  → 房主关闭 / 所有人离开 → status=2 (closed)
```

**查询规则**: 用户只能看到 `WHERE host_id=? OR id IN (SELECT room_id FROM room_players WHERE user_id=? AND left_at IS NULL)`。

---

### 6. `room_players` — 房间座位

```sql
CREATE TABLE room_players (
    id          UUID PRIMARY KEY,
    room_id     UUID        NOT NULL,
    user_id     UUID        NOT NULL,
    role        SMALLINT    NOT NULL DEFAULT 2,  -- 0:host, 1:player, 2:spectator
    port        SMALLINT,                        -- NULL=旁观, 0-based 手柄端口号
    joined_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    left_at     TIMESTAMPTZ
);

CREATE UNIQUE INDEX uk_rp_active ON room_players (room_id, user_id) WHERE left_at IS NULL;
CREATE INDEX        ix_rp_room    ON room_players (room_id);
CREATE INDEX        ix_rp_user    ON room_players (user_id);
```

| 列 | 类型 | 说明 |
|----|------|------|
| role | SMALLINT | 0=房主, 1=操作者(有手柄), 2=旁观者 |
| port | SMALLINT | NULL=旁观, 非NULL=映射到模拟器第 N 号手柄 |
| left_at | TIMESTAMPTZ | NULL=活跃中, 非NULL=已离开 |

角色与手柄的关系:

| role | port | 表现 |
|------|------|------|
| 0 (host) | 非NULL | 房主 + 操作者，默认 Port0 |
| 1 (player) | 非NULL | 分配到手柄的玩家，向 EmuRunner 发送输入 |
| 2 (spectator) | NULL | 只能看不能操作 |

---

### 7. `roms` — ROM 库（用户私有）

```sql
CREATE TABLE roms (
    id            UUID PRIMARY KEY,
    uploader_id   UUID         NOT NULL,
    title         VARCHAR(255) NOT NULL,
    file_name     VARCHAR(255) NOT NULL,
    emulator_type VARCHAR(32)  NOT NULL,           -- 'nes', 'gba', 'dos'
    file_size     BIGINT       NOT NULL,
    sha256        VARCHAR(64)  NOT NULL,
    status        SMALLINT     NOT NULL DEFAULT 0, -- 0:pending, 1:approved, 2:rejected
    minio_path    VARCHAR(512) NOT NULL,
    cover_path    VARCHAR(512),                    -- MinIO 封面图路径, NULL 则前端使用默认图
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX ix_roms_uploader ON roms (uploader_id);
CREATE INDEX ix_roms_status   ON roms (status);
CREATE INDEX ix_roms_emu      ON roms (emulator_type);
CREATE INDEX ix_roms_sha256   ON roms (sha256);
```

| 列 | 类型 | 说明 |
|----|------|------|
| uploader_id | UUID | 上传者，**查询时强制过滤 `WHERE uploader_id=?`** |
| sha256 | VARCHAR(64) | 文件去重（同一用户不可重复上传同一文件） |
| minio_path | VARCHAR(512) | MinIO 上的 ROM 文件存储路径 |
| cover_path | VARCHAR(512) | MinIO 上的封面图路径，NULL 则前端按 emulator_type 显示默认图 |
| status | SMALLINT | 0=待审核, 1=已通过(可用), 2=已拒绝 |

> **ROM 隔离**: `roms` 没有全局唯一索引。即使两个用户上传了同一个文件（相同 sha256），也会存储为两条独立记录，各自通过 `uploader_id` 隔离。`sha256` 索引仅用于同一用户去重。
>
> **封面图处理**: `cover_path` 为 NULL 时，前端根据 `emulator_type` 展示默认封面（如 NES 灰色卡带、GBA 白色卡带图标、DOS 深绿色磁盘图标）。覆盖图由用户在 ROM 上传时选择文件一起提交，存储到 MinIO 的 `cover/{uploader_id}/{rom_id}.{ext}` 路径。

---

## Redis — 房间实时状态

| Key | 类型 | 说明 |
|-----|------|------|
| `room:{room_id}:state` | Hash | `{status, emulator_type, rom_id, host_id}` |
| `room:{room_id}:ports` | Hash | `{0: user_id, 1: user_id, ...}` 手柄实时分配 |
| `room:{room_id}:livekit` | Hash | `{url, room}` LiveKit 连接信息（token 由 Worker 按用户独立生成，不再缓存） |
| `room:{room_id}:spectators` | Set | 旁观者 user_id 集合 |
| `port_lock:{room_id}:{port}` | String | 手柄抢占锁（SET NX），防止并发分配冲突 |

> 房间关闭时全部清理。

---

## Redis — Worker 注册与调度

> **所有 key 存储在 Redis DB 1**，与业务数据（DB 0）隔离。

| Key | 类型 | TTL | 说明 |
|-----|------|-----|------|
| `worker:{worker_id}` | String (JSON) | 30s | Worker 心跳数据，TTL 过期即视为宕机 |
| `worker:{worker_id}:lock` | String | 5s | Worker 调度锁（SET NX），防止并发分配给同一 Worker |

### worker:{worker_id} JSON 结构

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
| `addr` | Worker gRPC 地址 |
| `weight` | 调度权重（CPU核心数 × 30） |
| `sessions` | 当前运行会话数 |
| `max_sessions` | 最大会话数（= weight） |
| `cpu_percent` | CPU 使用率（仅监控，不参与调度） |
| `mem_percent` | 内存使用率（仅监控，不参与调度） |
| `started_at` | Worker 启动时间 |

### 心跳流程

```
Worker: 每 15s 执行 SET worker:{id} {json} EX 30
Control Plane: SCAN worker:* → 读取所有存活 Worker 用于调度
```

### 调度流程

```
1. SCAN worker:* → 所有存活 Worker
2. 过滤: sessions < max_sessions
3. 排序: score = sessions / weight 升序（加权最低负载优先）
4. 选中后: INCR worker:{id}:sessions（通过 Lua 脚本原子操作 JSON.sessions++）
5. 调度失败: DECR sessions → fallback 到下一个 Worker
```

---

## 房间功能 — 业务流程

### 添加好友

```
1. A POST /api/friends/add {friend_id: B}
2. INSERT friends (A, B, status=0)
3. B 收到好友通知（实时通知机制留待后续设计，见 docs/deferred.md）
4. B POST /api/friends/accept {friend_id: A}
5. UPDATE friends SET status=1, accepted_at=NOW()
```

### 创建房间（勾选好友一起玩）

```
1. 房主 POST /api/rooms/create
   {title, emulator_type, rom_id, max_ports, invitee_ids: [B, C]}

2. INSERT rooms (host_id, title, emulator_type, rom_id, max_ports)
3. INSERT room_players (room_id, host, role=0, port=0)  ← 房主默认 Port0

4. 对每个 invitee_id（校验必须为已接受好友）:
   INSERT room_players (room_id, user_id=invitee, role=2, port=NULL)  ← 好友自动加入，默认旁观，无需接受

5. 返回 room_id
```

### 房主邀请好友加入已有房间（新）

```
1. 房主 POST /api/rooms/invite
   {room_id, invitee_ids: [D]}

2. 校验:
   - 请求者 = 房主
   - 房间存在
   - 每个 invitee 是房主的好友（status=1）
   - 好友未已在房间中

3. 对每个 invitee:
   INSERT room_players (room_id, user_id=invitee, role=2, port=NULL)

4. 好友直接出现在房间中（类似微信拉群，无需接受）
```

### 房主分配手柄

```
1. 房主 POST /api/rooms/assign-port
   {room_id, user_id, port}

2. 校验:
   - 请求者 = host
   - user 在房间内且 left_at IS NULL
   - port ∈ [0, max_ports)

3. 如果 port 已被占 → 原占有者降为 role=2, port=NULL
4. 目标用户: role=1, port=指定值
5. Redis: HSET room:{room_id}:ports {port} {user_id}
6. 如果在游戏中 → 通知 EmuRunner 重新映射输入源
```

### 开始游戏

```
1. 房主 POST /api/rooms/start {room_id}
2. Control Plane:
   a. 查 DB 获取 rom.MinioPath
   b. Scheduler 选择最优 Worker
   c. 生成 MinIO 预签名 ROM 下载 URL（5 分钟有效）
   d. gRPC 调用 Worker.StartGame(host_user_id, rom_url, rom_path, emulator_type)
3. Worker:
   a. LiveKit SDK 创建房间
   b. 生成两个独立 token：
      - EmuRunner token（identity="emurunner", canPublish=true）
      - 房主 player token（identity="player:{host_id}", canPublish=false）
   c. 从 rom_url 下载 ROM → 启动 EmuRunner（传入 EmuRunner token）
   d. 返回 {emu_token, host_token, livekit_room, livekit_url} 给 Control Plane
4. Control Plane:
   a. UPDATE rooms SET status=1, started_at=NOW()
   b. Redis: HSET room:{room_id}:livekit {url, room}（不存 token）
   c. 向前端返回 {livekit_token: host_token, livekit_room, livekit_url}
5. 非房主轮询 GET /api/rooms/:id/livekit → CP 调 Worker GeneratePlayerToken(room, user_id)
   → 每人获得独立 token（identity="player:{user_id}"）
6. 前端用 livekit_url + 自己的 livekit_token 建立 WebRTC 连接
7. 有手柄的玩家 → 打开 DataChannel 发送输入
   无手柄 → 仅接收音视频（旁观）
```

### 离开房间

```
1. 玩家 POST /api/rooms/leave {room_id}
2. UPDATE room_players SET left_at=NOW() WHERE room_id=? AND user_id=?
3. Redis: HDEL room:{room_id}:ports {port}

4. 如果离开的是房主:
   - 选下一个 role=1 的玩家为新 host
   - 如果没有 → 关闭房间
```

### 查看自己的房间列表

```
GET /api/rooms  → 查当前用户参与的活跃房间:
  SELECT * FROM rooms
  WHERE status != 2
    AND (host_id = ?  OR  id IN (
      SELECT room_id FROM room_players
      WHERE user_id=? AND left_at IS NULL
    ))
  ORDER BY created_at DESC
```

### 查看自己的 ROM 列表

```
GET /api/roms  → 查当前用户的 ROM:
  SELECT * FROM roms
  WHERE uploader_id=? AND status=1
  ORDER BY created_at DESC

返回示例:
{
  "roms": [{
    "id": "uuid",
    "title": "Contra",
    "emulator_type": "nes",
    "cover_url": "/api/files/cover/xxx.jpg",   ← cover_path 非空则返回可访问 URL
    "cover_default": false                      ← NULL 时前端使用默认封面
  }]
}
```

**前端默认封面映射**:

| emulator_type | 默认封面 |
|---------------|---------|
| `nes` | `/assets/default-cover-nes.png` |
| `gba` | `/assets/default-cover-gba.png` |
| `dos` | `/assets/default-cover-dos.png` |

### 上传 ROM 带封面的请求

```
POST /api/roms/upload
Content-Type: multipart/form-data
Body:
  rom:   <file>             (必填)
  cover: <file>             (选填, 图片文件)
  title: "Contra"

→ 后台: 封面存储到 MinIO cover/{uploader_id}/{rom_id}.{ext}
→ cover_path = "cover/{uploader_id}/{rom_id}.jpg"
→ 若未传 cover 文件 → cover_path = NULL
```
