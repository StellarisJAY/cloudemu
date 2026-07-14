# 移动端 PlayView 适配

## Problem Statement

**How might we** 让休闲玩家在手机浏览器上流畅连入 CloudEmu 房间、用触摸虚拟手柄操作游戏，核心体验不输桌面端，且不开发原生 App？

---

## Recommended Direction

**三段式强制横屏 + 抽屉式全功能 + 多指触控虚拟手柄。**

### 页面布局

```
┌──────────────────────────────────────────────────┐ 横屏（orientation: landscape）
│ [⚙ 抽屉] ←  ←→  [👥 抽屉]                        │
│                                                    │
│   ┌─────────┐  ┌──────────────┐  ┌─────────┐      │
│   │ ▲       │  │              │  │    B    │      │
│   │◀ ● ▶    │  │    VIDEO     │  │  A      │      │
│   │ ▼       │  │  (contain)   │  │         │      │
│   │         │  │              │  │Sel Start│      │
│   └─────────┘  └──────────────┘  └─────────┘      │
│   ← 120px →    ← flex: 1 →       ← 120px →        │
└──────────────────────────────────────────────────┘
```

- **左段 (120px)**: D-pad 方向键（上/下/左/右），透明/半透明叠加
- **中段 (flex:1)**: `<video>` 全高 contain，复用现有 `GameScreen.vue` 的视频轨和占位逻辑
- **右段 (120px)**: A / B / Start / Select 按键
- **工具栏 + 成员面板**: Naive UI `n-drawer` 从左右边缘滑出，保持功能完整性
- **竖屏检测**: `@media (orientation: portrait)` 显示旋转提示遮罩

### 输入架构

```
touchstart/touchmove/touchend (multi-touch, identifier 追踪)
        │
        ▼
  VirtualGamepad.vue         ← 新增组件，管理触摸区域 → ButtonName 映射
  触摸状态机 (Map<ButtonName, boolean>)
        │
        ▼
  useGameInput.ts 模式       ← 复用现有 applyButton(btn, pressed) + 60Hz RAF
        │
        ▼
  LiveKit DataChannel        ← 现有 4-byte 二进制协议，零改动
```

核心改造点：
- `useGameInput.ts` 暴露 `applyButton()` 和 `releaseAll()` 供外部组件调用（`90:138` 已有 `applyButton` 内部函数，需提升为返回值）
- 新增 `VirtualGamepad.vue`：管理触摸区域定义、`touch.identifier` 多指追踪、ButtonName 状态机
- `GameScreen.vue` 底部 HUD 文案从 "ESC 打开菜单" 改为移动端菜单按钮入口

### 为什么这样设计

1. **三段式而非叠加式**：NES/GB 模拟器核心用户已经习惯 Delta 风格的左右分区。触摸叠加会遮挡画面像素，对于需要精确定位的平台游戏不可接受。
2. **全功能抽屉而非裁减**：房主在移动端仍需要选 ROM、管理成员、分配端口。裁掉这些功能意味着手机用户无法独立主持房间。
3. **复用输入协议不改动**：`useGameInput.ts` 的 60Hz RAF + 4-byte 协议是已有且经过测试的通路，虚拟手柄只需把触控状态映射到同一个 `ButtonName` bitset。

---

## Key Assumptions to Validate

### 不成立就死

- [ ] **WebRTC 视频流在移动端 60fps 流畅** — 用真机测试 LiveKit 远端视频轨在以下浏览器上的帧率和延迟：Safari iOS、Chrome Android、微信内置浏览器。**验证方式**: `useLiveKit.ts:23` 视频轨连接后，在移动端测试 3 款 NES/GB 游戏的实际帧率感知。
- [ ] **多指触控能可靠追踪 2+ 指同时操作** — `touchstart/touchmove/touchend` 的 `touch.identifier` 在快速切换时是否丢事件？"按住右 + 按 A" 同时操作时，手指从一个 D-pad 方向滑到另一个时状态转换是否正确？**验证方式**: 写一个独立 touch-debug 页面，实时渲染所有 touch point 位置和 identifier，测试极限操作（方向快速切换 + 按键连打）。
- [ ] **虚拟 D-pad 玩 NES 平台跳跃可接受** — 无物理反馈的触摸方向键在需要精准跳跃的游戏中（如马里奥、洛克人），用户是否在 5 分钟内放弃？**验证方式**: 内部试玩 3 款 NES 游戏各 10 分钟，记录主观体验和实际通关率。

