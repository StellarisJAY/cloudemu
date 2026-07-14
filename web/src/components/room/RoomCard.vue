<script setup lang="ts">
import { computed, h } from 'vue'
import { NDropdown, NButton, type DropdownOption } from 'naive-ui'
import type { Room, EmulatorType } from '@/types/api'

const props = defineProps<{
  room: Room
  currentUserId?: string
}>()

const emit = defineEmits<{
  click: [roomId: string]
  delete: [roomId: string]
  stop: [roomId: string]
  leave: [roomId: string]
}>()

const isHost = computed(() => props.currentUserId != null && props.currentUserId === props.room.host_id)

const showMore = computed(() => {
  if (props.room.status === 0 && isHost.value) return true
  if (props.room.status === 1) return true
  return false
})

const menuOptions = computed<DropdownOption[]>(() => {
  if (props.room.status === 0) {
    return [
      {
        key: 'delete',
        label: () => h('span', { style: { color: '#ef4444' } }, '删除房间'),
      },
    ]
  }
  if (isHost.value) {
    return [
      {
        key: 'stop',
        label: () => h('span', { style: { color: '#ef4444' } }, '停止游戏'),
      },
    ]
  }
  return [
    {
      key: 'leave',
      label: () => h('span', { style: { color: '#ef4444' } }, '退出房间'),
    },
  ]
})

function handleMenuSelect(key: string) {
  if (key === 'delete') {
    emit('delete', props.room.id)
  } else if (key === 'stop') {
    emit('stop', props.room.id)
  } else if (key === 'leave') {
    emit('leave', props.room.id)
  }
}

const emulatorLabels: Record<EmulatorType, string> = {
  nes: 'NES',
  gb: 'GBC/GBA',
  dos: 'DOS',
}

const statusLabels: Record<number, string> = {
  0: '等待中',
  1: '游戏中',
  2: '已关闭',
}

const emulatorCover: Record<EmulatorType, string> = {
  nes: '/assets/default-cover-nes.png',
  gb: '/assets/default-cover-gb.png',
  dos: '/assets/default-cover-dos.png',
}
</script>

<template>
  <div class="room-card">
    <div class="card-cover" :class="`cover-${room.emulator_type}`" @click="emit('click', room.id)">
      <img :src="emulatorCover[room.emulator_type]" :alt="room.emulator_type" class="cover-img" />
      <div class="cover-overlay" />
      <span class="card-status" :class="`status-${room.status}`">
        {{ statusLabels[room.status] ?? room.status }}
      </span>
      <div v-if="showMore" class="card-more" @click.stop>
        <n-dropdown trigger="click" :options="menuOptions" @select="handleMenuSelect">
          <n-button text size="tiny" class="more-btn">⋯</n-button>
        </n-dropdown>
      </div>
      <div class="card-title-area">
        <span class="card-title">{{ room.title }}</span>
        <span class="card-emulator">{{ emulatorLabels[room.emulator_type] }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.room-card {
  flex-shrink: 0;
  width: 230px;
  cursor: pointer;
  border-radius: var(--radius-lg);
  overflow: hidden;
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  transition:
    transform 0.2s,
    box-shadow 0.2s;
}

.room-card:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-lg);
}

.card-cover {
  position: relative;
  height: 140px;
  overflow: hidden;
}

.cover-nes {
  background: #636363;
}
.cover-gb {
  background: #4a148c;
}
.cover-dos {
  background: #1b5e20;
}

.cover-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  opacity: 0.6;
}

.cover-overlay {
  position: absolute;
  inset: 0;
  background: linear-gradient(0deg, rgba(0, 0, 0, 0.7) 0%, transparent 60%);
}

.card-status {
  position: absolute;
  top: 8px;
  right: 8px;
  padding: 2px 8px;
  border-radius: var(--radius-sm);
  font-size: var(--font-size-mini);
  font-weight: 600;
}

.status-0 {
  background: var(--color-success);
  color: #fff;
}
.status-1 {
  background: var(--color-info);
  color: #fff;
}
.status-2 {
  background: var(--color-text-tertiary);
  color: #fff;
}

.card-more {
  position: absolute;
  top: 2px;
  left: 2px;
}

.more-btn {
  color: rgba(255, 255, 255, 0.5) !important;
  font-size: 18px;
  line-height: 1;
  padding: 2px 6px;
  border-radius: var(--radius-sm);
}

.more-btn:hover {
  color: rgba(255, 255, 255, 0.9) !important;
  background: rgba(0, 0, 0, 0.3) !important;
}

.card-title-area {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  padding: 10px 12px;
}

.card-title {
  display: block;
  font-size: var(--font-size-small);
  font-weight: 600;
  color: #fff;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.card-emulator {
  display: block;
  font-size: var(--font-size-mini);
  color: rgba(255, 255, 255, 0.7);
  margin-top: 2px;
}
</style>
