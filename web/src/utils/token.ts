/** localStorage 中 token 的存储键 */
const ACCESS_TOKEN_KEY = 'access_token'
const REFRESH_TOKEN_KEY = 'refresh_token'

/** 获取 Access Token */
export function getAccessToken(): string | null {
  return localStorage.getItem(ACCESS_TOKEN_KEY)
}

/** 保存 Access Token */
export function setAccessToken(token: string): void {
  localStorage.setItem(ACCESS_TOKEN_KEY, token)
}

/** 获取 Refresh Token */
export function getRefreshToken(): string | null {
  return localStorage.getItem(REFRESH_TOKEN_KEY)
}

/** 保存 Refresh Token */
export function setRefreshToken(token: string): void {
  localStorage.setItem(REFRESH_TOKEN_KEY, token)
}

/** 保存 Token 对（登录/刷新时使用） */
export function setTokens(accessToken: string, refreshToken: string): void {
  setAccessToken(accessToken)
  setRefreshToken(refreshToken)
}

/** 清除所有 Token（登出时使用） */
export function clearTokens(): void {
  localStorage.removeItem(ACCESS_TOKEN_KEY)
  localStorage.removeItem(REFRESH_TOKEN_KEY)
}
