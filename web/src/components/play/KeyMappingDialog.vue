<script setup lang="ts">
import { ref, reactive, onMounted, onUnmounted } from 'vue'
import {
  DEFAULT_KEY_MAPPING,
  loadMapping,
  saveMapping,
  type ButtonName,
} from '@/utils/keyMapping'

defineProps<{
  show: boolean
}>()

const emit = defineEmits<{
  close: []
  save: [mapping: Record<ButtonName, string>]
}>()

// 当前按键映射（键名 → KeyboardEvent.code，如 "KeyZ" / "ArrowUp"）
const mapping = reactive<Record<ButtonName, string>>({ ...loadMapping() })

const listeningKey = ref<ButtonName | null>(null)

function startListen(button: ButtonName) {
  listeningKey.value = button
}

function handleKeydown(e: KeyboardEvent) {
  if (!listeningKey.value) return
  e.preventDefault()
  // 保存 KeyboardEvent.code，对应物理按键，跨布局稳定
  mapping[listeningKey.value] = e.code
  listeningKey.value = null
}

function resetDefaults() {
  for (const k of Object.keys(DEFAULT_KEY_MAPPING) as ButtonName[]) {
    mapping[k] = DEFAULT_KEY_MAPPING[k]
  }
}

function handleSave() {
  saveMapping({ ...mapping })
  emit('save', { ...mapping })
  emit('close')
}

onMounted(() => {
  window.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
})

const buttons: { key: ButtonName; label: string; icon: string }[] = [
  { key: 'Up', label: '上', icon: '▲' },
  { key: 'Left', label: '左', icon: '◀' },
  { key: 'Down', label: '下', icon: '▼' },
  { key: 'Right', label: '右', icon: '▶' },
  { key: 'A', label: 'A 键', icon: 'A' },
  { key: 'B', label: 'B 键', icon: 'B' },
  { key: 'Start', label: 'Start', icon: '▶︎' },
  { key: 'Select', label: 'Select', icon: '◉' },
  { key: 'TurboA', label: '连发 A', icon: '⟳A' },
  { key: 'TurboB', label: '连发 B', icon: '⟳B' },
]
</script>

<template>
  <n-modal
    :show="show"
    preset="card"
    title="手柄按键映射"
    style="width: 560px"
    :mask-closable="false"
    @close="$emit('close')"
  >
    <div class="mapping-body">
      <!-- 左侧：手柄示意 -->
      <div class="mapping-pad">
        <div class="pad-frame">
          <div class="pad-dpad">
            <div class="dpad-btn dpad-up" :class="{ active: listeningKey === 'Up' }">▲</div>
            <div class="dpad-btn dpad-left" :class="{ active: listeningKey === 'Left' }">◀</div>
            <div class="dpad-center" />
            <div class="dpad-btn dpad-right" :class="{ active: listeningKey === 'Right' }">▶</div>
            <div class="dpad-btn dpad-down" :class="{ active: listeningKey === 'Down' }">▼</div>
          </div>
          <div class="pad-actions">
            <div class="action-btn action-a" :class="{ active: listeningKey === 'A' }">A</div>
            <div class="action-btn action-b" :class="{ active: listeningKey === 'B' }">B</div>
          </div>
          <div class="pad-meta">
            <div class="meta-btn" :class="{ active: listeningKey === 'Select' }">Select</div>
            <div class="meta-btn" :class="{ active: listeningKey === 'Start' }">Start</div>
          </div>
        </div>
      </div>
      <!-- 右侧：按键映射表 -->
      <div class="mapping-list">
        <div v-for="btn in buttons" :key="btn.key" class="mapping-row">
          <span class="mapping-label">{{ btn.label }}</span>
          <div
            class="mapping-input"
            :class="{ 'is-listening': listeningKey === btn.key }"
            tabindex="0"
            @click="startListen(btn.key)"
          >
            <span v-if="listeningKey === btn.key" class="listening-text">按下按键...</span>
            <span v-else-if="mapping[btn.key]" class="key-name">{{ mapping[btn.key] }}</span>
            <span v-else class="key-empty">未设置</span>
          </div>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="mapping-footer">
        <n-button quaternary @click="resetDefaults">恢复默认</n-button>
        <div class="footer-right">
          <n-button @click="$emit('close')">取消</n-button>
          <n-button type="primary" @click="handleSave">保存</n-button>
        </div>
      </div>
    </template>
  </n-modal>
