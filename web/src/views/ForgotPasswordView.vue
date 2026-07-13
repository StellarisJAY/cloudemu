<script setup lang="ts">
import { ref, reactive, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage, type FormInst, type FormRules } from 'naive-ui'
import { Slide } from 'go-captcha-vue'
import 'go-captcha-vue/dist/style.css'
import { authApi } from '@/api/auth'
import type { CaptchaResp, VerifyCaptchaReq } from '@/types/api'

/** go-captcha-vue 官方组件 confirm 事件的点位参数 */
interface SlidePoint {
  x: number
  y: number
}

const router = useRouter()
const message = useMessage()

const formRef = ref<FormInst | null>(null)
const loading = ref(false)
const submitted = ref(false)
const submittedEmail = ref('')
const cooldown = ref(0)

const model = reactive({
  email: '',
})

/* ── 滑块验证码（弹窗模式，阶段1：滑动后立即服务端校验）── */
const showCaptchaModal = ref(false)
const captchaVerified = ref(false)
const captchaKey = ref('')
const captchaData = ref<CaptchaResp | null>(null)
const captchaLoading = ref(false)
const verifying = ref(false)
const slideRef = ref<InstanceType<typeof Slide> | null>(null)

const slideData = computed(() => {
  if (!captchaData.value)
    return { image: '', thumb: '', thumbX: 0, thumbY: 0, thumbWidth: 0, thumbHeight: 0 }
  return {
    image: captchaData.value.master_bg_base64,
    thumb: captchaData.value.tile_base64,
    thumbX: captchaData.value.thumb_x,
    thumbY: captchaData.value.thumb_y,
    thumbWidth: captchaData.value.tile_width,
    thumbHeight: captchaData.value.tile_height,
  }
})

const slideConfig = {
  width: 300,
  height: 200,
  showTheme: false,
}

async function onCaptchaConfirm(point: SlidePoint, reset: () => void) {
  if (!captchaKey.value || verifying.value) return
  verifying.value = true
  try {
    await authApi.verifyCaptcha({
      captcha_key: captchaKey.value,
      slide_x: point.x,
      slide_y: point.y,
    } as VerifyCaptchaReq)
    showCaptchaModal.value = false
    captchaVerified.value = true
  } catch (e: any) {
    message.error(e?.response?.data?.message || '验证失败，请重试')
    reset()
  } finally {
    verifying.value = false
  }
}

const slideEvents = {
  confirm: onCaptchaConfirm,
  refresh: () => fetchCaptcha(),
  close: () => {
    showCaptchaModal.value = false
  },
}

async function fetchCaptcha() {
  captchaLoading.value = true
  captchaData.value = null
  try {
    const { data } = await authApi.captcha()
    if (data.data) {
      captchaData.value = data.data
      captchaKey.value = data.data.captcha_key
    }
  } catch {
    message.error('获取验证码失败')
  } finally {
    captchaLoading.value = false
  }
}

function openCaptcha() {
  showCaptchaModal.value = true
  fetchCaptcha()
}

const rules: FormRules = {
  email: [
    { required: true, message: '请输入注册邮箱', trigger: 'blur' },
    { type: 'email', message: '请输入有效的邮箱地址', trigger: 'blur' },
  ],
}

async function handleSubmit() {
  try {
    await formRef.value!.validate()
  } catch {
    return
  }

  if (!captchaVerified.value) {
    message.warning('请先完成安全验证')
    return
  }

  loading.value = true
  try {
    await authApi.forgotPassword({ email: model.email, captcha_key: captchaKey.value })
    submittedEmail.value = model.email
    submitted.value = true
    cooldown.value = 60
    captchaVerified.value = false
    const timer = setInterval(() => {
      cooldown.value--
      if (cooldown.value <= 0) clearInterval(timer)
    }, 1000)
  } catch (e: any) {
    const msg = e?.response?.data?.message || '网络错误，请稍后重试'
    message.error(msg)
    captchaVerified.value = false
  } finally {
    loading.value = false
  }
}

function handleResend() {
  submitted.value = false
  cooldown.value = 0
  captchaVerified.value = false
}
</script>

