<script setup lang="ts">
import { ref } from 'vue'
import { useMessage } from 'naive-ui'
import { useRomStore } from '@/stores/rom'
import type { EmulatorType } from '@/types/api'

defineProps<{
  show: boolean
}>()

const emit = defineEmits<{
  close: []
  uploaded: []
}>()

const message = useMessage()
const romStore = useRomStore()

const title = ref('')
const emulatorType = ref<EmulatorType>('nes')
const romFile = ref<File | null>(null)
const romFileName = ref('')
const coverFile = ref<File | null>(null)
const coverFileName = ref('')
const romFileInput = ref<HTMLInputElement | null>(null)
const coverFileInput = ref<HTMLInputElement | null>(null)
const submitting = ref(false)

const emulatorOptions = [
  { label: 'NES', value: 'nes' as const },
  { label: 'GBA', value: 'gba' as const },
  { label: 'DOS', value: 'dos' as const },
]

function handleRomChange(e: Event) {
  const input = e.target as HTMLInputElement
  if (input.files && input.files.length > 0) {
    romFile.value = input.files[0]
    romFileName.value = input.files[0].name
  }
}

function handleCoverChange(e: Event) {
  const input = e.target as HTMLInputElement
  if (input.files && input.files.length > 0) {
    coverFile.value = input.files[0]
    coverFileName.value = input.files[0].name
  }
}

async function handleSubmit() {
  if (!title.value.trim()) {
    message.warning('请输入 ROM 名称')
    return
  }
  if (!romFile.value) {
    message.warning('请选择 ROM 文件')
    return
  }

  const formData = new FormData()
  formData.append('title', title.value.trim())
  formData.append('rom', romFile.value)
  if (coverFile.value) {
    formData.append('cover', coverFile.value)
  }

  submitting.value = true
  const err = await romStore.uploadRom(formData)
  submitting.value = false

  if (err) {
    message.error(err)
    return
  }

  message.success('ROM 上传成功')
  emit('uploaded')
  resetForm()
}

function handleClose() {
  resetForm()
  emit('close')
}

function resetForm() {
  title.value = ''
  emulatorType.value = 'nes'
  romFile.value = null
  romFileName.value = ''
  coverFile.value = null
  coverFileName.value = ''
}
</script>

<template>
  <n-modal
    :show="show"
    preset="card"
    title="上传 ROM"
    style="width: 440px"
    :mask-closable="false"
    @update:show="(v: boolean) => !v && handleClose()"
  >
    <n-form label-placement="top" class="upload-rom-form">
      <n-form-item label="ROM 名称" required>
        <n-input v-model:value="title" placeholder="输入 ROM 名称" maxlength="255" />
      </n-form-item>

      <n-form-item label="模拟器类型" required>
        <n-select v-model:value="emulatorType" :options="emulatorOptions" />
      </n-form-item>

      <n-form-item label="ROM 文件" required>
        <div class="file-input-row">
          <n-button size="small" @click="romFileInput?.click()"> 选择文件 </n-button>
          <span class="file-name">{{ romFileName || '未选择文件' }}</span>
          <input
            ref="romFileInput"
            type="file"
            accept=".nes,.gba"
            style="display: none"
            @change="handleRomChange"
          />
        </div>
      </n-form-item>

      <n-form-item label="封面图片（可选）">
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
        <n-button type="primary" :loading="submitting" @click="handleSubmit"> 上传 </n-button>
      </div>
    </template>
  </n-modal>
</template>

<style scoped>
.upload-rom-form {
  padding-top: 8px;
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
