<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'

const props = defineProps<{
  bgBase64: string
  tileBase64: string
  tileWidth: number
  tileHeight: number
}>()

const emit = defineEmits<{
  slideEnd: [pos: { x: number; y: number }]
}>()

const trackRef = ref<HTMLElement | null>(null)
const trackWidth = ref(300)
const trackHeight = computed(() => props.tileHeight + 10)
const dragging = ref(false)
const tileX = ref(0)
const tileY = ref(0)
const done = ref(false)

const tileStyle = computed(() => ({
  width: `${props.tileWidth}px`,
  height: `${props.tileHeight}px`,
  transform: `translateX(${tileX.value}px)`,
  transition: dragging.value ? 'none' : 'transform 0.2s ease',
}))

// 后端 ToBase64() 已返回完整 data URI（含 data:image/...;base64, 前缀），直接使用即可
const bgSrc = computed(() => props.bgBase64)
const tileSrc = computed(() => props.tileBase64)

function clampX(x: number): number {
  return Math.max(0, Math.min(x, trackWidth.value - props.tileWidth))
}

function onDragStart(e: MouseEvent | TouchEvent) {
  if (done.value) return
  dragging.value = true
  e.preventDefault()
}

function onMove(clientX: number) {
  if (!dragging.value || !trackRef.value) return
  const rect = trackRef.value.getBoundingClientRect()
  tileX.value = clampX(clientX - rect.left - props.tileWidth / 2)
}

function onDragMove(e: MouseEvent) {
  onMove(e.clientX)
}

function onTouchMove(e: TouchEvent) {
  if (e.touches.length > 0) {
    onMove(e.touches[0]!.clientX)
  }
}

function finish() {
  if (!dragging.value || done.value) return
  dragging.value = false
  done.value = true

  emit('slideEnd', { x: Math.round(tileX.value), y: 0 })
}

function onDragEnd() {
  finish()
}

function onTouchEnd(e: TouchEvent) {
  e.preventDefault()
  finish()
}

onMounted(() => {
  if (trackRef.value) trackWidth.value = trackRef.value.clientWidth
  window.addEventListener('mousemove', onDragMove)
  window.addEventListener('mouseup', onDragEnd)
  window.addEventListener('touchmove', onTouchMove, { passive: false })
  window.addEventListener('touchend', onTouchEnd)
})

onBeforeUnmount(() => {
  window.removeEventListener('mousemove', onDragMove)
  window.removeEventListener('mouseup', onDragEnd)
  window.removeEventListener('touchmove', onTouchMove)
  window.removeEventListener('touchend', onTouchEnd)
})

defineExpose({
  reset: () => {
    tileX.value = 0
    done.value = false
  },
})
</script>

<template>
  <div class="slide-captcha">
    <div class="slide-bg">
      <img :src="bgSrc" alt="" draggable="false" />
    </div>

    <div ref="trackRef" class="slide-track" :style="{ height: trackHeight + 'px' }">
      <div
        class="slide-tile"
        :class="{ dragging, done }"
        :style="tileStyle"
        @mousedown="onDragStart"
        @touchstart.prevent="onDragStart"
      >
        <img :src="tileSrc" alt="" draggable="false" />
      </div>

      <div class="slide-hint">
        <template v-if="!done">&larr; 拖动滑块完成拼图 &rarr;</template>
        <template v-else>请稍候...</template>
      </div>
    </div>
  </div>
</template>

<style scoped>
.slide-captcha {
  width: 300px;
  margin: 0 auto;
}

.slide-bg {
  width: 300px;
  height: 200px;
  border-radius: var(--radius-md);
  overflow: hidden;
  border: 1px solid var(--color-border);
  margin-bottom: 8px;
}

.slide-bg img {
  width: 100%;
  height: 100%;
  object-fit: fill;
  user-select: none;
  pointer-events: none;
}

.slide-track {
  position: relative;
  width: 300px;
  background: var(--color-bg-input);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  overflow: hidden;
  cursor: pointer;
}

.slide-tile {
  position: absolute;
  top: 0;
  left: 0;
  cursor: grab;
  z-index: 2;
  border-radius: 2px;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.3);
}

.slide-tile img {
  width: 100%;
  height: 100%;
  object-fit: fill;
  user-select: none;
  pointer-events: none;
}

.slide-tile.dragging {
  cursor: grabbing;
  box-shadow: 0 4px 12px rgba(79, 195, 247, 0.5);
}

.slide-tile.done {
  cursor: default;
  pointer-events: none;
}

.slide-hint {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  color: var(--color-text-tertiary);
  pointer-events: none;
  z-index: 0;
}
</style>