<template>
  <div class="auth-page">
    <div class="auth-card">
      <div class="auth-card-header">
        <h1 class="auth-logo">CloudEmu</h1>
        <p class="auth-subtitle">重置密码</p>
      </div>

      <!-- 已提交：显示成功提示 -->
      <div v-if="submitted" class="forgot-success">
        <p class="forgot-success-icon">&#10003;</p>
        <p class="forgot-success-title">重置链接已发送</p>
        <p class="forgot-success-desc">一封密码重置邮件已发送至 <strong>{{ submittedEmail }}</strong>，请查收邮件并点击重置链接。</p>
        <n-button
          v-if="cooldown > 0"
          block
          size="large"
          disabled
          class="resend-btn"
        >
          {{ cooldown }} 秒后可重新发送
        </n-button>
        <n-button
          v-else
          block
          size="large"
          @click="handleResend"
          class="resend-btn"
        >
          未收到邮件？重新发送
        </n-button>
      </div>

      <!-- 表单 -->
      <n-form v-else ref="formRef" :model="model" :rules="rules" class="auth-form">
        <n-form-item path="email">
          <n-input
            v-model:value="model.email"
            placeholder="请输入注册邮箱"
            size="large"
            :input-props="{ autocomplete: 'email' }"
            @keyup.enter="handleSubmit"
          />
        </n-form-item>

        <!-- 验证按钮 -->
        <n-button
          block
          size="large"
          :type="captchaVerified ? 'success' : 'default'"
          :secondary="!captchaVerified"
          :loading="verifying"
          @click="openCaptcha"
        >
          <template v-if="verifying">验证中...</template>
          <template v-else-if="captchaVerified">安全验证已完成</template>
          <template v-else>请完成安全验证</template>
        </n-button>

        <n-button
          type="primary"
          block
          size="large"
          :loading="loading"
          class="submit-btn"
          @click="handleSubmit"
        >
          发送重置链接
        </n-button>
      </n-form>

      <div class="auth-card-footer">
        <router-link to="/login">返回登录</router-link>
      </div>
    </div>

    <!-- 滑块验证弹窗 -->
    <n-modal
      v-model:show="showCaptchaModal"
      preset="card"
      title="安全验证"
      :mask-closable="false"
      class="captcha-modal"
      style="width: auto; max-width: 400px"
    >
      <div v-if="captchaLoading" class="captcha-modal-loading">加载中...</div>
      <template v-else-if="captchaData">
        <Slide ref="slideRef" :data="slideData" :config="slideConfig" :events="slideEvents" />
      </template>
      <div v-else class="captcha-modal-loading">
        <n-button text type="warning" @click="fetchCaptcha">加载失败，点此重试</n-button>
      </div>

      <template #footer>
        <n-button text type="primary" size="small" @click="fetchCaptcha" :disabled="captchaLoading">
          换一张
        </n-button>
      </template>
    </n-modal>
  </div>
</template>

<style scoped>
.auth-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  background: linear-gradient(135deg, var(--color-bg-primary) 0%, var(--color-bg-tertiary) 100%);
}

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

.auth-form :deep(.n-form-item) {
  margin-bottom: 18px;
}

.submit-btn {
  margin-top: 16px;
}

.forgot-success {
  text-align: center;
  margin-bottom: 16px;
}

.forgot-success-icon {
  font-size: 48px;
  color: var(--color-accent);
  margin: 0 0 12px;
}

.forgot-success-title {
  font-size: 18px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin: 0 0 8px;
}

.forgot-success-desc {
  font-size: 14px;
  color: var(--color-text-secondary);
  line-height: 1.6;
  margin: 0 0 20px;
}

.forgot-success-desc strong {
  color: var(--color-text-primary);
}

.resend-btn {
  margin-top: 8px;
}

.auth-card-footer {
  text-align: center;
  margin-top: 20px;
  font-size: 13px;
  color: var(--color-text-secondary);
}

.auth-card-footer a {
  color: var(--color-accent);
  text-decoration: none;
  margin-left: 2px;
}

.auth-card-footer a:hover {
  text-decoration: underline;
}

/* ── 验证弹窗 ── */
.captcha-modal :deep(.n-card__content) {
  display: flex;
  justify-content: center;
  padding: 16px 0;
}

.captcha-modal-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 260px;
  font-size: 14px;
  color: var(--color-text-secondary);
}
</style>
