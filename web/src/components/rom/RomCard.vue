<script setup lang="ts">
import type { Rom, EmulatorType } from '@/types/api'

defineProps<{
  rom: Rom
}>()

defineEmits<{
  click: [romId: string]
}>()

const emulatorLabels: Record<EmulatorType, string> = {
  nes: 'NES',
  gb: 'GBC/GBA',
  dos: 'DOS',
}

const emulatorCover: Record<EmulatorType, string> = {
  nes: '/assets/default-cover-nes.png',
  gb: '/assets/default-cover-gb.png',
  dos: '/assets/default-cover-dos.png',
}

function coverUrl(coverPath: string | null): string {
  if (coverPath) return `/api/files/${coverPath}`
  return ''
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}
</script>

<template>
  <div class="rom-card" @click="$emit('click', rom.id)">
    <div class="card-cover" :class="`cover-${rom.emulator_type}`">
      <img
        v-if="rom.cover_path"
        :src="coverUrl(rom.cover_path)"
        :alt="rom.title"
        class="cover-img"
      />
      <img
        v-else
        :src="emulatorCover[rom.emulator_type]"
        :alt="rom.emulator_type"
        class="cover-img"
      />
      <span class="card-emu-tag">{{ emulatorLabels[rom.emulator_type] }}</span>
    </div>
    <div class="card-info">
      <span class="card-rom-title">{{ rom.title }}</span>
      <span class="card-rom-size">{{ formatSize(rom.file_size) }}</span>
    </div>
  </div>
</template>

<style scoped>
.rom-card {
  flex-shrink: 0;
  width: 200px;
  cursor: pointer;
  border-radius: var(--radius-lg);
  overflow: hidden;
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  transition:
    transform 0.2s,
    box-shadow 0.2s;
}

.rom-card:hover {
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
}

.card-emu-tag {
  position: absolute;
  top: 8px;
  right: 8px;
  padding: 2px 6px;
  border-radius: var(--radius-sm);
  background: rgba(0, 0, 0, 0.6);
  color: #fff;
  font-size: 11px;
  font-weight: 600;
}

.card-info {
  display: flex;
  flex-direction: column;
  padding: 10px 12px;
  gap: 4px;
}

.card-rom-title {
  font-size: var(--font-size-small);
  font-weight: 600;
  color: var(--color-text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.card-rom-size {
  font-size: var(--font-size-mini);
  color: var(--color-text-secondary);
}
</style>
