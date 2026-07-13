<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useMessage, type FormInst, type FormRules } from 'naive-ui'
import { authApi } from '@/api/auth'

const router = useRouter()
const route = useRoute()
const message = useMessage()

const token = (route.query.token as string) || ''

const formRef = ref<FormInst | null>(null)
const loading = ref(false)
const succeeded = ref(false)

const model = reactive({
  newPassword: '',
  confirmPassword: '',
})

const rules: FormRules = {
  newPassword: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 6, message: '密码至少 6 位', trigger: 'blur' },
    { max: 128, message: '密码最多 128 位', trigger: 'blur' },
  ],
  confirmPassword: [
    { required: true, message: '请再次输入新密码', trigger: 'blur' },
    {
      validator: (_rule, value: string) => {
        if (value !== model.newPassword) {
          return new Error('两次输入的密码不一致')
        }
        return true
      },
      trigger: 'blur',
    },
  ],
}

async function handleSubmit() {
  try {
    await formRef.value!.validate()
  } catch {
    return
  }

  loading.value = true
  try {
    await authApi.resetPassword({
      token,
      new_password: model.newPassword,
    })
    message.success('密码重置成功')
    succeeded.value = true
    setTimeout(() => {
      router.push('/login')
    }, 1500)
  } catch (e: any) {
    const code = e?.response?.data?.code
    if (code === 1012 || code === 1013) {
      message.error('重置链接已过期或已被使用，请重新获取')
    } else {
      message.error(e?.response?.data?.message || '重置失败，请稍后重试')
    }
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="auth-page">
    <div class="auth-card">
      <div class="auth-card-header">
        <h1 class="auth-logo">CloudEmu</h1>
        <p class="auth-subtitle">设置新密码</p>
      </div>

      <!-- token 无效 -->
      <div v-if="!token" class="reset-error">
        <p class="reset-error-text">无效的重置链接，缺少必要参数</p>
        <n-button
          type="primary"
          block
          size="large"
          class="retry-btn"
          @click="router.push('/forgot-password')"
        >
          重新获取重置链接
        </n-button>
      </div>

      <!-- 重置成功 -->
      <div v-else-if="succeeded" class="reset-success">
        <p class="reset-success-icon">&#10003;</p>
        <p class="reset-success-text">密码已重置成功，即将跳转登录页...</p>
      </div>

      <!-- 重置表单 -->
      <n-form v-else ref="formRef" :model="model" :rules="rules" class="auth-form">
        <n-form-item path="newPassword">
          <n-input
            v-model:value="model.newPassword"
            type="password"
            placeholder="新密码（至少 6 位）"
            size="large"
            show-password-on="click"
            :input-props="{ autocomplete: 'new-password' }"
          />
        </n-form-item>

        <n-form-item path="confirmPassword">
          <n-input
            v-model:value="model.confirmPassword"
            type="password"
            placeholder="确认新密码"
            size="large"
            show-password-on="click"
            :input-props="{ autocomplete: 'new-password' }"
            @keyup.enter="handleSubmit"
          />
        </n-form-item>

        <n-button
          type="primary"
          block
          size="large"
          :loading="loading"
          class="submit-btn"
          @click="handleSubmit"
        >
          重置密码
        </n-button>
      </n-form>

      <div class="auth-card-footer">
        <router-link to="/login">返回登录</router-link>
      </div>
    </div>
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
  margin-top: 8px;
}

.reset-error {
  text-align: center;
}

.reset-error-text {
  font-size: 15px;
  color: var(--color-text-secondary);
  margin: 0 0 20px;
}

.retry-btn {
  margin-top: 8px;
}

.reset-success {
  text-align: center;
  padding: 20px 0;
}

.reset-success-icon {
  font-size: 48px;
  color: var(--color-accent);
  margin: 0 0 12px;
}

.reset-success-text {
  font-size: 15px;
  color: var(--color-text-secondary);
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
</style>
