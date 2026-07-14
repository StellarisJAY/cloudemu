<script setup lang="ts">
import RoomCard from './RoomCard.vue'
import type { Room } from '@/types/api'

defineProps<{
  rooms: Room[]
  loading: boolean
  currentUserId?: string
}>()

defineEmits<{
  create: []
  'room-click': [roomId: string]
  'room-delete': [roomId: string]
  'room-stop': [roomId: string]
  'room-leave': [roomId: string]
}>()
</script>

<template>
  <div class="room-list-section">
    <div class="section-header">
      <h2 class="section-title">游戏房间</h2>
      <n-button type="primary" size="small" @click="$emit('create')"> + 创建房间 </n-button>
    </div>

    <div v-if="loading" class="section-loading">加载中...</div>

    <div v-else-if="rooms.length === 0" class="section-empty">
      <p>暂无房间</p>
      <p class="sub-text">创建一个房间，邀请好友一起玩</p>
    </div>

    <div v-else class="card-grid">
      <RoomCard
        v-for="room in rooms"
        :key="room.id"
        :room="room"
        :current-user-id="currentUserId"
        @click="$emit('room-click', room.id)"
        @delete="$emit('room-delete', room.id)"
        @stop="$emit('room-stop', room.id)"
        @leave="$emit('room-leave', room.id)"
      />
    </div>
  </div>
</template>

<style scoped>
.room-list-section {
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 16px;
  margin-bottom: 12px;
  flex-shrink: 0;
}

.section-title {
  margin: 0;
  font-size: var(--font-size-medium);
  font-weight: 600;
  color: var(--color-text-primary);
}

.section-loading,
.section-empty {
  padding: 24px 16px;
  text-align: center;
  color: var(--color-text-secondary);
  font-size: var(--font-size-small);
}

.section-empty p {
  margin: 0;
}

.section-empty .sub-text {
  margin-top: 4px;
  font-size: var(--font-size-mini);
  color: var(--color-text-tertiary);
}

.card-grid {
  display: flex;
  gap: 12px;
  overflow-x: auto;
  padding: 4px 16px 12px;
  flex: 1;
}

.card-grid::-webkit-scrollbar {
  height: 4px;
}

.card-grid::-webkit-scrollbar-thumb {
  background: var(--color-scrollbar);
  border-radius: 2px;
}

.card-grid::-webkit-scrollbar-thumb:hover {
  background: var(--color-scrollbar-hover);
}
</style>