### 重要但不致命

- [ ] **Naive UI n-drawer 在横屏移动端体验合格** — 滑动手势与虚拟手柄的 touch 事件不冲突，抽屉宽度在横屏 812px 宽度上不局促。**验证方式**: 在 PlayView 中集成 n-drawer 后用真机测试。
- [ ] **Screen Orientation API 可用** — iOS Safari 对 `screen.orientation.lock('landscape')` 的支持程度，以及微信内置浏览器的表现。**验证方式**: 真机测试，若不支持则用 CSS `@media` + 旋转提示作为 fallback。

### 锦上添花（MVP 不做）

- [ ] 用户愿意在移动端管理房间（选 ROM、踢人）— 验证了游戏可玩后再做功能完整性
- [ ] 触觉反馈（`navigator.vibrate`）的价值 — 先做无震动的版本

---

## MVP Scope（1-2 周）

### 必须做

| 项 | 文件 | 工作量 | 说明 |
|----|------|--------|------|
| 移动端检测 + 布局切换 | `PlayView.vue` | 0.5d | `matchMedia('(max-width: 768px)')` 或 UA 检测，桌面端布局 zero diff |
| 三段式横屏 CSS | `styles/mobile.css` (新) | 1d | 左中右 flex 布局、强制横屏检测、竖屏提示遮罩、响应式断点 |
| 虚拟手柄组件 | `VirtualGamepad.vue` (新) | 3d | 触摸区域定义、`touch.identifier` 多指追踪、ButtonName 状态机、视觉样式 |
| useGameInput 触控适配 | `useGameInput.ts` | 0.5d | 暴露 `applyButton()` / `releaseAll()`，支持外部触控调用 |
| 工具栏 + 成员面板抽屉化 | `PlayView.vue` + 子组件 | 1d | 桌面端保持原样，移动端改为 n-drawer 触发入口 |
| 真机兼容性测试 | — | 1d | Safari iOS / Chrome Android / 微信内置浏览器，修复兼容问题 |

**合计: ~7 工作日**，核心风险在虚拟手柄的 touch 多指状态机。

### 不做（本次 MVP）

- 虚拟手柄按键自定义布局（拖拽调整按钮位置）
- 触觉反馈（`navigator.vibrate`）
- 手柄皮肤系统
- 大厅、登录、注册、个人设置页面的移动端适配
- Canvas 绘制虚拟按钮（初期用纯 CSS div，更快迭代）
- 蓝牙手柄 Web Gamepad API 支持

---

## Not Doing (and Why)

- **大厅/登录页移动端适配** — 用户明确范围仅 PlayView。这些页面移动端的价值远低于游戏页，且可后续复用同样的响应式策略。
- **"手机即手柄"模式（Jackbox 风格）** — 用户需要的是独立游戏体验，不是第二屏幕控制器。这个方向偏离核心需求。
- **PWA / Service Worker / 离线包** — 1-2 周时间不够，且 PWA 的安装提示、离线策略与当前"快速验证核心假设"目标冲突。
- **手机端上传 ROM** — ROM 文件几十 MB，移动端上传不现实。房主用桌面端管理 ROM 库。
- **虚拟手柄的按键自定义拖拽** — 先固定布局验证可行性，布局定制是优化项。
- **Canvas WebGL 渲染虚拟手柄** — CSS div 足够快速出原型，性能瓶颈不在按钮渲染而在 WebRTC 视频解码。

---

## Open Questions

- 微信内置浏览器对 WebRTC / Screen Orientation / multi-touch 的支持程度如何？是否需要针对微信做特殊 polyfill？
- 横屏三段式在 iPhone SE (375×667 竖屏 → 667×375 横屏) 这种小屏上，120px 的 D-pad/按钮区是否太窄？是否需要按屏幕宽度动态缩放按钮区比例？
- 用户从桌面端 URL（如 `/play/:roomId`）直接分享到微信，手机打开后默认竖屏——CSS 竖屏提示能否有效引导用户旋转？还是需要强制 JS 调用 `screen.orientation.lock()`？
