<script setup lang="ts">
import { ref, computed } from 'vue'
import { useDialog, useMessage } from 'naive-ui'
import KeyMappingDialog from './KeyMappingDialog.vue'
import type { Rom } from '@/types/api'

/** 成员连接状态 */
export type ConnectionState = 'waiting' | 'ready' | 'connecting' | 'connected' | 'error'

const props = defineProps<{
  isHost: boolean
  roomStatus?: number
  roms: Rom[]
  currentRomId?: string
  emulatorState: 'idle' | 'loading' | 'running' | 'paused' | 'error'
  fps: number
  cpuPercent: number
  connectionState: ConnectionState
  latencyMs?: number | null
  /** 当前正在执行的操作，null 表示空闲；用于按钮 loading 动画和防重复点击 */
  buttonLoading?: string | null
}>()

const emit = defineEmits<{
  selectRom: [romId: string]
  switchRom: [romId: string]
  startGame: []
  pause: []
  resume: []
  stop: []
  saveState: []
  loadState: []
  loadLatestState: []
  connect: []
  keyMappingSaved: []
}>()

const currentRom = computed(() => props.roms.find((r) => r.id === props.currentRomId))

const message = useMessage()
const dialog = useDialog()

const showKeyMapping = ref(false)

/** 是否存在 ROM 切换中（playing 状态下拦截 ROM 选择，待用户确认） */
const pendingRomId = ref<string | null>(null)

/** 是否正在游戏中（roomStatus=1 且模拟器已启动） */
const isPlaying = computed(() => props.roomStatus === 1 && (props.emulatorState === 'running' || props.emulatorState === 'paused'))

/** 处理 ROM 选择：waiting 状态直接切换，playing 状态弹窗确认 */
function handleRomSelect(romId: string) {
  if (romId === props.currentRomId) return
  if (isPlaying.value) {
    pendingRomId.value = romId
    dialog.warning({
      title: '确认切换 ROM',
      content: '切换 ROM 将丢失当前游戏进度，确定继续？',
      positiveText: '确定切换',
      negativeText: '取消',
      onPositiveClick: () => {
        if (pendingRomId.value) {
          emit('switchRom', pendingRomId.value)
        }
        pendingRomId.value = null
      },
      onNegativeClick: () => {
        pendingRomId.value = null
      },
      onClose: () => {
        pendingRomId.value = null
      },
    })
  } else {
    emit('selectRom', romId)
  }
}

/** 是否有控制操作正在执行中，所有控制按钮在此时禁用 */
const isBusy = computed(() => props.buttonLoading != null && props.buttonLoading !== '')

const stateLabels: Record<string, string> = {
  idle: '就绪',
  loading: '加载中...',
  running: '运行中',
  paused: '已暂停',
  error: '出错',
}

const stateColors: Record<string, string> = {
  idle: '#8b9bb4',
  loading: '#f59e0b',
  running: '#4ade80',
  paused: '#f59e0b',
  error: '#ef4444',
}

const connectionLabel: Record<ConnectionState, string> = {
  waiting: '等待房主开始游戏...',
  ready: '🔌 连接',
  connecting: '连接中...',
  connected: '✅ 已连接',
  error: '连接失败，点击重试',
}

const connectionBtnType: Record<ConnectionState, string> = {
  waiting: 'default',
  ready: 'primary',
  connecting: 'default',
  connected: 'success',
  error: 'error',
}
</script>

