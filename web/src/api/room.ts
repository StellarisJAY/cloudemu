import client from './client'
import type {
  ApiResponse,
  Room,
  CreateRoomReq,
  ChangeRoleReq,
  InviteToRoomReq,
  StartRoomReq,
  SelectRomReq,
  SwitchRomReq,
  StartRoomResp,
  LivekitTokenResp,
  RoomMemberInfo,
  KickPlayerReq,
  LeaveRoomReq,
  PauseRoomReq,
  ResumeRoomReq,
  StopRoomReq,
  DeleteRoomReq,
  SaveState,
  SaveStateReq,
  LoadStateReq,
  LoadLatestStateReq,
  RenameSaveStateReq,
  DeleteSaveStateReq,
} from '@/types/api'

export const roomApi = {
  /** GET /api/rooms — 获取当前用户参与的活跃房间列表 */
  list() {
    return client.get<ApiResponse<Room[]>>('/rooms')
  },

  /** POST /api/rooms/create — 创建房间 */
  create(data: CreateRoomReq) {
    return client.post<ApiResponse<Room>>('/rooms/create', data)
  },

  /** POST /api/rooms/invite — 房主邀请好友加入房间，直接加入无需接受 */
  inviteToRoom(data: InviteToRoomReq) {
    return client.post<ApiResponse<null>>('/rooms/invite', data)
  },

  /** POST /api/rooms/change-role — 房主调整成员角色（提升为玩家/降为旁观） */
  changeRole(data: ChangeRoleReq) {
    return client.post<ApiResponse<null>>('/rooms/change-role', data)
  },

  /** POST /api/rooms/select-rom — 房主选择/切换房间的 ROM */
  selectRom(data: SelectRomReq) {
    return client.post<ApiResponse<null>>('/rooms/select-rom', data)
  },

  /** POST /api/rooms/switch-rom — 房主在游戏中热切换 ROM */
  switchRom(data: SwitchRomReq) {
    return client.post<ApiResponse<null>>('/rooms/switch-rom', data)
  },

  /** POST /api/rooms/start — 房主启动游戏 */
  start(data: StartRoomReq) {
    return client.post<ApiResponse<StartRoomResp>>('/rooms/start', data)
  },

  /** GET /api/rooms/:id/members — 获取房间成员列表 */
  getMembers(roomId: string) {
    return client.get<ApiResponse<RoomMemberInfo[]>>(`/rooms/${roomId}/members`)
  },

  /** GET /api/rooms/:id/livekit — 查询 LiveKit token（非房主轮询用） */
  getLivekitToken(roomId: string) {
    return client.get<ApiResponse<LivekitTokenResp>>(`/rooms/${roomId}/livekit`)
  },

  /** POST /api/rooms/kick — 房主踢出玩家 */
  kick(data: KickPlayerReq) {
    return client.post<ApiResponse<null>>('/rooms/kick', data)
  },

  /** POST /api/rooms/leave — 离开房间 */
  leave(data: LeaveRoomReq) {
    return client.post<ApiResponse<null>>('/rooms/leave', data)
  },

  /** POST /api/rooms/pause — 房主暂停游戏 */
  pause(data: PauseRoomReq) {
    return client.post<ApiResponse<null>>('/rooms/pause', data)
  },

  /** POST /api/rooms/resume — 房主继续游戏 */
  resume(data: ResumeRoomReq) {
    return client.post<ApiResponse<null>>('/rooms/resume', data)
  },

  /** POST /api/rooms/stop — 房主停止游戏 */
  stop(data: StopRoomReq) {
    return client.post<ApiResponse<null>>('/rooms/stop', data)
  },

  /** POST /api/rooms/delete — 房主删除房间 */
  deleteRoom(data: DeleteRoomReq) {
    return client.post<ApiResponse<null>>('/rooms/delete', data)
  },

  /** POST /api/rooms/save-state — 房主保存存档 */
  saveState(data: SaveStateReq) {
    return client.post<ApiResponse<SaveState>>('/rooms/save-state', data)
  },

  /** POST /api/rooms/load-state — 房主读取存档 */
  loadState(data: LoadStateReq) {
    return client.post<ApiResponse<null>>('/rooms/load-state', data)
  },

  /** GET /api/rooms/:id/save-states — 列出房间存档 */
  listSaveStates(roomId: string) {
    return client.get<ApiResponse<SaveState[]>>(`/rooms/${roomId}/save-states`)
  },

  /** POST /api/rooms/load-latest-state — 房主加载最新存档 */
  loadLatestState(data: LoadLatestStateReq) {
    return client.post<ApiResponse<null>>('/rooms/load-latest-state', data)
  },

  /** POST /api/rooms/rename-save-state — 房主重命名存档 */
  renameSaveState(data: RenameSaveStateReq) {
    return client.post<ApiResponse<null>>('/rooms/rename-save-state', data)
  },

  /** POST /api/rooms/delete-save-state — 房主删除存档 */
  deleteSaveState(data: DeleteSaveStateReq) {
    return client.post<ApiResponse<null>>('/rooms/delete-save-state', data)
  },
}