</template>

<style scoped>
.mapping-body {
  display: flex;
  gap: 24px;
}

/* ── 手柄示意 ── */
.mapping-pad {
  flex-shrink: 0;
  width: 180px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.pad-frame {
  width: 140px;
  height: 200px;
  background: var(--color-bg-tertiary);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}

.pad-dpad {
  position: absolute;
  left: 12px;
  top: 50%;
  transform: translateY(-50%);
  display: grid;
  grid-template-columns: 20px 20px 20px;
  grid-template-rows: 20px 20px 20px;
  gap: 1px;
}

.dpad-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 10px;
  color: var(--color-text-secondary);
  background: var(--color-bg-input);
  border: 1px solid var(--color-border);
  border-radius: 2px;
  cursor: pointer;
  transition:
    background 0.15s,
    border-color 0.15s;
}

.dpad-btn.active {
  background: var(--color-accent);
  color: #fff;
  border-color: var(--color-accent);
}

.dpad-up {
  grid-column: 2;
  grid-row: 1;
}
.dpad-left {
  grid-column: 1;
  grid-row: 2;
}
.dpad-center {
  grid-column: 2;
  grid-row: 2;
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: 2px;
}
.dpad-right {
  grid-column: 3;
  grid-row: 2;
}
.dpad-down {
  grid-column: 2;
  grid-row: 3;
}

.pad-actions {
  position: absolute;
  right: 12px;
  top: 50%;
  transform: translateY(-50%);
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.action-btn {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  font-weight: 700;
  color: var(--color-text-primary);
  cursor: pointer;
  border: 1px solid var(--color-border);
  transition:
    background 0.15s,
    border-color 0.15s,
    color 0.15s;
}

.action-a {
  background: var(--color-bg-input);
  border-color: var(--color-error);
  color: var(--color-error);
}

.action-b {
  background: var(--color-bg-input);
  border-color: var(--color-warning);
  color: var(--color-warning);
}

.action-btn.active {
  background: var(--color-accent);
  color: #fff;
  border-color: var(--color-accent);
}

.pad-meta {
  position: absolute;
  bottom: 16px;
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  gap: 10px;
}

.meta-btn {
  padding: 2px 8px;
  font-size: 9px;
  color: var(--color-text-secondary);
  background: var(--color-bg-input);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition:
    background 0.15s,
    border-color 0.15s;
}

.meta-btn.active {
  background: var(--color-accent);
  color: #fff;
  border-color: var(--color-accent);
}

/* ── 映射列表 ── */
.mapping-list {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.mapping-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.mapping-label {
  font-size: var(--font-size-small);
  color: var(--color-text-primary);
  width: 60px;
  flex-shrink: 0;
}

.mapping-input {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  height: 32px;
  background: var(--color-bg-input);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: border-color 0.15s;
}

.mapping-input:hover {
  border-color: var(--color-accent);
}

.mapping-input.is-listening {
  border-color: var(--color-accent);
  box-shadow: 0 0 0 2px rgba(79, 195, 247, 0.25);
}

.key-name {
  font-size: var(--font-size-small);
  color: var(--color-text-primary);
  font-weight: 500;
}

.key-empty {
  font-size: var(--font-size-small);
  color: var(--color-text-tertiary);
}

.listening-text {
  font-size: var(--font-size-mini);
  color: var(--color-accent);
  animation: blink 1s infinite;
}

@keyframes blink {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.4;
  }
}

/* ── 底部 ── */
.mapping-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.footer-right {
  display: flex;
  gap: 8px;
}
</style>
