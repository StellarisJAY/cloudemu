import { defineStore } from 'pinia'
import { ref } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { roomApi } from '@/api/room'
import type {
  Room,
  CreateRoomReq,
  ChangeRoleReq,
  InviteToRoomReq,
  SelectRomReq,
  StartRoomResp,
  LivekitTokenResp,
  RoomMemberInfo,
  KickPlayerReq,
  PlayerRole,
  SaveState,
} from '@/types/api'

export interface PlayMember {
  userId: string
  username: string
  nickname: string | null
  avatar: string | null
  role: PlayerRole
  port: number | null
  isSelf: boolean
}

export const useRoomStore = defineStore('room', () => {
  const rooms = ref<Room[]>([])
  const loading = ref(false)

  async function fetchRooms(): Promise<void> {
    loading.value = true
    try {
      const res = await roomApi.list()
      rooms.value = res.data.data ?? []
    } catch {
      rooms.value = []
    } finally {
      loading.value = false
    }
  }

  async function createRoom(req: CreateRoomReq): Promise<string | null> {
    try {
      const res = await roomApi.create(req)
      if (res.data.code !== 0) {
        return res.data.message || '创建失败'
      }
      await fetchRooms()
      return null
    } catch (e: unknown) {
      if (e && typeof e === 'object' && 'response' in e) {
        const err = e as { response?: { data?: { message?: string } } }
        return err.response?.data?.message || '网络错误，创建失败'
      }
      return '网络错误，创建失败'
    }
  }

  async function selectRom(req: SelectRomReq): Promise<string | null> {
    try {
      const res = await roomApi.selectRom(req)
      if (res.data.code !== 0) {
        return res.data.message || '切换 ROM 失败'
      }
      return null
    } catch (e: unknown) {
      if (e && typeof e === 'object' && 'response' in e) {
        const err = e as { response?: { data?: { message?: string } } }
        return err.response?.data?.message || '网络错误，切换 ROM 失败'
      }
      return '网络错误，切换 ROM 失败'
    }
  }

  async function inviteToRoom(req: InviteToRoomReq): Promise<string | null> {
    try {
      const res = await roomApi.inviteToRoom(req)
      if (res.data.code !== 0) {
        return res.data.message || '邀请失败'
      }
      return null
    } catch (e: unknown) {
      if (e && typeof e === 'object' && 'response' in e) {
        const err = e as { response?: { data?: { message?: string } } }
        return err.response?.data?.message || '网络错误，邀请失败'
      }
      return '网络错误，邀请失败'
    }
  }

  async function startGame(roomId: string): Promise<StartRoomResp | null> {
    try {
      const res = await roomApi.start({ room_id: roomId })
      if (res.data.code !== 0) {
        throw new Error(res.data.message || '开始游戏失败')
      }
      return res.data.data ?? null
    } catch (e: unknown) {
      if (e && typeof e === 'object' && 'response' in e) {
        const err = e as { response?: { data?: { message?: string } } }
        throw new Error(err.response?.data?.message || '网络错误，开始游戏失败')
      }
      throw new Error('网络错误，开始游戏失败')
    }
  }

  async function getLivekitToken(roomId: string): Promise<LivekitTokenResp | null> {
    try {
      const res = await roomApi.getLivekitToken(roomId)
      if (res.data.code !== 0) {
        return null
      }
      return res.data.data ?? null
    } catch {
      return null
    }
  }

  async function changeRole(req: ChangeRoleReq): Promise<string | null> {
    try {
      const res = await roomApi.changeRole(req)
      if (res.data.code !== 0) {
        return res.data.message || '操作失败'
      }
      return null
    } catch (e: unknown) {
      if (e && typeof e === 'object' && 'response' in e) {
        const err = e as { response?: { data?: { message?: string } } }
        return err.response?.data?.message || '网络错误，操作失败'
      }
      return '网络错误，操作失败'
    }
  }

  async function fetchMembers(roomId: string): Promise<PlayMember[]> {
    const res = await roomApi.getMembers(roomId)
    if (res.data.code !== 0) {
      throw new Error(res.data.message || '获取成员失败')
    }
    const data = res.data.data ?? []
    const currentUserId = useAuthStore().user?.id
    return data.map((m: RoomMemberInfo) => ({
      userId: m.user_id,
      username: m.username,
      nickname: m.nickname,
      avatar: m.avatar,
      role: m.role,
      port: m.port,
      isSelf: m.user_id === currentUserId,
    }))
  }

  async function kickPlayer(req: KickPlayerReq): Promise<string | null> {
    try {
      const res = await roomApi.kick(req)
      if (res.data.code !== 0) {
        return res.data.message || '踢出失败'
      }
      return null
    } catch (e: unknown) {
      if (e && typeof e === 'object' && 'response' in e) {
        const err = e as { response?: { data?: { message?: string } } }
        return err.response?.data?.message || '网络错误，踢出失败'
      }
      return '网络错误，踢出失败'
    }
  }

  async function pauseGame(roomId: string): Promise<string | null> {
    try {
      const res = await roomApi.pause({ room_id: roomId })
      if (res.data.code !== 0) {
        return res.data.message || '暂停失败'
      }
      return null
    } catch (e: unknown) {
      if (e && typeof e === 'object' && 'response' in e) {
        const err = e as { response?: { data?: { message?: string } } }
        return err.response?.data?.message || '网络错误，暂停失败'
      }
      return '网络错误，暂停失败'
    }
  }

  async function resumeGame(roomId: string): Promise<string | null> {
    try {
      const res = await roomApi.resume({ room_id: roomId })
      if (res.data.code !== 0) {
        return res.data.message || '继续失败'
      }
      return null
    } catch (e: unknown) {
      if (e && typeof e === 'object' && 'response' in e) {
        const err = e as { response?: { data?: { message?: string } } }
        return err.response?.data?.message || '网络错误，继续失败'
      }
      return '网络错误，继续失败'
    }
  }

  async function stopGame(roomId: string): Promise<string | null> {
    try {
      const res = await roomApi.stop({ room_id: roomId })
      if (res.data.code !== 0) {
        return res.data.message || '停止失败'
      }
      return null
    } catch (e: unknown) {
      if (e && typeof e === 'object' && 'response' in e) {
        const err = e as { response?: { data?: { message?: string } } }
        return err.response?.data?.message || '网络错误，停止失败'
      }
      return '网络错误，停止失败'
    }
  }

  async function leaveRoom(roomId: string): Promise<string | null> {
    try {
      const res = await roomApi.leave({ room_id: roomId })
      if (res.data.code !== 0) {
        return res.data.message || '退出失败'
      }
      return null
    } catch (e: unknown) {
      if (e && typeof e === 'object' && 'response' in e) {
        const err = e as { response?: { data?: { message?: string } } }
        return err.response?.data?.message || '网络错误，退出失败'
      }
      return '网络错误，退出失败'
    }
  }

  async function deleteRoom(roomId: string): Promise<string | null> {
    try {
      const res = await roomApi.deleteRoom({ room_id: roomId })
      if (res.data.code !== 0) {
        return res.data.message || '删除失败'
      }
      return null
    } catch (e: unknown) {
      if (e && typeof e === 'object' && 'response' in e) {
        const err = e as { response?: { data?: { message?: string } } }
        return err.response?.data?.message || '网络错误，删除失败'
      }
      return '网络错误，删除失败'
    }
  }

  async function saveState(roomId: string): Promise<string | null> {
    try {
      const res = await roomApi.saveState({ room_id: roomId })
      if (res.data.code !== 0) {
        return res.data.message || '存档失败'
      }
      return null
    } catch (e: unknown) {
      if (e && typeof e === 'object' && 'response' in e) {
        const err = e as { response?: { data?: { message?: string } } }
        return err.response?.data?.message || '网络错误，存档失败'
      }
      return '网络错误，存档失败'
    }
  }

  async function loadState(roomId: string, saveStateId: string): Promise<string | null> {
    try {
      const res = await roomApi.loadState({ room_id: roomId, save_state_id: saveStateId })
      if (res.data.code !== 0) {
        return res.data.message || '读档失败'
      }
      return null
    } catch (e: unknown) {
      if (e && typeof e === 'object' && 'response' in e) {
        const err = e as { response?: { data?: { message?: string } } }
        return err.response?.data?.message || '网络错误，读档失败'
      }
      return '网络错误，读档失败'
    }
  }

  async function listSaveStates(roomId: string): Promise<SaveState[]> {
    try {
      const res = await roomApi.listSaveStates(roomId)
      if (res.data.code !== 0) {
        return []
      }
      return res.data.data ?? []
    } catch {
      return []
    }
  }

  async function loadLatestState(roomId: string): Promise<string | null> {
    try {
      const res = await roomApi.loadLatestState({ room_id: roomId })
      if (res.data.code !== 0) {
        return res.data.message || '读档失败'
      }
      return null
    } catch (e: unknown) {
      if (e && typeof e === 'object' && 'response' in e) {
        const err = e as { response?: { data?: { message?: string } } }
        return err.response?.data?.message || '网络错误，读档失败'
      }
      return '网络错误，读档失败'
    }
  }

  async function renameSaveState(
    roomId: string,
    saveStateId: string,
    name: string,
  ): Promise<string | null> {
    try {
      const res = await roomApi.renameSaveState({
        room_id: roomId,
        save_state_id: saveStateId,
        name,
      })
      if (res.data.code !== 0) {
        return res.data.message || '重命名失败'
      }
      return null
    } catch (e: unknown) {
      if (e && typeof e === 'object' && 'response' in e) {
        const err = e as { response?: { data?: { message?: string } } }
        return err.response?.data?.message || '网络错误，重命名失败'
      }
      return '网络错误，重命名失败'
    }
  }

  async function deleteSaveState(roomId: string, saveStateId: string): Promise<string | null> {
    try {
      const res = await roomApi.deleteSaveState({ room_id: roomId, save_state_id: saveStateId })
      if (res.data.code !== 0) {
        return res.data.message || '删除失败'
      }
      return null
    } catch (e: unknown) {
      if (e && typeof e === 'object' && 'response' in e) {
        const err = e as { response?: { data?: { message?: string } } }
        return err.response?.data?.message || '网络错误，删除失败'
      }
      return '网络错误，删除失败'
    }
  }

  return {
    rooms,
    loading,
    fetchRooms,
    createRoom,
    selectRom,
    changeRole,
    inviteToRoom,
    startGame,
    getLivekitToken,
    fetchMembers,
    kickPlayer,
    pauseGame,
    resumeGame,
    stopGame,
    leaveRoom,
    deleteRoom,
    saveState,
    loadState,
    listSaveStates,
    loadLatestState,
    renameSaveState,
    deleteSaveState,
  }
})
