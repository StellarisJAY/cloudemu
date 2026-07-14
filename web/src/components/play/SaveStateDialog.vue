<script setup lang="ts">
import { ref, watch } from 'vue'
import { useMessage, NModal, NButton, NEmpty, NSpin, NPopconfirm } from 'naive-ui'
import { useRoomStore } from '@/stores/room'
import type { SaveState } from '@/types/api'

const props = defineProps<{
  show: boolean
  roomId: string
}>()

const emit = defineEmits<{
  close: []
}>()

const roomStore = useRoomStore()
const message = useMessage()

const saveStates = ref<SaveState[]>([])
const loading = ref(false)
const loadingId = ref<string | null>(null)

async function refresh() {
  loading.value = true
  try {
    saveStates.value = await roomStore.listSaveStates(props.roomId)
  } finally {
    loading.value = false
  }
}

async function handleLoad(saveStateId: string) {
  loadingId.value = saveStateId
  try {
    const err = await roomStore.loadState(props.roomId, saveStateId)
    if (err) {
      message.error(err)
      return
    }
    message.success('读档成功')
    emit('close')
  } finally {
    loadingId.value = null
  }
}

function formatTime(iso: string): string {
  return new Date(iso).toLocaleString()
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / 1024 / 1024).toFixed(2)} MB`
}

// 打开弹窗时刷新列表
watch(
  () => props.show,
  (v) => {
    if (v) refresh()
  },
)
</script>

<template>
  <n-modal
    :show="show"
    preset="card"
    title="游戏存档"
    style="width: 480px; max-width: 92vw"
    @update:show="(v: boolean) => !v && emit('close')"
  >
    <div class="save-state-dialog">
      <n-spin :show="loading">
        <n-empty v-if="!loading && saveStates.length === 0" description="暂无存档" />
        <ul v-else class="save-list">
          <li v-for="ss in saveStates" :key="ss.id" class="save-item">
            <div class="save-info">
              <span class="save-time">{{ formatTime(ss.created_at) }}</span>
              <span class="save-meta">{{ ss.emulator_type.toUpperCase() }} · {{ formatSize(ss.size) }}</span>
            </div>
            <n-popconfirm @positive-click="handleLoad(ss.id)">
              <template #trigger>
                <n-button size="small" type="primary" :loading="loadingId === ss.id"> 读取 </n-button>
              </template>
              读取此存档将覆盖当前游戏进度，确认继续？
            </n-popconfirm>
          </li>
        </ul>
      </n-spin>
    </div>
  </n-modal>
</template>

<style scoped>
.save-state-dialog {
  min-height: 120px;
}

.save-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-height: 50vh;
  overflow-y: auto;
}

.save-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 12px;
  background: var(--color-bg-tertiary);
  border-radius: var(--radius-md);
}

.save-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.save-time {
  font-size: var(--font-size-small);
  color: var(--color-text-primary);
  font-weight: 600;
}

.save-meta {
  font-size: var(--font-size-mini);
  color: var(--color-text-tertiary);
  font-family: monospace;
}
</style>
