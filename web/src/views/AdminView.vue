<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import { useAdminStore } from '@/stores/admin'
import type { Rom, EmulatorType } from '@/types/api'

const router = useRouter()
const message = useMessage()
const adminStore = useAdminStore()

const emulatorLabels: Record<EmulatorType, string> = {
  nes: 'NES',
  gb: 'GBC/GBA',
  dos: 'DOS',
}

// ── 上传 / 编辑弹窗 ──
const showDialog = ref(false)
const editingRom = ref<Rom | null>(null) // null=上传新内置ROM，非null=编辑
const title = ref('')
const romFile = ref<File | null>(null)
const romFileName = ref('')
const coverFile = ref<File | null>(null)
const coverFileName = ref('')
const romFileInput = ref<HTMLInputElement | null>(null)
const coverFileInput = ref<HTMLInputElement | null>(null)
const submitting = ref(false)

// ── 删除确认 ──
const showDeleteConfirm = ref(false)
const pendingDelete = ref<Rom | null>(null)

function openUpload() {
  editingRom.value = null
  title.value = ''
  romFile.value = null
  romFileName.value = ''
  coverFile.value = null
  coverFileName.value = ''
  showDialog.value = true
}

function openEdit(rom: Rom) {
  editingRom.value = rom
  title.value = rom.title
  romFile.value = null
  romFileName.value = ''
  coverFile.value = null
  coverFileName.value = ''
  showDialog.value = true
}

function handleRomChange(e: Event) {
  const input = e.target as HTMLInputElement
  const f = input.files?.[0]
  if (f) {
    romFile.value = f
    romFileName.value = f.name
  }
}

function handleCoverChange(e: Event) {
  const input = e.target as HTMLInputElement
  const f = input.files?.[0]
  if (f) {
    coverFile.value = f
    coverFileName.value = f.name
  }
}

async function handleSubmit() {
  if (!title.value.trim()) {
    message.warning('请输入 ROM 名称')
    return
  }
  const isEdit = editingRom.value !== null
  if (!isEdit && !romFile.value) {
    message.warning('请选择 ROM 文件')
    return
  }

  const formData = new FormData()
  formData.append('title', title.value.trim())
  if (romFile.value) formData.append('rom', romFile.value)
  if (coverFile.value) formData.append('cover', coverFile.value)

  submitting.value = true
  const err = isEdit
    ? await adminStore.updateBuiltin(editingRom.value!.id, formData)
    : await adminStore.uploadBuiltin(formData)
  submitting.value = false

  if (err) {
    message.error(err)
    return
  }
  message.success(isEdit ? '内置 ROM 已更新' : '内置 ROM 上传成功')
  showDialog.value = false
}

function handleDelete(rom: Rom) {
  pendingDelete.value = rom
  showDeleteConfirm.value = true
}

async function confirmDelete() {
  if (!pendingDelete.value) return
  const err = await adminStore.deleteBuiltin(pendingDelete.value.id)
  if (err) {
    message.error(err)
  } else {
    message.success('内置 ROM 已删除')
  }
  showDeleteConfirm.value = false
  pendingDelete.value = null
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

onMounted(() => {
  adminStore.fetchBuiltinRoms()
})
</script>

<template>
  <div class="admin-layout">
    <!-- 顶栏 -->
    <header class="admin-header">
      <div class="header-left">
        <h1 class="header-logo">CloudEmu 管理后台</h1>
      </div>
      <div class="header-right">
        <n-button size="small" text @click="router.push('/')"> 返回大厅 </n-button>
      </div>
    </header>

    <main class="admin-main">
      <div class="section-header">
        <h2 class="section-title">平台内置 ROM</h2>
        <n-button type="primary" size="small" @click="openUpload"> + 上传内置 ROM </n-button>
      </div>

      <div v-if="adminStore.loading" class="section-loading">加载中...</div>

      <div v-else-if="adminStore.builtinRoms.length === 0" class="section-empty">
        <p>暂无内置 ROM</p>
        <p class="sub-text">上传平台内置 ROM，所有用户都能看到并使用</p>
      </div>

      <n-table v-else :bordered="false" :single-line="false">
        <thead>
          <tr>
            <th>名称</th>
            <th>类型</th>
            <th>大小</th>
            <th>上传时间</th>
            <th style="width: 160px">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="rom in adminStore.builtinRoms" :key="rom.id">
            <td>{{ rom.title }}</td>
            <td>{{ emulatorLabels[rom.emulator_type] }}</td>
            <td>{{ formatSize(rom.file_size) }}</td>
            <td>{{ rom.created_at }}</td>
            <td>
              <div class="row-actions">
                <n-button size="tiny" @click="openEdit(rom)">编辑</n-button>
                <n-button size="tiny" type="error" @click="handleDelete(rom)">删除</n-button>
              </div>
            </td>
          </tr>
        </tbody>
      </n-table>
    </main>

    <!-- 上传 / 编辑弹窗 -->
    <n-modal
      :show="showDialog"
      preset="card"
      :title="editingRom ? '编辑内置 ROM' : '上传内置 ROM'"
      style="width: 440px"
      :mask-closable="false"
      @update:show="(v: boolean) => (showDialog = v)"
    >
      <n-form label-placement="top" class="admin-rom-form">
        <n-form-item label="ROM 名称" required>
          <n-input v-model:value="title" placeholder="输入 ROM 名称" maxlength="255" />
        </n-form-item>

        <n-form-item v-if="!editingRom" label="ROM 文件" required>
          <div class="file-input-row">
            <n-button size="small" @click="romFileInput?.click()"> 选择文件 </n-button>
            <span class="file-name">{{ romFileName || '未选择文件' }}</span>
            <input
              ref="romFileInput"
              type="file"
              accept=".nes,.gba,.gbc,.zip"
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
          <n-button @click="showDialog = false">取消</n-button>
          <n-button type="primary" :loading="submitting" @click="handleSubmit">
            {{ editingRom ? '保存' : '上传' }}
          </n-button>
        </div>
      </template>
    </n-modal>

    <!-- 删除确认 -->
    <n-modal
      v-model:show="showDeleteConfirm"
      preset="dialog"
      title="删除内置 ROM"
      positive-text="确认删除"
      negative-text="取消"
      type="warning"
      @positive-click="confirmDelete"
      @negative-click="
        () => {
          showDeleteConfirm = false
          pendingDelete = null
        }
      "
    >
      <p>
        确定要删除内置 ROM「<strong>{{ pendingDelete?.title }}</strong>」吗？
      </p>
      <p style="color: var(--color-text-secondary); font-size: var(--font-size-small)">
        删除后所有用户将无法再看到和使用该 ROM，且文件将从存储中永久移除。
      </p>
    </n-modal>
  </div>
</template>

<style scoped>
.admin-layout {
  display: flex;
  flex-direction: column;
  height: 100vh;
  background: var(--color-bg-primary);
}

.admin-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 48px;
  padding: 0 16px;
  background: var(--color-bg-secondary);
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}

.header-logo {
  margin: 0;
  font-size: 18px;
  font-weight: 700;
  color: var(--color-accent);
  letter-spacing: 1px;
}

.admin-main {
  flex: 1;
  overflow: auto;
  padding: 24px;
  max-width: 960px;
  width: 100%;
  margin: 0 auto;
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}

.section-title {
  margin: 0;
  font-size: var(--font-size-medium);
  font-weight: 600;
  color: var(--color-text-primary);
}

.section-loading,
.section-empty {
  padding: 48px 16px;
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

.row-actions {
  display: flex;
  gap: 8px;
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
