<script setup lang="ts">
import { ref, reactive, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useMessage, type FormInst, type FormRules } from 'naive-ui'
import { Slide } from 'go-captcha-vue'
import 'go-captcha-vue/dist/style.css'
import { authApi } from '@/api/auth'
import { setTokens } from '@/utils/token'
import type { CaptchaResp, VerifyCaptchaReq } from '@/types/api'

/** go-captcha-vue 官方组件 confirm 事件的点位参数 */
interface SlidePoint {
  x: number
  y: number
}

const router = useRouter()
const route = useRoute()
const message = useMessage()

const formRef = ref<FormInst | null>(null)
const slideRef = ref<InstanceType<typeof Slide> | null>(null)
const loading = ref(false)

const model = reactive({
  account: '',
  password: '',
})

/* ── 滑块验证码（弹窗模式，阶段1：滑动后立即服务端校验）── */
const showCaptchaModal = ref(false)
const captchaVerified = ref(false)
const captchaKey = ref('')
const captchaData = ref<CaptchaResp | null>(null)
const captchaLoading = ref(false)
const verifying = ref(false)

// go-captcha-vue 官方 Slide 组件所需 data prop
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

// confirm 事件：用户松手后服务端校验
async function onCaptchaConfirm(point: SlidePoint, reset: () => void) {
  if (!captchaKey.value || verifying.value) return
  verifying.value = true
  try {
    await authApi.verifyCaptcha({
      captcha_key: captchaKey.value,
      slide_x: point.x,
      slide_y: point.y,
    } as VerifyCaptchaReq)
    // 校验通过
    showCaptchaModal.value = false
    captchaVerified.value = true
  } catch (e: any) {
    // 校验失败：复位拼图块，用户可重试
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

/* ── 表单校验 ── */
const rules: FormRules = {
  account: { required: true, message: '请输入用户名或邮箱', trigger: 'blur' },
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, message: '密码至少 6 位', trigger: 'blur' },
  ],
}

/* ── 登录 ── */
async function handleLogin() {
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
    const { data } = await authApi.login({
      account: model.account,
      password: model.password,
      captcha_key: captchaKey.value,
    })

    if (data.code !== 0 || !data.data) {
      message.error(data.message || '登录失败')
      captchaVerified.value = false
      return
    }

    setTokens(data.data.access_token, data.data.refresh_token)
    message.success('登录成功')

    const redirect = (route.query.redirect as string) || '/'
    router.push(redirect)
  } catch (e: any) {
    const msg = e?.response?.data?.message || '网络错误，请稍后重试'
    message.error(msg)
    captchaVerified.value = false
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
        <p class="auth-subtitle">云端复古游戏 · 多人同玩</p>
      </div>

      <!-- 表单 -->
      <n-form ref="formRef" :model="model" :rules="rules" class="auth-form">
        <n-form-item path="account">
          <n-input
            v-model:value="model.account"
            placeholder="用户名 / 邮箱"
            size="large"
            :input-props="{ autocomplete: 'username' }"
          />
        </n-form-item>

        <n-form-item path="password">
          <n-input
            v-model:value="model.password"
            type="password"
            placeholder="密码"
            size="large"
            show-password-on="click"
            :input-props="{ autocomplete: 'current-password' }"
            @keyup.enter="handleLogin"
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
          class="login-btn"
          @click="handleLogin"
        >
          登 录
        </n-button>
      </n-form>

      <!-- 底部链接 -->
      <div class="auth-card-footer">
        <router-link to="/forgot-password">忘记密码？</router-link>
      </div>
      <div class="auth-card-footer" style="margin-top: 4px">
        还没有账号？
        <router-link to="/register">立即注册</router-link>
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

.login-btn {
  margin-top: 16px;
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
</style>
