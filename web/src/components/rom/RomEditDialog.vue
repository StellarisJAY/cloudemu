<script setup lang="ts">
import { ref, watch } from 'vue'
import { useMessage } from 'naive-ui'
import { useRomStore } from '@/stores/rom'
import type { Rom, EmulatorType } from '@/types/api'

const props = defineProps<{
  show: boolean
  rom: Rom | null
}>()

const emit = defineEmits<{
  close: []
  updated: []
}>()

const message = useMessage()
const romStore = useRomStore()

const title = ref('')
const coverFile = ref<File | null>(null)
const coverFileName = ref('')
const coverFileInput = ref<HTMLInputElement | null>(null)
const submitting = ref(false)

watch(
  () => props.rom,
  (rom) => {
    if (rom) {
      title.value = rom.title
    }
  },
  { immediate: true },
)

const emulatorLabels: Record<EmulatorType, string> = {
  nes: 'NES',
  gba: 'GBC/GBA',
  dos: 'DOS',
}

function coverUrl(coverPath: string | null): string {
  if (coverPath) return `/api/files/${coverPath}`
  return ''
}

function handleCoverChange(e: Event) {
  const input = e.target as HTMLInputElement
  if (input.files && input.files.length > 0) {
    coverFile.value = input.files[0]
    coverFileName.value = input.files[0].name
  }
}

async function handleSubmit() {
  if (!props.rom) return
  if (!title.value.trim()) {
    message.warning('请输入 ROM 名称')
    return
  }

  const formData = new FormData()
  formData.append('title', title.value.trim())
  if (coverFile.value) {
    formData.append('cover', coverFile.value)
  }

  submitting.value = true
  const err = await romStore.updateRom(props.rom.id, formData)
  submitting.value = false

  if (err) {
    message.error(err)
    return
  }

  message.success('ROM 更新成功')
  emit('updated')
}

function handleClose() {
  coverFile.value = null
  coverFileName.value = ''
  emit('close')
}
</script>

<template>
  <n-modal
    :show="show"
    preset="card"
    title="编辑 ROM"
    style="width: 440px"
    :mask-closable="false"
    @update:show="(v: boolean) => !v && handleClose()"
  >
    <n-form v-if="rom" label-placement="top" class="edit-rom-form">
      <n-form-item label="当前封面">
        <img
          v-if="rom.cover_path"
          :src="coverUrl(rom.cover_path)"
          :alt="rom.title"
          class="current-cover"
        />
        <span v-else class="no-cover-text">未设置封面</span>
      </n-form-item>

      <n-form-item label="模拟器类型">
        <n-tag :bordered="false" size="small">
          {{ emulatorLabels[rom.emulator_type] }}
        </n-tag>
      </n-form-item>

      <n-form-item label="ROM 名称" required>
        <n-input v-model:value="title" placeholder="输入 ROM 名称" maxlength="255" />
      </n-form-item>

      <n-form-item label="更换封面图片（可选）">
        <div class="file-input-row">
          <n-button size="small" @click="coverFileInput?.click()"> 选择图片 </n-button>
          <span class="file-name">{{ coverFileName || '未选择文件' }}</span>
          <input
            ref="coverFileInput"
            type="file"
            accept="image/*"
            style="display: none"
            @change="handleCoverChange"
          />
        </div>
      </n-form-item>
    </n-form>

    <template #footer>
      <div class="form-footer">
        <n-button @click="handleClose">取消</n-button>
        <n-button type="primary" :loading="submitting" @click="handleSubmit"> 保存 </n-button>
      </div>
    </template>
  </n-modal>
</template>

<style scoped>
.edit-rom-form {
  padding-top: 8px;
}

.current-cover {
  width: 160px;
  height: 100px;
  object-fit: cover;
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border);
}

.no-cover-text {
  font-size: var(--font-size-small);
  color: var(--color-text-tertiary);
}

.file-input-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.file-name {
  font-size: var(--font-size-small);
  color: var(--color-text-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 200px;
}

.form-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