<template>
  <div class="game-toolbar">
    <!-- ==================== 房主区域 ==================== -->
    <template v-if="isHost">
      <!-- ROM 选择 -->
      <div class="toolbar-section">
        <div class="section-header">ROM 选择</div>
        <n-select
          :value="currentRomId"
          :options="roms.map((r) => ({ label: r.title, value: r.id }))"
          placeholder="选择 ROM 文件"
          filterable
          class="rom-select"
          @update:value="handleRomSelect"
        />
        <div v-if="currentRom" class="current-rom">
          <span class="current-rom-label">{{ currentRom.title }}</span>
        </div>
      </div>

      <!-- 模拟器状态 -->
      <div class="toolbar-section">
        <div class="section-header">模拟器状态</div>
        <div class="stat-grid">
          <div class="stat-item">
            <span class="stat-label">状态</span>
            <span class="stat-value" :style="{ color: stateColors[emulatorState] }">
              {{ stateLabels[emulatorState] || '未知' }}
            </span>
          </div>
          <div class="stat-item">
            <span class="stat-label">FPS</span>
            <span class="stat-value stat-mono">{{ fps > 0 ? fps : '--' }}</span>
          </div>
          <div class="stat-item">
            <span class="stat-label">CPU</span>
            <span class="stat-value stat-mono">{{
              cpuPercent > 0 ? cpuPercent + '%' : '--%'
            }}</span>
          </div>
          <div class="stat-item">
            <span class="stat-label">延迟</span>
            <span class="stat-value stat-mono">{{
              latencyMs != null ? latencyMs + 'ms' : '--ms'
            }}</span>
          </div>
        </div>
        <n-progress
          v-if="cpuPercent > 0"
          type="line"
          :percentage="cpuPercent"
          :color="cpuPercent > 80 ? '#ef4444' : '#4fc3f7'"
          :height="4"
          :border-radius="2"
          :show-indicator="false"
          class="cpu-bar"
        />
      </div>

      <!-- 游戏控制 -->
      <div class="toolbar-section">
        <div class="section-header">游戏控制</div>
        <div class="control-buttons">
          <n-button
            v-if="(emulatorState === 'idle' || emulatorState === 'error') && roomStatus === 0"
            block
            type="primary"
            :disabled="isBusy"
            :loading="buttonLoading === 'start'"
            @click="$emit('startGame')"
          >
            🚀 开始游戏
          </n-button>
          <n-button
            v-if="emulatorState === 'running'"
            block
            secondary
            :disabled="isBusy"
            :loading="buttonLoading === 'pause'"
            @click="$emit('pause')"
          >
            ⏯ 暂停
          </n-button>
          <n-button
            v-else-if="emulatorState === 'paused'"
            block
            secondary
            type="primary"
            :disabled="isBusy"
            :loading="buttonLoading === 'resume'"
            @click="$emit('resume')"
          >
            ▶ 继续
          </n-button>
          <n-button
            v-if="emulatorState === 'running' || emulatorState === 'paused'"
            block
            secondary
            type="warning"
            :disabled="isBusy"
            :loading="buttonLoading === 'stop'"
            @click="$emit('stop')"
          >
            ⏹ 停止
          </n-button>
          <n-button
            v-if="emulatorState === 'running'"
            block
            secondary
            :disabled="isBusy"
            :loading="buttonLoading === 'saveState'"
            @click="$emit('saveState')"
          >
            💾 存档
          </n-button>
          <n-button
            v-if="emulatorState === 'running' || emulatorState === 'paused'"
            block
            secondary
            :disabled="isBusy"
            :loading="buttonLoading === 'loadLatestState'"
            @click="$emit('loadLatestState')"
          >
            ⏪ 加载最新存档
          </n-button>
          <n-button
            v-if="emulatorState === 'running' || emulatorState === 'paused'"
            block
            secondary
            :disabled="isBusy"
            @click="$emit('loadState')"
          >
            📂 读档
          </n-button>
          <span
            v-if="emulatorState === 'loading'"
            class="control-hint"
          >
            加载中...
          </span>
        </div>
      </div>
    </template>

    <!-- ==================== 成员区域 ==================== -->
    <template v-else>
      <!-- 连接状态 -->
      <div class="toolbar-section">
        <div class="section-header">游戏状态</div>
        <div class="connect-area">
          <!-- 等待中：静态提示 -->
          <div v-if="connectionState === 'waiting'" class="connect-waiting">
            <span class="waiting-dot" />
            <span class="waiting-text">等待房主开始游戏...</span>
          </div>

          <!-- 就绪 / 连接中 / 错误：可点击按钮 -->
          <n-button
            v-else
            block
            :type="
              connectionBtnType[connectionState] as 'default' | 'primary' | 'success' | 'error'
            "
            :disabled="connectionState === 'connecting'"
            :loading="connectionState === 'connecting'"
            @click="$emit('connect')"
          >
            {{ connectionLabel[connectionState] }}
          </n-button>

          <!-- 已连接：静态显示 -->
          <div v-if="connectionState === 'connected'" class="connect-detail">
            <span class="detail-label">推流延迟</span>
            <span class="detail-value">{{ latencyMs != null ? latencyMs + 'ms' : '-- ms' }}</span>
          </div>
        </div>
      </div>

      <!-- 提示：无 ROM/控制 权限 -->
      <div v-if="connectionState !== 'connected'" class="toolbar-hint">
        ROM 选择和游戏控制由房主操作
      </div>
    </template>

    <!-- 分隔线 -->
    <div class="toolbar-divider" />

    <!-- ==================== 按键映射（所有人可见） ==================== -->
    <div class="toolbar-section">
      <div class="section-header">手柄设置</div>
      <n-button block secondary @click="showKeyMapping = true"> 🎮 按键映射 </n-button>
    </div>

    <!-- 按键映射弹窗 -->
    <KeyMappingDialog
      :show="showKeyMapping"
      @close="showKeyMapping = false"
      @save="
        () => {
          message.success('按键映射已保存')
          $emit('keyMappingSaved')
        }
      "
    />
  </div>
