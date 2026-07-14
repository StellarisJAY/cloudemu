<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage, type FormInst, type FormRules, type UploadFileInfo } from 'naive-ui'
import { useAuthStore } from '@/stores/auth'
import { fileUrl } from '@/utils/url'
import AvatarCropperDialog from '@/components/common/AvatarCropperDialog.vue'

const auth = useAuthStore()
const router = useRouter()
const message = useMessage()

/* ── 个人信息 ── */
const profileFormRef = ref<FormInst | null>(null)
const profileLoading = ref(false)

const profileModel = ref({
  nickname: '',
  bio: '',
})

const profileRules: FormRules = {
  nickname: { max: 64, message: '昵称最多 64 个字符', trigger: 'blur' },
  bio: { max: 512, message: '简介最多 512 个字符', trigger: 'blur' },
}

/** 头像文件（上传前暂存） */
const avatarFile = ref<File | null>(null)
const avatarPreview = ref<string>('')

/** 当前展示的头像 URL：优先用上传预览（blob URL），其次用服务端头像；都没有则返回空串 */
function getAvatarUrl(): string {
  if (avatarPreview.value) return avatarPreview.value
  return fileUrl(auth.user?.avatar)
}

/* ── 密码修改 ── */
const passwordFormRef = ref<FormInst | null>(null)
const passwordLoading = ref(false)

const passwordModel = ref({
  old_password: '',
  new_password: '',
  confirm_password: '',
})

function validateConfirmPassword(_rule: any, value: string): boolean {
  if (value !== passwordModel.value.new_password) {
    return new Error('两次密码输入不一致') as any
  }
  return true
}

const passwordRules: FormRules = {
  old_password: [{ required: true, message: '请输入当前密码', trigger: 'blur' }],
  new_password: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 6, message: '密码至少 6 位', trigger: 'blur' },
    { max: 128, message: '密码最多 128 位', trigger: 'blur' },
  ],
  confirm_password: [
    { required: true, message: '请确认新密码', trigger: 'blur' },
    { validator: validateConfirmPassword, trigger: ['blur', 'input'] },
  ],
}

/* ── 头像上传 ── */
/** 选中的原始图片，待送入裁剪弹窗 */
const cropSource = ref<File | null>(null)
const showCropper = ref(false)

/** 选择图片后不直接使用，先打开裁剪弹窗 */
function handleAvatarChange(options: { file: UploadFileInfo; fileList: UploadFileInfo[] }) {
  const file = options.file.file
  if (!file) return
  cropSource.value = file
  showCropper.value = true
}

/** 裁剪完成：暂存裁剪后的文件并生成预览 */
function handleAvatarCropped(file: File) {
  handleAvatarRemove()
  avatarFile.value = file
  avatarPreview.value = URL.createObjectURL(file)
  showCropper.value = false
  cropSource.value = null
}

function handleCropperClose() {
  showCropper.value = false
  cropSource.value = null
}

function handleAvatarRemove() {
  avatarFile.value = null
  if (avatarPreview.value) {
    URL.revokeObjectURL(avatarPreview.value)
    avatarPreview.value = ''
  }
}

/* ── 保存个人信息 ── */
async function handleSaveProfile() {
  try {
    await profileFormRef.value!.validate()
  } catch {
    return
  }

  profileLoading.value = true
  try {
    const fd = new FormData()
    fd.append('nickname', profileModel.value.nickname)
    fd.append('bio', profileModel.value.bio)
    if (avatarFile.value) {
      fd.append('avatar', avatarFile.value)
    }

    const err = await auth.updateProfile(fd)
    if (err) {
      message.error(err)
    } else {
      message.success('个人信息已保存')
    }
  } finally {
    profileLoading.value = false
  }
}

/* ── 修改密码 ── */
async function handleChangePassword() {
  try {
    await passwordFormRef.value!.validate()
  } catch {
    return
  }

  passwordLoading.value = true
  try {
    const err = await auth.updatePassword({
      old_password: passwordModel.value.old_password,
      new_password: passwordModel.value.new_password,
    })
    if (err) {
      message.error(err)
    } else {
      message.success('密码已修改')
      passwordModel.value = { old_password: '', new_password: '', confirm_password: '' }
    }
  } finally {
    passwordLoading.value = false
  }
}

/* ── 初始化 ── */
onMounted(async () => {
  if (!auth.user) await auth.fetchUser()
  if (auth.user) {
    profileModel.value.nickname = auth.user.nickname || ''
    profileModel.value.bio = auth.user.bio || ''
  }
})
</script>

