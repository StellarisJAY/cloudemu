<script setup lang="ts">
import { ref, watch } from 'vue'
import type { EmulatorType } from '@/types/api'

const props = defineProps<{
  emulatorType?: EmulatorType
  roomTitle?: string
  videoTrack?: MediaStreamTrack | null
  latencyMs?: number | null
  isMobile?: boolean
}>()

const videoEl = ref<HTMLVideoElement>()

watch(
  () => props.videoTrack,
  (track) => {
    if (!videoEl.value) return
    if (track) {
      videoEl.value.srcObject = new MediaStream([track])
    } else {
      videoEl.value.srcObject = null
    }
  },
  { immediate: true },
)

const emulatorCover: Record<EmulatorType, string> = {
  nes: '/assets/default-cover-nes.png',
  gb: '/assets/default-cover-gb.png',
  dos: '/assets/default-cover-dos.png',
}
</script>

<template>
  <div class="game-screen">
    <!-- 顶部半透明 HUD -->
    <div class="screen-hud screen-hud--top">
      <span v-if="roomTitle" class="hud-room-name">{{ roomTitle }}</span>
      <div class="hud-stats">
        <span class="hud-stat">FPS: --</span>
        <span class="hud-stat">延迟: {{ latencyMs != null ? latencyMs + 'ms' : '--ms' }}</span>
      </div>
    </div>

    <!-- 视频播放区域 -->
    <div class="screen-video-area">
      <!-- 远端视频流 -->
      <video
        v-show="videoTrack"
        ref="videoEl"
        autoplay
        muted
        playsinline
        class="screen-video"
      />

      <!-- 无流时的占位 -->
      <div v-if="!videoTrack" class="screen-placeholder">
        <img
          v-if="emulatorType"
          :src="emulatorCover[emulatorType]"
          :alt="emulatorType"
          class="placeholder-cover"
        />
        <span class="placeholder-text">等待游戏开始...</span>
      </div>
    </div>

    <!-- 底部半透明控制提示 -->
    <div class="screen-hud screen-hud--bottom">
      <span class="hud-hint">{{ isMobile ? '⚙ 点击此处打开菜单' : 'ESC 打开菜单' }}</span>
    </div>
  </div>
</template>

<style scoped>
.game-screen {
  flex: 1;
  display: flex;
  flex-direction: column;
  position: relative;
  background: #000;
  min-width: 0;
}

.screen-hud {
  position: absolute;
  left: 0;
  right: 0;
  padding: 8px 16px;
  display: flex;
  align-items: center;
  z-index: 2;
  pointer-events: none;
}

.screen-hud--top {
  top: 0;
  justify-content: space-between;
  background: linear-gradient(180deg, rgba(0, 0, 0, 0.55) 0%, transparent 100%);
}

.screen-hud--bottom {
  bottom: 0;
  justify-content: flex-end;
  background: linear-gradient(0deg, rgba(0, 0, 0, 0.55) 0%, transparent 100%);
}

.hud-room-name {
  font-size: var(--font-size-small);
  font-weight: 600;
  color: rgba(255, 255, 255, 0.85);
}

.hud-stats {
  display: flex;
  gap: 16px;
}

.hud-stat {
  font-size: var(--font-size-mini);
  color: rgba(255, 255, 255, 0.55);
  font-family: monospace;
}

.hud-hint {
  font-size: var(--font-size-mini);
  color: rgba(255, 255, 255, 0.3);
}

/* ── 视频/占位区 ── */
.screen-video-area {
  flex: 1;
  min-height: 0;
  min-width: 0;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
}

.screen-video {
  max-width: 100%;
  max-height: 100%;
  width: auto;
  height: auto;
  object-fit: contain;
}

.screen-placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
  opacity: 0.5;
}

.placeholder-cover {
  width: 240px;
  height: 240px;
  object-fit: contain;
  border-radius: var(--radius-md);
}

.placeholder-text {
  font-size: var(--font-size-small);
  color: rgba(255, 255, 255, 0.45);
}
</style>
