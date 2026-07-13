import axios from 'axios'
import type { AxiosError, InternalAxiosRequestConfig } from 'axios'
import { getAccessToken, getRefreshToken, setTokens, clearTokens } from '@/utils/token'
import type { ApiResponse, TokenPair } from '@/types/api'

/** 创建 axios 实例，配置 baseURL 和超时 */
const client = axios.create({
  baseURL: import.meta.env.VITE_API_BASE || '/api',
  timeout: 15000,
  headers: { 'Content-Type': 'application/json' },
})

/** 标记是否正在刷新 token，避免并发请求重复刷新 */
let isRefreshing = false
/** 等待刷新完成的请求队列 */
let refreshSubscribers: Array<(token: string) => void> = []

/** 当前是否在 guest 路由上（登录/注册/找回密码等），此时不应触发 401 跳转 */
const guestPaths = ['/login', '/register', '/forgot-password', '/reset-password']
function isGuestPath() {
  return guestPaths.some((p) => window.location.pathname.startsWith(p))
}

/** 将等待刷新的请求加入队列 */
function subscribeTokenRefresh(cb: (token: string) => void) {
  refreshSubscribers.push(cb)
}

/** 刷新完成后，执行队列中所有等待的请求 */
function onTokenRefreshed(newToken: string) {
  refreshSubscribers.forEach((cb) => cb(newToken))
  refreshSubscribers = []
}

// ==================== 请求拦截器：自动注入 Authorization ====================

client.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  const token = getAccessToken()
  if (token && config.headers) {
    config.headers.Authorization = `Bearer ${token}`
  }
  // FormData 时删除 Content-Type，让浏览器自动设置 multipart/form-data + boundary
  if (config.data instanceof FormData && config.headers) {
    delete config.headers['Content-Type']
  }
  return config
})

// ==================== 响应拦截器：401 自动刷新 token ====================

client.interceptors.response.use(
  (res) => res,
  async (error: AxiosError<ApiResponse>) => {
    const originalRequest = error.config as InternalAxiosRequestConfig & { _retry?: boolean }

    // 非 401 或已重试过，直接抛出
    if (error.response?.status !== 401 || originalRequest._retry) {
      return Promise.reject(error)
    }

    const refreshToken = getRefreshToken()
    if (!refreshToken) {
      clearTokens()
      if (!isGuestPath()) {
        window.location.href = '/login'
      }
      return Promise.reject(error)
    }

    // 如果正在刷新中，将当前请求加入等待队列
    if (isRefreshing) {
      return new Promise((resolve) => {
        subscribeTokenRefresh((newToken: string) => {
          if (originalRequest.headers) {
            originalRequest.headers.Authorization = `Bearer ${newToken}`
          }
          resolve(client(originalRequest))
        })
      })
    }

    isRefreshing = true
    originalRequest._retry = true

    try {
      const { data } = await axios.post<ApiResponse<TokenPair>>(
        `${import.meta.env.VITE_API_BASE || '/api'}/auth/refresh`,
        { refresh_token: refreshToken },
      )

      if (data.code !== 0 || !data.data) {
        throw new Error('refresh failed')
      }

      const { access_token, refresh_token } = data.data
      setTokens(access_token, refresh_token)

      // 执行等待队列中的请求
      onTokenRefreshed(access_token)

      // 重试原请求
      if (originalRequest.headers) {
        originalRequest.headers.Authorization = `Bearer ${access_token}`
      }
      return client(originalRequest)
    } catch {
      clearTokens()
      if (!isGuestPath()) {
        window.location.href = '/login'
      }
      return Promise.reject(error)
    } finally {
      isRefreshing = false
    }
  },
)

export default client
