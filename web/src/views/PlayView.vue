<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import { useRoomStore } from '@/stores/room'
import type { PlayMember } from '@/stores/room'
import { useRomStore } from '@/stores/rom'
import { useAuthStore } from '@/stores/auth'
import { useLiveKit } from '@/composables/useLiveKit'
import { useGameInput } from '@/composables/useGameInput'
import { useLatencyMeasurer } from '@/composables/useLatencyMeasurer'
import GameToolbar from '@/components/play/GameToolbar.vue'
import GameScreen from '@/components/play/GameScreen.vue'
import MemberPanel from '@/components/play/MemberPanel.vue'
import VirtualGamepad from '@/components/play/VirtualGamepad.vue'
import SaveStateDialog from '@/components/play/SaveStateDialog.vue'
import type { Room, PlayerRole } from '@/types/api'
import type { ButtonName } from '@/utils/keyMapping'
import { useMediaQuery } from '@vueuse/core'
import '@/styles/mobile.css'

const route = useRoute()
const router = useRouter()
const message = useMessage()
const roomStore = useRoomStore()
const romStore = useRomStore()
const authStore = useAuthStore()

const roomId = route.params.roomId as string

const room = computed<Room | undefined>(() => roomStore.rooms.find((r) => r.id === roomId))

// 当前选中的 ROM
const currentRomId = ref<string | undefined>()
const currentRom = computed(() => romStore.roms.find((r) => r.id === currentRomId.value))

// 筛选与房间模拟器类型匹配的 ROM
const compatibleRoms = computed(() => {
  if (!room.value) return romStore.roms
  return romStore.roms.filter((r) => r.emulator_type === room.value!.emulator_type)
})

// 监听房间 ROM 变化，自动预选；rom_id 为 null 表示房主尚未选择 ROM
watch(room, (r) => {
  if (r?.rom_id) {
    currentRomId.value = r.rom_id
  } else {
    currentRomId.value = undefined
  }
}, { immediate: true })

const isHost = computed(
  () => authStore.user?.id != null && room.value?.host_id === authStore.user.id,
)

// LiveKit 连接
const livekit = useLiveKit()
const connectionState = computed(() => livekit.connectionState.value)
const videoTrack = computed(() => livekit.videoTrack.value)
const audioTrack = computed(() => livekit.audioTrack.value)

// 模拟器状态
const emulatorState = ref<'idle' | 'loading' | 'running' | 'paused' | 'error'>('idle')

// 游戏输入：连接成功后启用，断开/暂停时禁用
const inputEnabled = computed(
  () => livekit.connectionState.value === 'connected' && emulatorState.value === 'running',
)
const gameInput = useGameInput(livekit.publishInput, inputEnabled)

// 移动端检测（仅按触摸能力判断，不限制宽度：手机横屏仍有 pointer:coarse）
const isMobile = useMediaQuery('(pointer: coarse)')
const showDrawerLeft = ref(false)
const showDrawerRight = ref(false)

/** 虚拟手柄按键回调，将触摸事件桥接到 useGameInput */
function handleGamepadButton(btn: ButtonName, pressed: boolean) {
  gameInput.applyButton(btn, pressed)
}

const latencyMeasurer = useLatencyMeasurer(
  livekit.publishInput,
  inputEnabled,
  livekit.setOnDataReceived,
)
const latencyMs = computed(() => latencyMeasurer.latencyMs.value)

const livekitToken = ref<string | undefined>(route.query.token as string | undefined)
const livekitRoom = ref<string | undefined>(route.query.livekitRoom as string | undefined)
const livekitUrl = ref<string | undefined>(route.query.livekitUrl as string | undefined)

let tokenPollTimer: ReturnType<typeof setInterval> | null = null

// 成员列表轮询
let memberPollTimer: ReturnType<typeof setInterval> | null = null
const members = ref<PlayMember[]>([])

const fps = ref(0)
const cpuPercent = ref(0)



const MEMBER_POLL_INTERVAL = 3000

// 监听 LiveKit 连接状态，同步模拟器状态
watch(
  () => livekit.connectionState.value,
  (state) => {
    if (state === 'connected') {
      emulatorState.value = 'running'
    } else if (state === 'error') {
      emulatorState.value = 'error'
    } else if (state === 'connecting') {
      emulatorState.value = 'loading'
    }
  },
)

