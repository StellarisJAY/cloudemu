<script setup lang="ts">
import { ref } from 'vue'
import type { ButtonName } from '@/utils/keyMapping'

const props = defineProps<{
  side: 'left' | 'right'
  enabled: boolean
}>()

const emit = defineEmits<{
  buttonChange: [btn: ButtonName, pressed: boolean]
}>()

/** 触摸标识符 → 当前按下的按钮 */
const touchMap = new Map<number, ButtonName>()
/** 每个按钮被几个触摸点按下（ref-counting，防止多指按同一按钮时过早释放） */
const pressCount = new Map<ButtonName, number>()
/** 当前激活（正在按下）的按钮集合，用于视觉高亮 */
const activeButtons = ref(new Set<ButtonName>())
/** 容器 DOM 引用，用于 hit-test */
const rootEl = ref<HTMLElement>()

/** 根据触摸坐标查找命中的按钮（在容器内遍历 data-btn 元素进行 rect 碰撞检测） */
function findButtonByTouch(touch: Touch): ButtonName | null {
  if (!rootEl.value) return null
  const buttons = rootEl.value.querySelectorAll<HTMLElement>('[data-btn]')
  for (const btn of buttons) {
    const rect = btn.getBoundingClientRect()
    if (
      touch.clientX >= rect.left &&
      touch.clientX <= rect.right &&
      touch.clientY >= rect.top &&
      touch.clientY <= rect.bottom
    ) {
      return btn.dataset.btn as ButtonName
    }
  }
  return null
}

/** 按下按钮（ref-counting，首次按下时 emit） */
function press(btn: ButtonName) {
  const count = pressCount.get(btn) ?? 0
  if (count === 0) {
    emit('buttonChange', btn, true)
    const next = new Set(activeButtons.value)
    next.add(btn)
    activeButtons.value = next
  }
  pressCount.set(btn, count + 1)
}

/** 释放按钮（ref-counting，最后一个触摸点离开时才 emit） */
function release(btn: ButtonName) {
  const count = pressCount.get(btn)
  if (count == null || count <= 1) {
    emit('buttonChange', btn, false)
    pressCount.delete(btn)
    const next = new Set(activeButtons.value)
    next.delete(btn)
    activeButtons.value = next
  } else {
    pressCount.set(btn, count - 1)
  }
}

/** 释放当前触摸点按下的按钮，并清理 touchMap */
function releaseTouch(id: number) {
  const btn = touchMap.get(id)
  if (btn) {
    release(btn)
    touchMap.delete(id)
  }
}

/** 更新触摸点对应的按钮（touchstart / touchmove） */
function updateTouch(id: number, newBtn: ButtonName | null) {
  const oldBtn = touchMap.get(id)
  if (oldBtn === newBtn) return // 没变化
  if (oldBtn) release(oldBtn)
  if (newBtn) {
    press(newBtn)
    touchMap.set(id, newBtn)
  } else {
    touchMap.delete(id)
  }
}

function onTouchStart(e: TouchEvent) {
  if (!props.enabled) return
  e.preventDefault()
  for (let i = 0; i < e.changedTouches.length; i++) {
    const touch = e.changedTouches[i]
    if (!touch) continue
    const btn = findButtonByTouch(touch)
    updateTouch(touch.identifier, btn)
  }
}

function onTouchMove(e: TouchEvent) {
  if (!props.enabled) return
  e.preventDefault()
  for (let i = 0; i < e.changedTouches.length; i++) {
    const touch = e.changedTouches[i]
    if (!touch) continue
    const btn = findButtonByTouch(touch)
    updateTouch(touch.identifier, btn)
  }
}

function onTouchEnd(e: TouchEvent) {
  e.preventDefault()
  for (let i = 0; i < e.changedTouches.length; i++) {
    const touch = e.changedTouches[i]
    if (!touch) continue
    releaseTouch(touch.identifier)
  }
}

function onTouchCancel(e: TouchEvent) {
  for (let i = 0; i < e.changedTouches.length; i++) {
    const touch = e.changedTouches[i]
    if (!touch) continue
    releaseTouch(touch.identifier)
  }
}

/** 按钮是否处于激活态 */
function isActive(btn: ButtonName): boolean {
  return activeButtons.value.has(btn)
}
</script>

