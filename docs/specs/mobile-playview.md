# Spec: 移动端 PlayView 适配

## Objective

让休闲玩家在手机浏览器上加入 CloudEmu 房间并通过触摸虚拟手柄玩游戏。仅适配 PlayView（游戏页面），登录/大厅/注册保留桌面布局不变。

**用户故事:**
- 作为房主，我想邀请好友加入房间，好友在手机上打开链接后能直接看到游戏画面并用触摸手柄操作
- 作为手机玩家，我进入 `/play/:roomId` 后，在横屏下能看到游戏画面、虚拟 D-pad 和动作按钮，边看边玩
- 作为手机房主（非 MVP），我仍需要选择 ROM、管理成员、分配端口

**成功标准:**
- Pixel 7 (412×915) / iPhone 14 (390×844) 横屏下，游戏画面占屏幕 ≥ 55%
- 虚拟手柄多指触控 10 秒内无卡键（按 D-pad + A 同时操作不丢事件）
- 桌面端 `/play/:roomId` 布局和交互 zero diff
- `pnpm build` + `pnpm type-check` 通过，无新增 TS 错误

## Tech Stack

- Vue 3.5 / TypeScript 6.0 / Vite 8
- Naive UI 2.44（n-drawer、n-button、n-tag）
- @vueuse/core 14.3（useMediaQuery）
- LiveKit JS SDK 2.19（视频轨、DataChannel）
- pnpm（包管理）
- **零新依赖**

## Commands

```bash
# 开发
pnpm dev                      # 在 web/ 目录下启动 Vite dev server

# 构建
pnpm build                    # type-check + build

# 仅类型检查
pnpm type-check               # vue-tsc --build

# Lint
pnpm lint                     # oxlint + eslint --fix

# 格式化
pnpm format                   # prettier --write
```

## Project Structure

```
web/src/
├── views/
│   └── PlayView.vue                 # [改] 主游戏页，添加移动端分支
├── components/
│   └── play/
│       ├── GameScreen.vue           # [改] HUD 提示文案 + 视频样式微调
│       ├── GameToolbar.vue          # [不改] 桌面端保持原样
│       ├── MemberPanel.vue          # [不改] 桌面端保持原样
│       ├── KeyMappingDialog.vue     # [不改]
│       ├── MemberItem.vue           # [不改]
│       └── VirtualGamepad.vue       # [新] 虚拟手柄组件（D-pad + 动作键）
├── composables/
│   └── useGameInput.ts              # [改] 暴露 applyButton/releaseAll + 支持触控模式
├── styles/
│   ├── tokens.css                   # [改] 添加 viewport + 移动端基础 token
│   └── mobile.css                   # [新] 移动端专用 CSS（横屏强制、游戏手柄样式）
└── utils/
    └── keyMapping.ts                # [不改] 复用 BUTTON_BITS
```

## Code Style

遵循项目现有约定：

```typescript
// composable: 返回对象，函数式
export function useGameInput(publishInput, enabled) {
  let state = 0
  function applyButton(btn: ButtonName, pressed: boolean) { /* ... */ }
  function releaseAll() { state = 0 }
  return { applyButton, releaseAll, reloadMapping }
}
```

```vue
<!-- 组件: <script setup> + scoped CSS + CSS 变量 -->
<script setup lang="ts">
const props = defineProps<{ enabled: boolean }>()
</script>
<template>
  <div class="virtual-gamepad" :class="{ 'is-disabled': !enabled }">
    <!-- ... -->
  </div>
</template>
<style scoped>
.virtual-gamepad { /* 使用 --color-* 变量，不写死色值 */ }
</style>
```

- 全中文注释
- CSS 变量名：`--color-*` 来自 `tokens.css`
- TypeScript 严格模式，props 用 `defineProps<T>()`
- 事件用 `defineEmits<{ eventName: [payloadType] }>()`

## Testing Strategy

| 层级 | 工具 | 测试内容 |
|------|------|----------|
| 类型检查 | `pnpm type-check` (vue-tsc) | 所有 `.ts/.vue` 文件 |
| Lint | `pnpm lint` (oxlint + eslint) | 代码风格 |
| 手动真机 | Chrome DevTools Device Mode + 真机 | 布局、触控、视频流 |
| 回归 | 桌面端 Chrome `pnpm dev` | 确保桌面端 zero diff |

**本阶段不写单元测试**（1-2 周原型验证，触控事件和 WebRTC 不适配 jsdom/node 环境）。手动测试 checklist：
- [ ] 桌面端 PlayView 三栏布局完整显示
- [ ] Chrome DevTools 模拟 Pixel 7：竖屏显示旋转提示，横屏显示虚拟手柄
- [ ] 真机 Safari iOS / Chrome Android：多指同时按 D-pad + A，10 秒不卡键
- [ ] 真机视频流播放流畅，无明显掉帧（肉眼判断）

## Boundaries

### Always do
- 使用项目已有的 CSS 变量（`--color-*`），不引入新色值
- 中文注释
- `pnpm type-check` 通过后再提交代码
- 桌面端 PlayView 视觉效果和行为 zero diff

### Ask first
- 引入新 npm 依赖
- 修改 `useLiveKit.ts` 或 LiveKit 连接逻辑
- 改动后端 API 或输入协议格式

### Never do
- 删除或重命名现有 CSS class/文件
- 修改 `GameToolbar.vue` / `MemberPanel.vue` 的内部实现
- 硬编码色值或字号（始终用 `--*` token）

## Success Criteria

1. **布局**: 桌面端 `PlayView.vue` 三栏无变化；移动端横屏时显示三段式（120px 左虚拟手柄 + flex:1 视频 + 120px 右按键）
2. **竖屏**: 显示旋转提示遮罩，引导用户旋转手机
3. **虚拟手柄**: D-pad（上下左右）+ A/B/Start/Select 按钮可点击，支持多指同时操作
4. **输入协议**: 触控操作通过 `useGameInput.applyButton()` 进入 60Hz RAF → DataChannel，不改协议
5. **抽屉**: 移动端工具栏和成员面板通过 `n-drawer` 从左右滑出，桌面端保持原样
6. **构建**: `pnpm type-check` 和 `pnpm build` 通过

## Open Questions

- 微信内置浏览器对 WebRTC 视频流的支持是否需要特殊处理？
- 虚拟手柄 D-pad 和动作键的 120px 宽度在 iPhone SE(375×667) 上是否过窄？是否需要按屏幕宽度动态计算？
- `n-drawer` 在横屏 812px 宽度下的抽屉宽度默认值是否合适？是否需要改成 `placement="bottom"`？