onMounted(async () => {
  if (roomStore.rooms.length === 0) {
    await roomStore.fetchRooms()
  }
  if (!authStore.user) {
    await authStore.fetchUser()
  }
  if (romStore.roms.length === 0) {
    await romStore.fetchRoms()
  }

  // 如果房间已在游戏中，初始化模拟器状态
  if (room.value?.status === 1) {
    emulatorState.value = 'loading'
  }

  // 拉取初始成员列表
  await refreshMembers()

  // 启动成员列表轮询
  memberPollTimer = setInterval(refreshMembers, MEMBER_POLL_INTERVAL)

  // 如果已有 token（房主通过 URL 带过来），直接连接 LiveKit
  if (livekitToken.value && livekitUrl.value) {
    await livekit.connect(livekitUrl.value, livekitToken.value)
  } else {
    startTokenPolling()
  }
})

onUnmounted(() => {
  stopTokenPolling()
  if (memberPollTimer) {
    clearInterval(memberPollTimer)
    memberPollTimer = null
  }
})

async function refreshMembers() {
  try {
    members.value = await roomStore.fetchMembers(roomId)
  } catch {
    message.warning('房间已被关闭或删除')
    await livekit.disconnect()
    stopTokenPolling()
    if (memberPollTimer) {
      clearInterval(memberPollTimer)
      memberPollTimer = null
    }
    router.push('/')
  }
}

/** 轮询获取 LiveKit token（仅非房主） */
function startTokenPolling() {
  if (livekitToken.value) return
  tokenPollTimer = setInterval(pollForToken, 2000)
}

function stopTokenPolling() {
  if (tokenPollTimer) {
    clearInterval(tokenPollTimer)
    tokenPollTimer = null
  }
}

/** 轮询 GET /api/rooms/:id/livekit → 获取到 token 后自动连接 LiveKit */
async function pollForToken() {
  const resp = await roomStore.getLivekitToken(roomId)
  if (!resp || resp.waiting) return

  livekitToken.value = resp.livekit_token
  livekitRoom.value = resp.livekit_room
  livekitUrl.value = resp.livekit_url
  stopTokenPolling()

  if (resp.livekit_token && resp.livekit_url) {
    await livekit.connect(resp.livekit_url, resp.livekit_token)
  }
}

/** 成员连接游戏（手动触发重试） */
async function handleConnect() {
  if (livekitToken.value && livekitUrl.value) {
    await livekit.connect(livekitUrl.value, livekitToken.value)
  } else {
    await pollForToken()
    if (!livekitToken.value) startTokenPolling()
  }
}

async function handleSelectRom(romId: string) {
  const err = await roomStore.selectRom({ room_id: roomId, rom_id: romId })
  if (err) {
    message.error(err)
    return
  }
  currentRomId.value = romId
  message.success(`已选择 ${currentRom.value?.title ?? romId}`)
}

async function handleStartGame() {
  if (buttonLoading.value) return
  buttonLoading.value = 'start'
  try {
    const resp = await roomStore.startGame(roomId)
    if (!resp) {
      message.error('开始游戏失败：未收到服务器响应')
      return
    }
    livekitToken.value = resp.livekit_token
    livekitRoom.value = resp.livekit_room
    livekitUrl.value = resp.livekit_url
    await livekit.connect(resp.livekit_url, resp.livekit_token)
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : '开始游戏失败'
    message.error(msg)
  } finally {
    buttonLoading.value = null
  }
}

async function handlePause() {
  if (buttonLoading.value) return
  buttonLoading.value = 'pause'
  try {
    const err = await roomStore.pauseGame(roomId)
    if (err) {
      message.error(err)
      return
    }
    emulatorState.value = 'paused'
    fps.value = 0
  } finally {
    buttonLoading.value = null
  }
}

async function handleResume() {
  if (buttonLoading.value) return
  buttonLoading.value = 'resume'
  try {
    const err = await roomStore.resumeGame(roomId)
    if (err) {
      message.error(err)
      return
    }
    emulatorState.value = 'running'
    fps.value = 60
  } finally {
    buttonLoading.value = null
  }
}

