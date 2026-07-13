import client from './client'
import type { ApiResponse } from '@/types/api'
import type {
  FriendAddReq,
  FriendAcceptReq,
  FriendRejectReq,
  FriendListResp,
  FriendPendingListResp,
  UserSearchResp,
} from '@/types/api'

export const friendApi = {
  /** GET /api/friends — 获取当前用户的好友列表（含用户信息） */
  list() {
    return client.get<ApiResponse<FriendListResp>>('/friends')
  },

  /** GET /api/friends/pending — 获取待处理的好友请求 */
  listPending() {
    return client.get<ApiResponse<FriendPendingListResp>>('/friends/pending')
  },

  /** GET /api/users/search?q= — 按用户名搜索用户 */
  searchUsers(q: string) {
    return client.get<ApiResponse<UserSearchResp>>('/users/search', { params: { q } })
  },

  /** POST /api/friends/add — 发送好友申请 */
  add(data: FriendAddReq) {
    return client.post<ApiResponse<null>>('/friends/add', data)
  },

  /** POST /api/friends/accept — 接受好友申请 */
  accept(data: FriendAcceptReq) {
    return client.post<ApiResponse<null>>('/friends/accept', data)
  },

  /** POST /api/friends/reject — 拒绝好友申请 */
  reject(data: FriendRejectReq) {
    return client.post<ApiResponse<null>>('/friends/reject', data)
  },
}
