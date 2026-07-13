<script setup lang="ts">
import RomCard from './RomCard.vue'
import type { Rom } from '@/types/api'

defineProps<{
  roms: Rom[]
  loading: boolean
}>()

defineEmits<{
  upload: []
  'rom-click': [romId: string]
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
      <RomCard v-for="rom in roms" :key="rom.id" :rom="rom" @click="$emit('rom-click', rom.id)" />
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
