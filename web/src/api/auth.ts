import client from './client'
import type { ApiResponse } from '@/types/api'
import type {
  CaptchaResp,
  LoginReq,
  LoginResp,
  RegisterReq,
  TokenPair,
  User,
  UserProfile,
  UpdatePasswordReq,
  VerifyCaptchaReq,
  VerifyEmailReq,
  ResendCodeReq,
  ForgotPasswordReq,
} from '@/types/api'

export const authApi = {
  /** GET /api/auth/captcha — 获取图形验证码 */
  captcha() {
    return client.get<ApiResponse<CaptchaResp>>('/auth/captcha')
  },

  /** POST /api/auth/captcha/verify — 校验滑块验证码（阶段1） */
  verifyCaptcha(data: VerifyCaptchaReq) {
    return client.post<ApiResponse<null>>('/auth/captcha/verify', data)
  },

  /** POST /api/auth/register — 用户注册 */
  register(data: RegisterReq) {
    return client.post<ApiResponse<User>>('/auth/register', data)
  },

  /** POST /api/auth/verify-email — 邮箱验证激活 */
  verifyEmail(data: VerifyEmailReq) {
    return client.post<ApiResponse<null>>('/auth/verify-email', data)
  },

  /** POST /api/auth/login — 用户登录 */
  login(data: LoginReq) {
    return client.post<ApiResponse<LoginResp>>('/auth/login', data)
  },

  /** POST /api/auth/resend-code — 重发验证码 */
  resendCode(data: ResendCodeReq) {
    return client.post<ApiResponse<null>>('/auth/resend-code', data)
  },

  /** POST /api/auth/refresh — 刷新 Token */
  refresh(token: string) {
    return client.post<ApiResponse<TokenPair>>('/auth/refresh', { refresh_token: token })
  },

  /** GET /api/auth/me — 获取当前用户信息（需登录） */
  me() {
    return client.get<ApiResponse<User>>('/auth/me')
  },

  /** PUT /api/auth/profile — 更新个人信息（需登录，multipart/form-data） */
  updateProfile(data: FormData) {
    return client.put<ApiResponse<User>>('/auth/profile', data)
  },

  /** PUT /api/auth/password — 修改密码（需登录） */
  updatePassword(data: UpdatePasswordReq) {
    return client.put<ApiResponse<null>>('/auth/password', data)
  },

  /** POST /api/auth/forgot-password — 请求密码重置 */
  forgotPassword(data: ForgotPasswordReq) {
    return client.post<ApiResponse<null>>('/auth/forgot-password', data)
  },

  /** POST /api/auth/reset-password — 执行密码重置 */
  resetPassword(data: { token: string; new_password: string }) {
    return client.post<ApiResponse<null>>('/auth/reset-password', data)
  },

  /** GET /api/users/:id — 获取指定用户的公开信息（需登录） */
  getUser(id: string) {
    return client.get<ApiResponse<UserProfile>>(`/users/${id}`)
  },
}