<template>
  <div class="auth-page">
    <div class="auth-card">
      <!-- 卡片头 — 与登录页一致 -->
      <div class="auth-card-header">
        <h1 class="auth-logo">个人设置</h1>
        <p class="auth-subtitle">{{ auth.user?.username }}</p>
      </div>

      <!-- 头像区 -->
      <div class="avatar-section">
        <!-- 有头像 URL（本地预览或服务端路径）走 src + fallback slot；无 URL 直接 default slot 显示首字母 -->
        <n-avatar
          v-if="getAvatarUrl()"
          :size="96"
          :src="getAvatarUrl()"
          round
          class="avatar-img"
        >
          <template #fallback>
            {{ auth.user?.username?.charAt(0)?.toUpperCase() || '?' }}
          </template>
        </n-avatar>
        <n-avatar v-else :size="96" round class="avatar-img avatar-placeholder">
          {{ auth.user?.username?.charAt(0)?.toUpperCase() || '?' }}
        </n-avatar>
        <div class="avatar-actions">
          <n-upload accept="image/*" :max="1" :show-file-list="false" @change="handleAvatarChange">
            <n-button
              size="small"
              :secondary="!avatarFile"
              :type="avatarFile ? 'primary' : 'default'"
            >
              {{ avatarFile ? '更换头像' : '上传头像' }}
            </n-button>
          </n-upload>
          <n-button
            v-if="avatarFile"
            size="small"
            secondary
            type="error"
            @click="handleAvatarRemove"
          >
            移除
          </n-button>
        </div>
      </div>

      <!-- 头像裁剪弹窗 -->
      <AvatarCropperDialog
        :show="showCropper"
        :file="cropSource"
        @cropped="handleAvatarCropped"
        @close="handleCropperClose"
      />

      <!-- 个人信息表单 -->
      <n-form ref="profileFormRef" :model="profileModel" :rules="profileRules" class="auth-form">
        <n-form-item path="nickname">
          <n-input
            v-model:value="profileModel.nickname"
            placeholder="昵称（不填则显示用户名）"
            size="large"
            maxlength="64"
            show-count
          />
        </n-form-item>

        <n-form-item path="bio">
          <n-input
            v-model:value="profileModel.bio"
            type="textarea"
            placeholder="简单介绍一下自己..."
            size="large"
            maxlength="512"
            show-count
            :rows="3"
          />
        </n-form-item>

        <n-button
          type="primary"
          block
          size="large"
          :loading="profileLoading"
          class="submit-btn"
          @click="handleSaveProfile"
        >
          保存个人信息
        </n-button>
      </n-form>

      <!-- 分隔 -->
      <div class="section-divider">
        <span>修改密码</span>
      </div>

      <!-- 修改密码表单 -->
      <n-form ref="passwordFormRef" :model="passwordModel" :rules="passwordRules" class="auth-form">
        <n-form-item path="old_password">
          <n-input
            v-model:value="passwordModel.old_password"
            type="password"
            placeholder="当前密码"
            size="large"
            show-password-on="click"
            :input-props="{ autocomplete: 'current-password' }"
          />
        </n-form-item>

        <n-form-item path="new_password">
          <n-input
            v-model:value="passwordModel.new_password"
            type="password"
            placeholder="新密码（至少 6 位）"
            size="large"
            show-password-on="click"
            :input-props="{ autocomplete: 'new-password' }"
          />
        </n-form-item>

        <n-form-item path="confirm_password">
          <n-input
            v-model:value="passwordModel.confirm_password"
            type="password"
            placeholder="确认新密码"
            size="large"
            show-password-on="click"
            :input-props="{ autocomplete: 'new-password' }"
          />
        </n-form-item>

        <n-button
          type="warning"
          block
          size="large"
          :loading="passwordLoading"
          class="submit-btn"
          @click="handleChangePassword"
        >
          修改密码
        </n-button>
      </n-form>

      <!-- 底部 — 与登录页一致 -->
      <div class="auth-card-footer">
        <router-link to="/">← 返回大厅</router-link>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* ── 页面容器 — 与 LoginView 完全一致 ── */
.auth-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  background: linear-gradient(135deg, var(--color-bg-primary) 0%, var(--color-bg-tertiary) 100%);
}

/* ── 卡片 — 与 LoginView 完全一致 ── */
.auth-card {
  position: relative;
  z-index: 1;
  width: 400px;
  max-width: 100%;
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: 40px 36px 28px;
  backdrop-filter: blur(8px);
  box-shadow: var(--shadow-lg);
}

/* ── 卡片头 — 与 LoginView 完全一致 ── */
.auth-card-header {
  text-align: center;
  margin-bottom: 32px;
}

.auth-logo {
  font-size: 32px;
  font-weight: 700;
  color: var(--color-accent);
  margin: 0 0 6px;
  letter-spacing: 1px;
}

.auth-subtitle {
  font-size: 14px;
  color: var(--color-text-secondary);
  margin: 0;
}

/* ── 头像 ── */
.avatar-section {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  margin-bottom: 28px;
}

.avatar-img {
  border: 2px solid var(--color-border);
}

.avatar-placeholder {
  background: var(--color-accent);
  color: #fff;
  font-size: 34px;
  font-weight: 600;
}

.avatar-actions {
  display: flex;
  gap: 8px;
}

/* ── 表单 ── */
.auth-form :deep(.n-form-item) {
  margin-bottom: 18px;
}

.submit-btn {
  margin-top: 16px;
}

/* ── 分隔线 ── */
.section-divider {
  display: flex;
  align-items: center;
  gap: 12px;
  margin: 28px 0 24px;
  color: var(--color-text-secondary);
  font-size: 13px;
}

.section-divider::before,
.section-divider::after {
  content: '';
  flex: 1;
  height: 1px;
  background: var(--color-divider);
}

/* ── 底部 — 与 LoginView 完全一致 ── */
.auth-card-footer {
  text-align: center;
  margin-top: 20px;
  font-size: 13px;
}

.auth-card-footer a {
  color: var(--color-accent);
  text-decoration: none;
}

.auth-card-footer a:hover {
  text-decoration: underline;
}
</style>
