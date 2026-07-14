import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { authApi } from '@/api/auth'
import { getAccessToken, setTokens, clearTokens } from '@/utils/token'
import type { User, LoginReq, UpdatePasswordReq } from '@/types/api'

/** 认证状态管理：用户信息、登录状态、登录/登出/更新资料 */
export const useAuthStore = defineStore('auth', () => {
  const router = useRouter()
  const user = ref<User | null>(null)

  /** 是否已登录（有 user 对象且有 token） */
  const isLoggedIn = computed(() => user.value !== null && getAccessToken() !== null)

  /** 是否为管理员（来自 /auth/me 的 is_admin） */
  const isAdmin = computed(() => user.value?.is_admin === true)

  /** 从服务端获取当前用户信息 */
  async function fetchUser(): Promise<void> {
    try {
      const res = await authApi.me()
      user.value = res.data.data ?? null
    } catch {
      user.value = null
    }
  }

  /** 登录：调用 API → 存储 token → 拉取用户信息 → 跳转 */
  async function login(data: LoginReq): Promise<string | null> {
    const res = await authApi.login(data)
    const resp = res.data.data
    if (res.data.code !== 0 || !resp) {
      return res.data.message || '登录失败'
    }
    setTokens(resp.access_token, resp.refresh_token)
    await fetchUser()
    return null
  }

  /** 更新个人信息（昵称/简介/头像） */
  async function updateProfile(formData: FormData): Promise<string | null> {
    try {
      const res = await authApi.updateProfile(formData)
      if (res.data.code !== 0) {
        return res.data.message || '保存失败'
      }
      user.value = res.data.data ?? null
      return null
    } catch {
      return '网络错误，保存失败'
    }
  }

  /** 修改密码 */
  async function updatePassword(data: UpdatePasswordReq): Promise<string | null> {
    try {
      const res = await authApi.updatePassword(data)
      if (res.data.code !== 0) {
        return res.data.message || '密码修改失败'
      }
      return null
    } catch {
      return '网络错误，修改失败'
    }
  }

  /** 登出：清除 token + user → 跳转登录页 */
  function logout(): void {
    clearTokens()
    user.value = null
    router.push('/login')
  }

  return { user, isLoggedIn, isAdmin, fetchUser, login, updateProfile, updatePassword, logout }
})
