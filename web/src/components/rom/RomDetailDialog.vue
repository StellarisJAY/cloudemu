<script setup lang="ts">
import type { Rom, EmulatorType } from '@/types/api'

defineProps<{
  show: boolean
  rom: Rom | null
}>()

defineEmits<{
  close: []
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
  <n-modal
    :show="show && rom != null"
    preset="card"
    title="ROM 详情"
    style="width: 400px"
    @update:show="(v: boolean) => !v && $emit('close')"
  >
    <template v-if="rom">
      <div class="detail-body">
        <div class="detail-cover">
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
          <span class="detail-emu-tag">{{ emulatorLabels[rom.emulator_type] }}</span>
          <span v-if="rom.is_builtin" class="detail-builtin-tag">平台内置</span>
        </div>

        <div class="detail-info">
          <div class="detail-row">
            <label>名称</label>
            <span>{{ rom.title }}</span>
          </div>
          <div class="detail-row">
            <label>模拟器类型</label>
            <span>{{ emulatorLabels[rom.emulator_type] }}</span>
          </div>
          <div class="detail-row">
            <label>文件大小</label>
            <span>{{ formatSize(rom.file_size) }}</span>
          </div>
        </div>
      </div>
    </template>
  </n-modal>
</template>

<style scoped>
.detail-body {
  padding-top: 4px;
}

.detail-cover {
  position: relative;
  width: 100%;
  height: 200px;
  border-radius: var(--radius-md);
  overflow: hidden;
  margin-bottom: 16px;
}

.cover-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.detail-emu-tag {
  position: absolute;
  top: 8px;
  right: 8px;
  padding: 2px 8px;
  border-radius: var(--radius-sm);
  background: rgba(0, 0, 0, 0.6);
  color: #fff;
  font-size: 12px;
  font-weight: 600;
}

.detail-builtin-tag {
  position: absolute;
  top: 8px;
  left: 8px;
  padding: 2px 8px;
  border-radius: var(--radius-sm);
  background: var(--color-accent);
  color: #fff;
  font-size: 12px;
  font-weight: 600;
}

.detail-info {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.detail-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 4px;
}

.detail-row label {
  font-size: var(--font-size-small);
  color: var(--color-text-secondary);
}

.detail-row span {
  font-size: var(--font-size-small);
  color: var(--color-text-primary);
  font-weight: 500;
}
</style>