</template>

<style scoped>
.game-toolbar {
  width: 260px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  padding: 16px;
  gap: 12px;
  background: var(--color-bg-secondary);
  border-right: 1px solid var(--color-border);
  overflow-y: auto;
}

.game-toolbar::-webkit-scrollbar {
  width: 4px;
}

.game-toolbar::-webkit-scrollbar-thumb {
  background: var(--color-scrollbar);
  border-radius: 2px;
}

/* ── 模块卡片 ── */
.toolbar-section {
  background: var(--color-bg-tertiary);
  border-radius: var(--radius-md);
  padding: 12px;
}

.section-header {
  font-size: 11px;
  font-weight: 600;
  color: var(--color-text-tertiary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
  margin-bottom: 10px;
}

/* ── ROM 选择 ── */
.rom-select {
  width: 100%;
}

.current-rom {
  margin-top: 8px;
  padding: 6px 8px;
  background: var(--color-bg-primary);
  border-radius: var(--radius-sm);
}

.current-rom-label {
  font-size: var(--font-size-mini);
  color: var(--color-text-secondary);
}

/* ── 状态 ── */
.stat-grid {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.stat-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.stat-label {
  font-size: var(--font-size-mini);
  color: var(--color-text-secondary);
}

.stat-value {
  font-size: var(--font-size-mini);
  font-weight: 600;
  color: var(--color-text-primary);
}

.stat-mono {
  font-family: monospace;
}

.cpu-bar {
  margin-top: 8px;
}

/* ── 控制按钮 ── */
.control-buttons {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.control-hint {
  font-size: var(--font-size-mini);
  color: var(--color-text-tertiary);
  text-align: center;
  padding: 8px 0;
}

/* ── 连接区域（成员） ── */
.connect-area {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.connect-waiting {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 0;
}

.waiting-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--color-warning);
  animation: pulse 1.5s infinite;
}

@keyframes pulse {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.3;
  }
}

.waiting-text {
  font-size: var(--font-size-small);
  color: var(--color-text-secondary);
}

.connect-detail {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.detail-label {
  font-size: var(--font-size-mini);
  color: var(--color-text-secondary);
}

.detail-value {
  font-size: var(--font-size-mini);
  font-weight: 600;
  color: var(--color-text-primary);
  font-family: monospace;
}

/* ── 提示 ── */
.toolbar-hint {
  font-size: var(--font-size-mini);
  color: var(--color-text-tertiary);
  text-align: center;
  padding: 8px;
  background: var(--color-bg-tertiary);
  border-radius: var(--radius-md);
}

/* ── 分隔线 ── */
.toolbar-divider {
  height: 1px;
  background: var(--color-divider);
  margin: 0 4px;
}
</style>
