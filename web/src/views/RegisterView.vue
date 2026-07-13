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

/* ── 注册步骤：0=填表注册, 1=邮箱验证 ── */
const step = ref<0 | 1>(0)
const formRef = ref<FormInst | null>(null)
const loading = ref(false)

const model = reactive({
  username: '',
  email: '',
  password: '',
  confirmPassword: '',
  verifyCode: '',
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

/* ── 注册表单校验 ── */
const registerRules: FormRules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 3, max: 20, message: '用户名 3-20 个字符', trigger: 'blur' },
    {
      pattern: /^[a-zA-Z0-9_]+$/,
      message: '仅支持字母、数字、下划线',
      trigger: 'blur',
    },
  ],
  email: [
    { required: true, message: '请输入邮箱', trigger: 'blur' },
    { type: 'email', message: '邮箱格式不正确', trigger: 'blur' },
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 8, message: '密码至少 8 位', trigger: 'blur' },
    {
      pattern: /^(?=.*[A-Za-z])(?=.*\d).+$/,
      message: '需包含字母和数字',
      trigger: 'blur',
    },
  ],
  confirmPassword: [
    { required: true, message: '请再次输入密码', trigger: 'blur' },
    {
      validator: (_rule, value: string) => value === model.password,
      message: '两次密码不一致',
      trigger: ['blur', 'input'],
    },
  ],
}

/* ── 提交注册 → 进入邮箱验证 ── */
async function handleRegister() {
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
    const { data } = await authApi.register({
      username: model.username,
      email: model.email,
      password: model.password,
      captcha_key: captchaKey.value,
    })

    if (data.code !== 0) {
      message.error(data.message || '注册失败')
      captchaVerified.value = false
      return
    }

    message.success('验证码已发送至邮箱')
    step.value = 1
  } catch (e: any) {
    const msg = e?.response?.data?.message || '网络错误，请稍后重试'
    message.error(msg)
    captchaVerified.value = false
  } finally {
    loading.value = false
  }
}

/* ── 验证码校验 ── */
const verifyRules: FormRules = {
  verifyCode: { required: true, message: '请输入邮箱验证码', trigger: 'blur' },
}

async function handleResend() {
  try {
    await authApi.resendCode({ email: model.email })
    message.success('验证码已重新发送')
  } catch {
    message.error('重发失败')
  }
}

async function handleVerify() {
  if (!model.verifyCode.trim()) {
    message.warning('请输入验证码')
    return
  }

  loading.value = true
  try {
    const { data } = await authApi.verifyEmail({
      email: model.email,
      code: model.verifyCode,
    })

    if (data.code !== 0) {
      message.error(data.message || '验证失败')
      return
    }

    message.success('注册成功，请登录')
    router.push('/login')
  } catch (e: any) {
    const msg = e?.response?.data?.message || '验证失败'
    message.error(msg)
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="auth-page">
    <div class="auth-card">
      <!-- 卡片头 -->
      <div class="auth-card-header">
        <h1 class="auth-logo">CloudEmu</h1>
        <p class="auth-subtitle">创建账号，开始云端怀旧之旅</p>
      </div>

      <!-- 步骤 0：注册表单 -->
      <template v-if="step === 0">
        <n-form ref="formRef" :model="model" :rules="registerRules" class="auth-form">
          <n-form-item path="username">
            <n-input
              v-model:value="model.username"
              placeholder="用户名"
              size="large"
              maxlength="20"
              :input-props="{ autocomplete: 'username' }"
            />
          </n-form-item>

          <n-form-item path="email">
            <n-input
              v-model:value="model.email"
              placeholder="邮箱"
              size="large"
              :input-props="{ autocomplete: 'email' }"
            />
          </n-form-item>

          <n-form-item path="password">
            <n-input
              v-model:value="model.password"
              type="password"
              placeholder="密码（8位以上，含字母和数字）"
              size="large"
              show-password-on="click"
              :input-props="{ autocomplete: 'new-password' }"
              @keyup.enter="handleRegister"
            />
          </n-form-item>

          <n-form-item path="confirmPassword">
            <n-input
              v-model:value="model.confirmPassword"
              type="password"
              placeholder="确认密码"
              size="large"
              show-password-on="click"
              :input-props="{ autocomplete: 'new-password' }"
              @keyup.enter="handleRegister"
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

          <n-button type="primary" block size="large" :loading="loading" class="register-btn" @click="handleRegister">
            注 册
          </n-button>
        </n-form>
      </template>

      <!-- 步骤 1：邮箱验证 -->
      <template v-else>
        <n-form :model="model" :rules="verifyRules" class="auth-form">
          <div class="verify-tip">
            验证码已发送至 <strong>{{ model.email }}</strong>
          </div>

          <n-form-item path="verifyCode">
            <n-input
              v-model:value="model.verifyCode"
              placeholder="请输入 6 位验证码"
              size="large"
              maxlength="6"
              @keyup.enter="handleVerify"
            />
          </n-form-item>

          <n-button type="primary" block size="large" :loading="loading" @click="handleVerify">
            验证并激活
          </n-button>

          <div class="resend-row">
            <n-button text type="primary" @click="handleResend">重新发送验证码</n-button>
            <n-button text type="default" @click="step = 0">返回修改信息</n-button>
          </div>
        </n-form>
      </template>

      <!-- 底部链接 -->
      <div class="auth-card-footer">
        已有账号？
        <router-link to="/login">立即登录</router-link>
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

/* ── 卡片 ── */
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

/* ── 表单 ── */
.auth-form :deep(.n-form-item) {
  margin-bottom: 18px;
}

.auth-form :deep(.n-form-item:nth-last-child(2)) {
  margin-bottom: 24px;
}

.register-btn {
  margin-top: 16px;
}

/* ── 验证提示 ── */
.verify-tip {
  text-align: center;
  font-size: 14px;
  color: var(--color-text-secondary);
  margin-bottom: 20px;
  line-height: 1.6;
}

.resend-row {
  display: flex;
  justify-content: center;
  gap: 16px;
  margin-top: 16px;
}

/* ── 底部 ── */
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
