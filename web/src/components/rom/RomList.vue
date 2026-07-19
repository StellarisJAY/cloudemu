<script setup lang="ts">
import RomCard from './RomCard.vue'
import type { Rom } from '@/types/api'

defineProps<{
  roms: Rom[]
  loading: boolean
}>()

defineEmits<{
  upload: []
  'edit-rom': [rom: Rom]
  'delete-rom': [rom: Rom]
  'detail-rom': [rom: Rom]
  'start-with-rom': [rom: Rom]
}>()
</script>

<template>
  <div class="rom-list-section">
    <div class="section-header">
      <h2 class="section-title">ROM 库</h2>
      <n-button type="primary" size="small" @click="$emit('upload')"> + 上传 ROM </n-button>
    </div>

    <div v-if="loading" class="section-loading">加载中...</div>

    <div v-else-if="roms.length === 0" class="section-empty">
      <p>暂无 ROM</p>
      <p class="sub-text">上传 ROM 文件开始游戏</p>
    </div>

    <div v-else class="card-grid">
      <RomCard
        v-for="rom in roms"
        :key="rom.id"
        :rom="rom"
        @edit="$emit('edit-rom', $event)"
        @delete="$emit('delete-rom', $event)"
        @detail="$emit('detail-rom', $event)"
        @startGame="$emit('start-with-rom', $event)"
      />
    </div>
  </div>
</template>

<style scoped>
.rom-list-section {
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
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 12px;
  overflow-y: auto;
  align-content: start;
  padding: 4px 16px 12px;
  flex: 1;
}

.card-grid > :deep(.rom-card) {
  width: 100%;
}

.card-grid::-webkit-scrollbar {
  width: 4px;
}

.card-grid::-webkit-scrollbar-thumb {
  background: var(--color-scrollbar);
  border-radius: 2px;
}

.card-grid::-webkit-scrollbar-thumb:hover {
  background: var(--color-scrollbar-hover);
}
</style>
