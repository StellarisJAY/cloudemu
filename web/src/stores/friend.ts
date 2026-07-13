import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { friendApi } from '@/api/friend'
import type { FriendWithUser, FriendPendingItem, UserSearchItem } from '@/types/api'

export const useFriendStore = defineStore('friend', () => {
  const friends = ref<FriendWithUser[]>([])
  const pendingList = ref<FriendPendingItem[]>([])
  const searchResults = ref<UserSearchItem[]>([])
  const loading = ref(false)
  const searchLoading = ref(false)

  const pendingCount = computed(() => pendingList.value.length)

  /** 获取已接受的好友列表 */
  async function fetchFriends(): Promise<void> {
    loading.value = true
    try {
      const res = await friendApi.list()
      friends.value = res.data.data?.friends ?? []
    } catch {
      friends.value = []
    } finally {
      loading.value = false
    }
  }

  /** 获取待处理的好友请求 */
  async function fetchPending(): Promise<void> {
    try {
      const res = await friendApi.listPending()
      pendingList.value = res.data.data?.pending ?? []
    } catch {
      pendingList.value = []
    }
  }

  /** 搜索用户 */
  async function searchUsers(q: string): Promise<void> {
    if (!q.trim()) {
      searchResults.value = []
      return
    }
    searchLoading.value = true
    try {
      const res = await friendApi.searchUsers(q)
      searchResults.value = res.data.data?.users ?? []
    } catch {
      searchResults.value = []
    } finally {
      searchLoading.value = false
    }
  }

  /** 添加好友 */
  async function addFriend(friendId: string): Promise<string | null> {
    try {
      const res = await friendApi.add({ friend_id: friendId })
      if (res.data.code !== 0) {
        return res.data.message || '发送失败'
      }
      return null
    } catch (e: unknown) {
      if (e && typeof e === 'object' && 'response' in e) {
        const err = e as { response?: { data?: { message?: string } } }
        return err.response?.data?.message || '网络错误，发送失败'
      }
      return '网络错误，发送失败'
    }
  }

  /** 接受好友申请 */
  async function acceptFriend(friendId: string): Promise<string | null> {
    try {
      const res = await friendApi.accept({ friend_id: friendId })
      if (res.data.code !== 0) {
        return res.data.message || '操作失败'
      }
      await fetchFriends()
      await fetchPending()
      return null
    } catch (e: unknown) {
      if (e && typeof e === 'object' && 'response' in e) {
        const err = e as { response?: { data?: { message?: string } } }
        return err.response?.data?.message || '网络错误，操作失败'
      }
      return '网络错误，操作失败'
    }
  }

  /** 拒绝好友申请 */
  async function rejectFriend(friendId: string): Promise<string | null> {
    try {
      const res = await friendApi.reject({ friend_id: friendId })
      if (res.data.code !== 0) {
        return res.data.message || '操作失败'
      }
      await fetchPending()
      return null
    } catch (e: unknown) {
      if (e && typeof e === 'object' && 'response' in e) {
        const err = e as { response?: { data?: { message?: string } } }
        return err.response?.data?.message || '网络错误，操作失败'
      }
      return '网络错误，操作失败'
    }
  }

  return {
    friends,
    pendingList,
    searchResults,
    loading,
    searchLoading,
    pendingCount,
    fetchFriends,
    fetchPending,
    searchUsers,
    addFriend,
    acceptFriend,
    rejectFriend,
  }
})
