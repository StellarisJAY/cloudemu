# CloudEmu 前端架构

## 技术栈

| 层 | 选型 |
|----|------|
| 框架 | Vue 3 + TypeScript |
| 构建 | Vite |
| HTTP | Axios |
| 状态管理 | Pinia |
| 路由器 | Vue Router 4 |
| 组件库 | Naive UI |
| 实时通信 | REST + Polling（暂无 WebSocket） |
| 音视频 | LiveKit JS SDK |
| UI 风格 | 暗色主题，Steam 风格深蓝/亮蓝调 |

---

## 依赖关系

```
views ──→ components
  │           │
  ├──→ stores ←── api
  ├──→ router
  └──→ composables
         │
         └──→ api    (Axios)
         └──→ stores (Pinia)
```

| 层 | 职责 |
|----|------|
| `views` | 页面级组件，对应路由，编排子组件 |
| `components` | 可复用 UI 组件，无业务逻辑 |
| `composables` | 组合式函数，封装业务逻辑 / 第三方 SDK |
| `stores` | Pinia store，集中管理全局状态 |
| `api` | Axios 实例 + 接口函数，负责 HTTP 通信 |

---

## 目录结构

```
web/
├── index.html
├── vite.config.ts
├── tsconfig.json
├── package.json
│
├── public/
│   └── assets/
│       ├── default-cover-nes.png
│       └── default-cover-gba.png
│
└── src/
    ├── main.ts
    ├── App.vue
    │
    ├── api/
    │   ├── client.ts         # axios instance + 拦截器（token注入 + 401并发保护刷新）
    │   ├── auth.ts           # 注册/登录/验证码/刷新 token
    │   ├── room.ts           # 房间 CRUD + 邀请/分配/启动
    │   ├── rom.ts            # ROM 列表/上传
    │   └── friend.ts         # 好友添加/接受/列表
    │
    ├── stores/
    │   ├── auth.ts           # 用户信息、token、登录状态
    │   ├── room.ts           # 当前活跃房间、手柄映射
    │   └── rom.ts            # ROM 列表、上传进度
    │
    ├── router/
    │   └── index.ts          # 路由定义 + 守卫
    │
    ├── views/
    │   ├── LoginView.vue
    │   ├── RegisterView.vue
    │   ├── LobbyView.vue     # 主大厅：房间列表 + ROM 库
    │   ├── RoomView.vue      # 房间内：等待、分配手柄、邀请
    │   └── PlayView.vue      # 游戏画面：LiveKit 视频 + 手柄输入
    │
    ├── components/
    │   ├── auth/
    │   │   ├── CaptchaInput.vue
    │   │   └── EmailVerify.vue
    │   ├── room/
    │   │   ├── RoomCard.vue
    │   │   ├── RoomList.vue
    │   │   ├── PlayerSlot.vue
    │   │   ├── PlayerList.vue
    │   │   └── CreateRoomDialog.vue
    │   ├── rom/
    │   │   ├── RomCard.vue
    │   │   ├── RomList.vue
    │   │   └── RomUploadDialog.vue
    │   ├── friend/
    │   │   ├── FriendList.vue
    │   │   └── AddFriendDialog.vue
    │   └── common/
    │       ├── AppHeader.vue
    │       └── GamepadVisual.vue
    │
    ├── composables/
    │   ├── useAuth.ts
    │   ├── useRoom.ts
    │   ├── useLiveKit.ts
    │   └── useRoms.ts
    │
    ├── types/
    │   └── api.ts            # 全部 API 请求/响应类型（与后端 DTO 一一对应）
    │
    └── utils/
        └── token.ts          # localStorage token 读写（getAccessToken/setTokens/clearTokens）
```

---

## 页面路由

| 路径 | 视图 | 权限 | 说明 |
|------|------|------|------|
| `/login` | LoginView | 公开 | 图形验证码 + 账号密码登录 |
| `/register` | RegisterView | 公开 | 邮箱注册 + 验证码激活 |
| `/` | LobbyView | 需登录 | 主大厅：查看房间 + ROM 库 + 好友 |
| `/profile` | ProfileView | 需登录 | 个人信息：修改头像、昵称、简介、密码 |
| `/room/:id` | RoomView | 需登录 | 房间等待室：分配手柄、邀请好友 |
| `/play/:roomId` | PlayView | 需登录 | 游戏画面：LiveKit 视频流 + 手柄输入 |

---

## 风格 & 主题

基于 Naive UI 暗色主题 + 自定义 `themeOverrides`，模拟 Steam 风格：

```ts
// src/main.ts
import { createApp } from 'vue'
import { createPinia } from 'pinia'
import naive from 'naive-ui'
import App from './App.vue'

const app = createApp(App)
app.use(createPinia())
app.use(naive, {
  themeOverrides: {
    common: {
      primaryColor: '#4fc3f7',
      primaryColorHover: '#81d4fa',
      bodyColor: '#1b2838',
      cardColor: '#16202d',
      modalColor: '#171d25',
      textColorBase: '#c7d5e0',
      textColor1: '#ffffff',
      borderRadius: '6px',
    },
  },
})
app.mount('#app')
```

