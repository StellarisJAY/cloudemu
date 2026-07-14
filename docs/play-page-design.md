# 游玩页面（PlayView）设计方案

## 页面布局

```
┌──────────────────────────────────────────────────────────────────────────┐
│ 顶栏: ← 离开游戏  |  房间名称 [状态]                     |  NES/GB       │
├──────────────┬──────────────────────────────┬────────────────────────────┤
│ 左栏 260px   │  中栏 flex:1                  │  右栏 260px                │
│ 游戏工具栏   │  视频播放器                    │  成员列表                  │
│              │  (LiveKit Video Stream)      │                            │
│ [房主]       │                              │  👑 房主 (蓝)              │
│  ROM 选择    │   16:9 画面                  │  🎮 玩家 (绿)              │
│  模拟器状态  │   letterbox 黑边              │  👁️ 旁观 (黄)              │
│  游戏控制    │                              │                            │
│  🚀 开始游戏 │  无流时显示占位封面            │  [邀请好友]                │
│ ──────────  │                              │                            │
│ [成员]       │                              │                            │
│  游戏状态    │                              │                            │
│  🔌 连接    │                              │                            │
│ ──────────  │                              │                            │
│ [所有人]     │                              │                            │
│  🎮 按键映射 │                              │                            │
└──────────────┴──────────────────────────────┴────────────────────────────┘
```

## 组件树

```
PlayView.vue                     (路由: /play/:roomId)
├── GameToolbar.vue              (左栏, 260px)
│   ├── 房主: RomSelector       (ROM 选择 n-select)
│   ├── 房主: EmulatorStatus    (FPS/CPU/状态标签)
│   ├── 房主: GameControls      (开始游戏/暂停/继续/停止/存档)
│   ├── 成员: ConnectionArea    (等待提示/连接按钮/已连接)
│   ├── 提示                    (成员: ROM+控制由房主操作)
│   └── 所有人: KeyMappingButton → KeyMappingDialog
├── GameScreen.vue              (中栏, flex:1)
│   └── LiveKit Video | 占位封面
└── MemberPanel.vue             (右栏, 260px)
    ├── MemberItem.vue ×N       (成员行: 头像+名称+角色+操作)
    └── n-modal: InviteDialog
```

## 连接状态机（成员侧）

### 状态定义

| 状态 | 值 | 含义 | UI |
|------|-----|------|-----|
| `waiting` | room.status=0 | 游戏未开始 | 黄色脉冲圆点 + "等待房主开始游戏..." |
| `ready` | room.status=1 | 游戏已开始，可连接 | 蓝色 "🔌 连接" 按钮 |
| `connecting` | 点击连接后 | 正在连接 LiveKit | 按钮 loading 态 + "连接中..." |
| `connected` | 连接成功 | 已连上推流 | 绿色 "✅ 已连接" + 延迟显示 |
| `error` | 连接失败 | 网络/服务异常 | 红色 "连接失败，点击重试" 按钮 |

### 状态流转

```
成员进入页面
   │
   ├─ room.status === 0 ──→ waiting ──(轮询 3s)──→ room.status === 1 ──→ ready
   └─ room.status === 1 ──→ ready
                               │
                         点击 [🔌 连接]
                               │
                         connecting ──→ connected (成功)
                               │
                               └──→ error (失败) ──点击──→ connecting (重试)
```

### 轮询策略

- 成员进入页面时立即检查一次房间状态
- 每 3 秒调用 `roomStore.fetchRooms()` 刷新
- 检测到 `room.status === 1` 时，`connectionState` 变为 `ready`
- 已连接（`connected`）后停止轮询
- 房主不启动轮询
- 页面卸载（`onUnmounted`）时清除定时器

## 房主 / 成员 工具栏差异

| 模块 | 房主 | 成员 | 说明 |
|------|:----:|:----:|------|
| ROM 选择 | ✅ | ❌ | n-select + 当前 ROM 标签 |
| 模拟器状态 | ✅ | ❌ | FPS / CPU / 状态标签 + 进度条 |
| 🚀 开始游戏 | ✅ | ❌ | `type=primary`，ROM 加载后显示 |
| ⏯ 暂停 | ✅ | ❌ | 运行状态下显式 |
| ▶ 继续 | ✅ | ❌ | 暂停状态下显式 |
| ⏹ 停止 | ✅ | ❌ | 运行/暂停状态下显式 |
| 💾 存档 | ✅ | ❌ | 运行状态下显式 |
| 🔌 连接 | ❌ | ✅ | 见连接状态机 |
| 等待提示 | ❌ | ✅ | 脉冲圆点 + 文案 |
| 🎮 按键映射 | ✅ | ✅ | 所有人可设置键盘映射 |

## 连接接口（LiveKit Token）

```
POST /api/rooms/:id/connect
  - 需登录
  - 仅当 room.status === 1 且用户是房间成员
  - 返回 { livekit_token, livekit_ws_url }

错误响应:
  200 { code: 1, message: "游戏尚未开始" }  // room.status !== 1
  403                                    // 非房间成员
```

该接口即通用的连接接口（成员获取 LiveKit token）。

## 文件清单

| 文件 | 作用 |
|------|------|
| `web/src/views/PlayView.vue` | 页面容器，管理 isHost / connectionState / 轮询 / 事件分发 |
| `web/src/components/play/GameToolbar.vue` | 左栏，根据 isHost 渲染不同模块 |
| `web/src/components/play/GameScreen.vue` | 中栏，视频区 + 占位封面 + HUD |
| `web/src/components/play/MemberPanel.vue` | 右栏，成员列表 + 邀请弹窗 |
| `web/src/components/play/MemberItem.vue` | 成员行，房主可切换角色/踢出 |
| `web/src/components/play/KeyMappingDialog.vue` | 按键映射弹窗，点击捕获键盘按键 |
| `web/src/router/index.ts` | 路由 /play/:roomId |
| (后端) `POST /api/rooms/:id/connect` | 直播连接接口，返回 LiveKit token |
| (后端) `GET /api/rooms/:id` | 房间详情（用于轮询），也可复用 `GET /api/rooms` |

## 待接入后端时需完成

1. `handleStartGame()` — 调用 `roomApi.start()`
2. `handleConnect()` — 调用 `POST /api/rooms/:id/connect`，获取 LiveKit token 并连接
3. 移除 mock 数据（成员列表、模拟器状态模拟）
4. 真实 FPS / CPU 数据通过 Worker 上报
5. GameScreen 集成 LiveKit JS SDK 播放视频流