async function handleStop() {
  if (buttonLoading.value) return
  buttonLoading.value = 'stop'
  try {
    const err = await roomStore.stopGame(roomId)
    if (err) {
      message.error(err)
      return
    }
    message.success('游戏已停止')
    await livekit.disconnect()
    emulatorState.value = 'idle'
    fps.value = 0
    cpuPercent.value = 0
    currentRomId.value = undefined
    router.push('/')
  } finally {
    buttonLoading.value = null
  }
}

async function handleSaveState() {
  if (buttonLoading.value) return
  buttonLoading.value = 'saveState'
  try {
    const err = await roomStore.saveState(roomId)
    if (err) {
      message.error(err)
      return
    }
    message.success('存档已保存')
  } finally {
    buttonLoading.value = null
  }
}

const showSaveStateDialog = ref(false)

function handleLoadState() {
  showSaveStateDialog.value = true
}

/** 按钮操作的加载状态，防止重复点击并为按钮提供 loading 动画 */
type ControlAction = 'start' | 'pause' | 'resume' | 'stop' | 'saveState' | 'loadLatestState' | null
const buttonLoading = ref<ControlAction>(null)

async function handleLoadLatestState() {
  if (buttonLoading.value) return
  buttonLoading.value = 'loadLatestState'
  try {
    const err = await roomStore.loadLatestState(roomId)
    if (err) {
      message.error(err)
      return
    }
    message.success('已加载最新存档')
  } finally {
    buttonLoading.value = null
  }
}

async function handleRoleChange(userId: string, role: PlayerRole, port?: number) {
  let err: string | null
  if (role === 1) {
    if (port == null) return
    err = await roomStore.changeRole({ room_id: roomId, user_id: userId, role: 1, port })
  } else {
    err = await roomStore.changeRole({ room_id: roomId, user_id: userId, role: 2 })
  }
  if (err) {
    message.error(err)
    return
  }
  message.success('操作成功')
  await refreshMembers()
}

async function handleKick(userId: string) {
  const err = await roomStore.kickPlayer({ room_id: roomId, user_id: userId })
  if (err) {
    message.error(err)
    return
  }
  message.success('已踢出玩家')
  await refreshMembers()
}

async function handleInvited() {
  await refreshMembers()
}

function handleLeave() {
  router.push('/')
}
</script>