<template>
  <div
    ref="rootEl"
    class="virtual-gamepad"
    :class="[`virtual-gamepad--${side}`, { 'is-disabled': !enabled }]"
    @touchstart="onTouchStart"
    @touchmove="onTouchMove"
    @touchend="onTouchEnd"
    @touchcancel="onTouchCancel"
  >
    <!-- D-pad（左侧） -->
    <template v-if="side === 'left'">
      <div class="dpad-grid">
        <div class="dpad-row">
          <div class="dpad-cell" />
          <div
            class="dpad-btn"
            :class="{ 'is-pressed': isActive('Up') }"
            data-btn="Up"
          >▲</div>
          <div class="dpad-cell" />
        </div>
        <div class="dpad-row">
          <div
            class="dpad-btn"
            :class="{ 'is-pressed': isActive('Left') }"
            data-btn="Left"
          >◀</div>
          <div class="dpad-center" />
          <div
            class="dpad-btn"
            :class="{ 'is-pressed': isActive('Right') }"
            data-btn="Right"
          >▶</div>
        </div>
        <div class="dpad-row">
          <div class="dpad-cell" />
          <div
            class="dpad-btn"
            :class="{ 'is-pressed': isActive('Down') }"
            data-btn="Down"
          >▼</div>
          <div class="dpad-cell" />
        </div>
      </div>
    </template>

    <!-- 动作按键（右侧） -->
    <template v-else>
      <div class="action-layout">
        <div class="action-main-row">
          <div
            class="action-btn action-btn--b"
            :class="{ 'is-pressed': isActive('B') }"
            data-btn="B"
          >B</div>
          <div
            class="action-btn action-btn--a"
            :class="{ 'is-pressed': isActive('A') }"
            data-btn="A"
          >A</div>
        </div>
        <div class="action-fn-row">
          <div
            class="action-btn-fn"
            :class="{ 'is-pressed': isActive('Select') }"
            data-btn="Select"
          >SEL</div>
          <div
            class="action-btn-fn"
            :class="{ 'is-pressed': isActive('Start') }"
            data-btn="Start"
          >STA</div>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.virtual-gamepad {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  touch-action: manipulation;
  -webkit-user-select: none;
  user-select: none;
}

.virtual-gamepad.is-disabled {
  opacity: 0.3;
  pointer-events: none;
}

/* ── D-pad 十字网格 ── */
.dpad-grid {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
}

.dpad-row {
  display: flex;
  align-items: center;
  gap: 4px;
}

.dpad-cell {
  width: 40px;
  height: 40px;
}

.dpad-btn {
  width: 40px;
  height: 40px;
  border-radius: var(--radius-md);
  background: rgba(255, 255, 255, 0.08);
  border: 1px solid rgba(255, 255, 255, 0.12);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
  color: rgba(255, 255, 255, 0.55);
  transition: background 0.05s, transform 0.05s;
  touch-action: manipulation;
}

.dpad-btn.is-pressed {
  background: rgba(255, 255, 255, 0.2);
  border-color: rgba(255, 255, 255, 0.3);
  color: rgba(255, 255, 255, 0.85);
  transform: scale(0.92);
}

.dpad-center {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid rgba(255, 255, 255, 0.06);
}

/* ── 动作按键布局 ── */
.action-layout {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
}

.action-main-row {
  display: flex;
  gap: 12px;
  align-items: flex-end;
}

.action-btn {
  width: 52px;
  height: 52px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  font-weight: 700;
  border: 2px solid rgba(255, 255, 255, 0.15);
  transition: background 0.05s, transform 0.05s;
  touch-action: manipulation;
}

/* A 按钮：偏暖色调，位置稍上 */
.action-btn--a {
  background: rgba(220, 60, 60, 0.2);
  color: rgba(255, 120, 120, 0.8);
  margin-top: -8px;
}

.action-btn--a.is-pressed {
  background: rgba(220, 60, 60, 0.45);
  transform: scale(0.9);
}

/* B 按钮：偏冷色调 */
.action-btn--b {
  background: rgba(60, 120, 220, 0.2);
  color: rgba(120, 160, 255, 0.8);
}

.action-btn--b.is-pressed {
  background: rgba(60, 120, 220, 0.45);
  transform: scale(0.9);
}

/* Start / Select 功能键 */
.action-fn-row {
  display: flex;
  gap: 16px;
}

.action-btn-fn {
  min-width: 36px;
  height: 22px;
  padding: 0 6px;
  border-radius: 11px;
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.08);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 9px;
  font-weight: 700;
  letter-spacing: 1px;
  color: rgba(255, 255, 255, 0.35);
  transition: background 0.05s;
  touch-action: manipulation;
}

.action-btn-fn.is-pressed {
  background: rgba(255, 255, 255, 0.15);
  color: rgba(255, 255, 255, 0.7);
}
</style>
