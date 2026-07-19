/** 统一 API 响应结构（后端 response.Body） */
export interface ApiResponse<T = unknown> {
  code: number
  message: string
  data?: T
}

// ==================== 用户 ====================

/** 用户状态：0=待激活, 1=已激活, 2=已禁用 */
export type UserStatus = 0 | 1 | 2

export interface User {
  id: string
  username: string
  email: string
  nickname: string | null
  avatar: string | null
  bio: string | null
  status: UserStatus
  is_admin: boolean
  last_login_at: string | null
  created_at: string
  updated_at: string
}

// ==================== 认证 ====================

export interface CaptchaResp {
  captcha_key: string
  master_bg_base64: string
  tile_base64: string
  thumb_x: number
  thumb_y: number
  tile_width: number
  tile_height: number
}

export interface VerifyCaptchaReq {
  captcha_key: string
  slide_x: number
  slide_y: number
}

export interface RegisterReq {
  username: string
  email: string
  password: string
  captcha_key: string
}

export interface LoginReq {
  account: string
  password: string
  captcha_key: string
}

export interface LoginResp {
  access_token: string
  refresh_token: string
  expires_in: number
}

export interface TokenPair {
  access_token: string
  refresh_token: string
  expires_in: number
}

export interface VerifyEmailReq {
  email: string
  code: string
}

export interface ResendCodeReq {
  email: string
}

export interface UpdateProfileReq {
  nickname?: string | null
  bio?: string | null
}

export interface UpdatePasswordReq {
  old_password: string
  new_password: string
}

export interface ForgotPasswordReq {
  email: string
  captcha_key: string
}

export interface ResetPasswordReq {
  token: string
  new_password: string
}

// ==================== ROM ====================

/** ROM 状态：0=待审核, 1=已通过, 2=已拒绝 */
export type RomStatus = 0 | 1 | 2

/** 模拟器类型 */
export type EmulatorType = 'nes' | 'gb' | 'dos'

export interface Rom {
  id: string
  title: string
  emulator_type: EmulatorType
  file_size: number
  cover_path: string | null
  is_builtin: boolean
  created_at: string
}

export interface RomListResp {
  roms: Rom[]
  total: string
}

export interface UpdateRomReq {
  title: string
}

// ==================== 房间 ====================

/** 房间状态：0=等待中, 1=游戏中, 2=已关闭 */
export type RoomStatus = 0 | 1 | 2

/** 房间玩家角色：0=房主, 1=操作者, 2=旁观者 */
export type PlayerRole = 0 | 1 | 2

export interface Room {
  id: string
  host_id: string
  title: string
  emulator_type: EmulatorType
  rom_id: string | null
  max_ports: number
  status: RoomStatus
  started_at: string | null
  closed_at: string | null
  created_at: string
  updated_at: string
}

export interface CreateRoomReq {
  title: string
  emulator_type: EmulatorType
  rom_id?: string
  max_ports: number
  invitee_ids?: string[]
}

export interface ChangeRoleReq {
  room_id: string
  user_id: string
  /** 目标角色：1=玩家, 2=旁观 */
  role: 1 | 2
  /** 手柄端口号，role=1 时必传 */
  port?: number
}

export interface InviteToRoomReq {
  room_id: string
  invitee_ids: string[]
}

export interface StartRoomReq {
  room_id: string
}

export interface SelectRomReq {
  room_id: string
  rom_id: string
}

export interface SwitchRomReq {
  room_id: string
  rom_id: string
}

export interface StartRoomResp {
  livekit_token: string
  livekit_room: string
  livekit_url: string
}

export interface LivekitTokenResp {
  livekit_token?: string
  livekit_room?: string
  livekit_url?: string
  waiting: boolean
}

export interface LeaveRoomReq {
  room_id: string
}

export interface RoomMemberInfo {
  user_id: string
  username: string
  nickname: string | null
  avatar: string | null
  role: PlayerRole
  port: number | null
}

export interface KickPlayerReq {
  room_id: string
  user_id: string
}

export interface PauseRoomReq {
  room_id: string
}

export interface ResumeRoomReq {
  room_id: string
}

export interface StopRoomReq {
  room_id: string
}

export interface DeleteRoomReq {
  room_id: string
}

/** 游戏存档记录（与后端 model.SaveState 对应） */
export interface SaveState {
  id: string
  room_id: string
  name: string
  emulator_type: EmulatorType
  rom_id: string
  size: number
  created_by: string
  created_at: string
}

export interface SaveStateReq {
  room_id: string
}

export interface LoadStateReq {
  room_id: string
  save_state_id: string
}

export interface LoadLatestStateReq {
  room_id: string
}

export interface RenameSaveStateReq {
  room_id: string
  save_state_id: string
  name: string
}

export interface DeleteSaveStateReq {
  room_id: string
  save_state_id: string
}

// ==================== 好友 ====================

/** 好友状态：0=待接受, 1=已接受, 2=已拉黑, 3=已拒绝 */
export type FriendStatus = 0 | 1 | 2 | 3

/** 好友列表项（含好友用户信息） */
export interface FriendWithUser {
  id: string
  user_id: string
  friend_id: string
  status: FriendStatus
  accepted_at: string | null
  created_at: string
  username: string
  nickname: string | null
  avatar: string | null
}

export interface FriendListResp {
  friends: FriendWithUser[]
}

export interface FriendAddReq {
  friend_id: string
}

export interface FriendAcceptReq {
  friend_id: string
}

export interface FriendRejectReq {
  friend_id: string
}

/** 用户搜索结果项 */
export interface UserSearchItem {
  id: string
  username: string
  nickname: string | null
  avatar: string | null
}

export interface UserSearchResp {
  users: UserSearchItem[]
}

/** 待处理好友请求项（含发起者用户信息） */
export interface FriendPendingItem {
  id: string
  user_id: string
  friend_id: string
  created_at: string
  username: string
  nickname: string | null
  avatar: string | null
}

export interface FriendPendingListResp {
  pending: FriendPendingItem[]
}

/** 用户公开信息（查看其他用户资料时返回） */
export interface UserProfile {
  id: string
  username: string
  nickname: string | null
  avatar: string | null
  bio: string | null
}
