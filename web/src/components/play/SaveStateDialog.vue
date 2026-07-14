<script setup lang="ts">
import { ref, watch } from 'vue'
import {
  useMessage,
  NModal,
  NButton,
  NEmpty,
  NSpin,
  NPopconfirm,
  NInput,
  NSpace,
} from 'naive-ui'
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
const deletingId = ref<string | null>(null)

// 重命名状态
const editingId = ref<string | null>(null)
const editingName = ref('')
const savingName = ref(false)

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

function startRename(ss: SaveState) {
  editingId.value = ss.id
  editingName.value = ss.name
}

function cancelRename() {
  editingId.value = null
  editingName.value = ''
}

async function confirmRename(saveStateId: string) {
  const name = editingName.value.trim()
  if (!name) {
    message.warning('名称不能为空')
    return
  }
  savingName.value = true
  try {
    const err = await roomStore.renameSaveState(props.roomId, saveStateId, name)
    if (err) {
      message.error(err)
      return
    }
    const target = saveStates.value.find((s) => s.id === saveStateId)
    if (target) target.name = name
    message.success('已重命名')
    cancelRename()
  } finally {
    savingName.value = false
  }
}

async function handleDelete(saveStateId: string) {
  deletingId.value = saveStateId
  try {
    const err = await roomStore.deleteSaveState(props.roomId, saveStateId)
    if (err) {
      message.error(err)
      return
    }
    saveStates.value = saveStates.value.filter((s) => s.id !== saveStateId)
    message.success('已删除')
  } finally {
    deletingId.value = null
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
    if (v) {
      cancelRename()
      refresh()
    }
  },
)
</script>

<template>
  <n-modal
    :show="show"
    preset="card"
    title="游戏存档"
    style="width: 520px; max-width: 92vw"
    @update:show="(v: boolean) => !v && emit('close')"
  >
    <div class="save-state-dialog">
      <n-spin :show="loading">
        <n-empty v-if="!loading && saveStates.length === 0" description="暂无存档" />
        <ul v-else class="save-list">
          <li v-for="ss in saveStates" :key="ss.id" class="save-item">
            <div class="save-info">
              <template v-if="editingId === ss.id">
                <n-input
                  v-model:value="editingName"
                  size="small"
                  maxlength="64"
                  placeholder="存档名称"
                  @keyup.enter="confirmRename(ss.id)"
                />
              </template>
              <template v-else>
                <span class="save-name">{{ ss.name }}</span>
                <span class="save-meta">
                  {{ ss.emulator_type.toUpperCase() }} · {{ formatSize(ss.size) }} ·
                  {{ formatTime(ss.created_at) }}
                </span>
              </template>
            </div>

            <n-space :size="6" class="save-actions">
              <template v-if="editingId === ss.id">
                <n-button size="small" type="primary" :loading="savingName" @click="confirmRename(ss.id)">
                  保存
                </n-button>
                <n-button size="small" tertiary @click="cancelRename">取消</n-button>
              </template>
              <template v-else>
                <n-popconfirm @positive-click="handleLoad(ss.id)">
                  <template #trigger>
                    <n-button size="small" type="primary" :loading="loadingId === ss.id"> 读取 </n-button>
                  </template>
                  读取此存档将覆盖当前游戏进度，确认继续？
                </n-popconfirm>
                <n-button size="small" tertiary @click="startRename(ss)">改名</n-button>
                <n-popconfirm @positive-click="handleDelete(ss.id)">
                  <template #trigger>
                    <n-button size="small" tertiary type="error" :loading="deletingId === ss.id">
                      删除
                    </n-button>
                  </template>
                  删除后无法恢复，确认删除此存档？
                </n-popconfirm>
              </template>
            </n-space>
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
  gap: 12px;
  padding: 10px 12px;
  background: var(--color-bg-tertiary);
  border-radius: var(--radius-md);
}

.save-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
  flex: 1;
  min-width: 0;
}

.save-name {
  font-size: var(--font-size-small);
  color: var(--color-text-primary);
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.save-meta {
  font-size: var(--font-size-mini);
  color: var(--color-text-tertiary);
  font-family: monospace;
}

.save-actions {
  flex-shrink: 0;
}
</style>