<template>
  <div class="play-view" :class="{ 'is-mobile': isMobile }">
    <!-- 房间未找到 -->
    <div v-if="!room" class="play-not-found">
      <h2>房间不存在或已关闭</h2>
      <n-button @click="router.push('/')">返回大厅</n-button>
    </div>

    <!-- 游戏界面 -->
    <template v-else>
      <!-- 顶部状态栏 -->
      <header class="play-header">
        <div class="header-left">
          <button v-if="isMobile" class="mobile-drawer-trigger" title="工具栏" @click="showDrawerLeft = true">
            ⚙
          </button>
          <n-button text @click="handleLeave" class="back-btn"> ← 离开游戏 </n-button>
          <span class="header-title">{{ room.title }}</span>
          <n-tag :type="room.status === 1 ? 'success' : 'default'" size="small">
            {{ room.status === 0 ? '等待中' : room.status === 1 ? '游戏中' : '已关闭' }}
          </n-tag>
        </div>
        <div class="header-right">
          <button v-if="isMobile" class="mobile-drawer-trigger" title="成员列表" @click="showDrawerRight = true">
            👥
          </button>
          <span class="header-platform">{{ room.emulator_type.toUpperCase() }}</span>
        </div>
      </header>

      <!-- 三栏主区域（桌面端） -->
      <div v-if="!isMobile" class="play-body">
        <!-- 左栏：工具栏 -->
        <GameToolbar
          :is-host="isHost"
          :room-status="room?.status"
          :roms="compatibleRoms"
          :current-rom-id="currentRomId"
          :emulator-state="emulatorState"
          :fps="fps"
          :cpu-percent="cpuPercent"
          :connection-state="connectionState"
          :latency-ms="latencyMs"
          :button-loading="buttonLoading"
          @select-rom="handleSelectRom"
          @start-game="handleStartGame"
          @pause="handlePause"
          @resume="handleResume"
          @stop="handleStop"
          @save-state="handleSaveState"
          @load-state="handleLoadState"
          @load-latest-state="handleLoadLatestState"
          @connect="handleConnect"
          @key-mapping-saved="gameInput.reloadMapping"
        />

        <!-- 中栏：游戏画面 -->
        <GameScreen
          :emulator-type="room.emulator_type"
          :room-title="room.title"
          :video-track="videoTrack"
          :audio-track="audioTrack"
          :latency-ms="latencyMs"
        />

        <!-- 右栏：成员面板 -->
        <MemberPanel
          :members="members"
          :room-id="roomId"
          :is-host="isHost"
          :max-ports="room?.max_ports ?? 4"
          @role-change="handleRoleChange"
          @kick="handleKick"
          @invited="handleInvited"
        />
      </div>

      <!-- 移动端：三段式 + 抽屉 + 竖屏提示 -->
      <template v-else>
        <div class="play-body-mobile">
          <div class="gamepad-zone">
            <VirtualGamepad
              side="left"
              :enabled="inputEnabled"
              @button-change="handleGamepadButton"
            />
          </div>
          <GameScreen
            :emulator-type="room.emulator_type"
            :room-title="room.title"
            :video-track="videoTrack"
            :audio-track="audioTrack"
            :latency-ms="latencyMs"
            :is-mobile="true"
          />
          <div class="gamepad-zone">
            <VirtualGamepad
              side="right"
              :enabled="inputEnabled"
              @button-change="handleGamepadButton"
            />
          </div>
        </div>

        <n-drawer v-model:show="showDrawerLeft" placement="left" :width="260">
          <GameToolbar
            :is-host="isHost"
            :room-status="room?.status"
            :roms="compatibleRoms"
            :current-rom-id="currentRomId"
            :emulator-state="emulatorState"
            :fps="fps"
            :cpu-percent="cpuPercent"
            :connection-state="connectionState"
            :latency-ms="latencyMs"
            :button-loading="buttonLoading"
            @select-rom="handleSelectRom"
            @start-game="handleStartGame"
            @pause="handlePause"
            @resume="handleResume"
            @stop="handleStop"
            @save-state="handleSaveState"
            @load-state="handleLoadState"
            @load-latest-state="handleLoadLatestState"
            @connect="handleConnect"
            @key-mapping-saved="gameInput.reloadMapping"
          />
        </n-drawer>

        <n-drawer v-model:show="showDrawerRight" placement="right" :width="260">
          <MemberPanel
            :members="members"
            :room-id="roomId"
            :is-host="isHost"
            :max-ports="room?.max_ports ?? 4"
            @role-change="handleRoleChange"
            @kick="handleKick"
            @invited="handleInvited"
          />
        </n-drawer>

        <div class="mobile-rotate-overlay">
          <div class="rotate-icon">
            <svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
              <rect x="5" y="2" width="14" height="20" rx="2" ry="2" />
              <path d="M12 18h.01" />
            </svg>
          </div>
          <span>请旋转手机至横屏</span>
        </div>
      </template>
    </template>

    <SaveStateDialog
      :show="showSaveStateDialog"
      :room-id="roomId"
      @close="showSaveStateDialog = false"
    />
  </div>
</template>

<style scoped>
.play-view {
  display: flex;
  flex-direction: column;
  height: 100vh;
  /* 移动端浏览器工具栏会占用高度，100vh 会超出可视区导致底部被裁切；
     用动态视口高度 dvh 让布局严格贴合当前屏幕可视范围 */
  height: 100dvh;
  background: var(--color-bg-primary);
  overflow: hidden;
}

.play-not-found {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  gap: 16px;
}

.play-not-found h2 {
  color: var(--color-text-secondary);
}

/* ── 顶栏 ── */
.play-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 48px;
  padding: 0 16px;
  background: var(--color-bg-secondary);
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.back-btn {
  font-size: var(--font-size-small);
}

.header-title {
  font-size: var(--font-size-medium);
  font-weight: 600;
  color: var(--color-text-primary);
}

.header-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.header-platform {
  font-size: var(--font-size-small);
  color: var(--color-text-secondary);
  font-weight: 600;
}

/* ── 主体三栏 ── */
.play-body {
  display: flex;
  flex: 1;
  overflow: hidden;
}
</style>