```vue
<!-- App.vue -->
<template>
  <n-config-provider :theme="darkTheme" :theme-overrides="themeOverrides">
    <n-loading-bar-provider>
      <n-notification-provider>
        <n-message-provider>
          <router-view />
        </n-message-provider>
      </n-notification-provider>
    </n-loading-bar-provider>
  </n-config-provider>
</template>
```

---

## Axios 封装

```ts
// src/api/client.ts
import axios from 'axios'
import { getAccessToken, getRefreshToken, setTokens, clearTokens } from '@/utils/token'

const client = axios.create({
  baseURL: import.meta.env.VITE_API_BASE || '/api',
  timeout: 15000,
})

// 标记是否正在刷新 token，避免并发 401 重复刷新
let isRefreshing = false
let refreshSubscribers: Array<(token: string) => void> = []

function subscribeTokenRefresh(cb: (token: string) => void) {
  refreshSubscribers.push(cb)
}
function onTokenRefreshed(newToken: string) {
  refreshSubscribers.forEach((cb) => cb(newToken))
  refreshSubscribers = []
}

// 请求拦截 — 自动注入 token
client.interceptors.request.use((config) => {
  const token = getAccessToken()
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

// 响应拦截 — 401 时尝试刷新 token（并发安全）
client.interceptors.response.use(
  (res) => res,
  async (error) => {
    const originalRequest = error.config
    if (error.response?.status !== 401 || originalRequest._retry) {
      return Promise.reject(error)
    }

    const refreshToken = getRefreshToken()
    if (!refreshToken) {
      clearTokens()
      window.location.href = '/login'
      return Promise.reject(error)
    }

    // 正在刷新中 → 排队等待
    if (isRefreshing) {
      return new Promise((resolve) => {
        subscribeTokenRefresh((newToken: string) => {
          originalRequest.headers.Authorization = `Bearer ${newToken}`
          resolve(client(originalRequest))
        })
      })
    }

    isRefreshing = true
    originalRequest._retry = true

    try {
      const { data } = await axios.post('/api/auth/refresh', { refresh_token: refreshToken })
      if (data.code !== 0 || !data.data) throw new Error('refresh failed')
      const { access_token, refresh_token } = data.data
      setTokens(access_token, refresh_token)
      onTokenRefreshed(access_token)
      originalRequest.headers.Authorization = `Bearer ${access_token}`
      return client(originalRequest)
    } catch {
      clearTokens()
      window.location.href = '/login'
      return Promise.reject(error)
    } finally {
      isRefreshing = false
    }
  },
)

export default client
```

### API 接口函数示例

类型定义统一在 `src/types/api.ts`，与后端 DTO 一一对应。

```ts
// src/api/auth.ts
import client from './client'
import type { ApiResponse, CaptchaResp, LoginReq, LoginResp } from '@/types/api'

export const authApi = {
  captcha() {
    return client.get<ApiResponse<CaptchaResp>>('/auth/captcha')
  },
  login(data: LoginReq) {
    return client.post<ApiResponse<LoginResp>>('/auth/login', data)
  },
  register(data: RegisterReq) {
    return client.post<ApiResponse<User>>('/auth/register', data)
  },
  refresh(token: string) {
    return client.post<ApiResponse<TokenPair>>('/auth/refresh', { refresh_token: token })
  },
  me() {
    return client.get<ApiResponse<User>>('/auth/me')
  },
}
```

---

## Pinia Store 设计

### auth store

```ts
// src/stores/auth.ts
export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const isLoggedIn = computed(() => user.value !== null)

  async function login(account: string, password: string) { /* ... */ }
  async function logout() { /* 清除 token + user */ }
  async function refreshToken() { /* 调用 authApi.refresh */ }

  return { user, isLoggedIn, login, logout, refreshToken }
})
```

### room store

```ts
// src/stores/room.ts
export const useRoomStore = defineStore('room', () => {
  const rooms = ref<Room[]>([])              // 用户参与的房间列表
  const loading = ref(false)

  async function fetchRooms() { /* GET /rooms */ }
  async function createRoom(req: CreateRoomReq) { /* POST /rooms/create */ }
  async function inviteToRoom(req: InviteToRoomReq) { /* POST /rooms/invite — 好友直接加入 */ }

  return { rooms, loading, fetchRooms, createRoom, inviteToRoom }
})
```

### rom store

```ts
// src/stores/rom.ts
export const useRomStore = defineStore('rom', () => {
  const roms = ref<Rom[]>([])
  const uploadProgress = ref(0)

  async function fetchRoms() { /* GET /roms */ }
  async function uploadRom(file: File, cover?: File, title: string) { /* POST FormData */ }

  return { roms, uploadProgress, fetchRoms, uploadRom }
})
```

