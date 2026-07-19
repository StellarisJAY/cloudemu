import client from './client'
import type { ApiResponse } from '@/types/api'
import type { Rom, RomListResp } from '@/types/api'

export const romApi = {
  /** GET /api/roms — 获取当前用户的 ROM 列表 */
  list() {
    return client.get<ApiResponse<RomListResp>>('/roms')
  },

  /** POST /api/roms/upload — 上传 ROM 文件（multipart/form-data） */
  upload(formData: FormData) {
    return client.post<ApiResponse<Rom>>('/roms/upload', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
      timeout: 60000, // 上传超时 60 秒
    })
  },

  /** PUT /api/roms/:id — 更新 ROM 标题和封面（multipart/form-data） */
  update(id: string, formData: FormData) {
    return client.put<ApiResponse<Rom>>(`/roms/${id}`, formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
      timeout: 60000,
    })
  },

  /** POST /api/roms/delete — 删除自有 ROM */
  deleteRom(romId: string) {
    return client.post<ApiResponse<null>>('/roms/delete', { rom_id: romId })
  },
}