---

## Composables 封装

```ts
// src/composables/useLiveKit.ts
import { Room, RemoteTrackPublication } from 'livekit-client'

export function useLiveKit() {
  const room = ref<Room | null>(null)
  const videoTrack = ref<RemoteTrackPublication | null>(null)

  async function connectRoom(token: string, liveKitUrl: string) {
    const r = new Room()
    await r.connect(liveKitUrl, token)
    room.value = r
    // 订阅远程用户的音视频轨道
    r.on('trackPublished', (pub) => { /* ... */ })
  }

  function sendInput(inputData: ArrayBuffer) {
    // 通过 DataChannel 发送手柄输入
    room.value?.localParticipant.publishData(inputData)
  }

  return { room, videoTrack, connectRoom, sendInput }
}
```

---

## 页面布局概览

### LobbyView — 主大厅

```
┌────────────────────────────────────────────────┐
│  AppHeader  (Logo | 导航 | 搜索 | 用户菜单)     │
├────────┬───────────────────────────────────────┤
│        │                                       │
│ 侧边栏  │    ROM 封面网格 / 房间列表             │
│ 好友列表 │                                      │
│ 在线状态 │                                       │
│        │                                       │
|────────│───────────────────────────────────────|
│  Footbar  (状态栏：当前房间、通知)               │
└────────┴───────────────────────────────────────┘
```

### RoomView — 房间等待室 (完善版)

```
┌────────────────────────────────────────────────┐
│  ← 返回   房间名   模拟器类型   状态标签  离开按钮 │
├──────────────────┬─────────────────────────────┤
│                  │                             │
│    成员列表       │   操作面板（仅房主可见）       │
│                  │                             │
│   房主 (Port 0)  │   [邀请好友]                  │
│   玩家1 (Port 1) │   [开始游戏]                  │
│   旁观者...      │                              │
│                  │                             │
└──────────────────┴─────────────────────────────┘
```

### PlayView — 游戏画面

```
┌────────────────────────────────────────────────┐
│  ← 返回   房间名   直播状态   网络延迟   设置    │
├────────────────────────────────────────────────┤
│                                                │
│            ┌────────────────────┐              │
│            │                    │              │
│            │   LiveKit 视频流    │              │
│            │   (游戏画面渲染)    │              │
│            │                    │              │
│            └────────────────────┘              │
│                                                │
│  手柄输入提示 / 快捷键 / 手柄状态              │
└────────────────────────────────────────────────┘
```

---

## 关键交互流程

### 登录 → 进入大厅

```
LoginView                    auth store                  router
    │                           │                         │
    ├─ 输入账号密码+验证码 ──────┤                         │
    │                           ├─ POST /auth/login       │
    │                           │← { access_token, user } │
    │                           ├─ 存入 localStorage      │
    │                           ├─ 更新 user              │
    │                           └─────────────────────── router.push('/')
```

### 创建房间 + 好友直接加入

```
LobbyView / CreateRoomDialog
    │
    ├─ 选好友（多选）→ 提交
    │
    room store → POST /rooms/create { invitee_ids: [...] }
    │              ← { room_id }
    │              backend 校验好友关系后直接加入（无需接受）
    │
    router.push(`/room/${room_id}`)
    ↓
RoomView
    ├─ 房主：分配手柄 → 邀请更多好友 → 开始游戏
    └─ 其他成员：看到房间，等待分配手柄或旁观
```

### 房主邀请好友到已有房间

```
RoomView (房主点击邀请)
    │
    ├─ 选择好友 → POST /rooms/invite { room_id, invitee_ids }
    │              ← 200 OK（好友直接成为房间成员）
    │
    ↓ 好友刷新房间列表即可看到
```

### 进入游戏

```
RoomView (房主点击开始)
    │
    POST /rooms/start { room_id }
    │← Backend 返回 LiveKit token + 手柄映射
    │
    useLiveKit().connectRoom(token, liveKitUrl)
    │
    router.push(`/play/${room_id}`)
    ↓
PlayView
    ├─ videoTrack → <video> 展示游戏画面
    ├─ DataChannel → 发送手柄输入 (ArrayBuffer)
    └─ 收到离开指令 / 掉线 → 返回大厅
```

---

## 编码约定

| 约定 | 规则 |
|------|------|
| 文件名 | kebab-case（`room-card.vue`, `use-auth.ts`） |
| 组件命名 | PascalCase 多词（`<RoomCard>`, `<PlayerSlot>`） |
| 组合式函数 | `useXxx` 前缀 |
| Store | `useXxxStore` 命名 |
| Style | `<style scoped>`，不写全局样式（除 App.vue） |
| 类型 | 接口定义在 `types/`，API 响应类型放在对应 `api/*.ts` |
| API 函数 | 统一 `client.get/post` 调用，不直接 axios |
| 错误处理 | 在 composable / store 中 try-catch，UI 层用 Naive UI 的 `useMessage` 展示 |
