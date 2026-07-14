import client from './client'
import type { ApiResponse } from '@/types/api'
import type { Rom, RomListResp } from '@/types/api'

export const adminApi = {
  /** GET /api/admin/roms — 列出全部平台内置 ROM（需管理员） */
  listBuiltin() {
    return client.get<ApiResponse<RomListResp>>('/admin/roms')
  },

  /** POST /api/admin/roms/upload — 上传平台内置 ROM（multipart/form-data） */
  uploadBuiltin(formData: FormData) {
    return client.post<ApiResponse<Rom>>('/admin/roms/upload', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
      timeout: 60000,
    })
  },

  /** PUT /api/admin/roms/:id — 更新内置 ROM 标题和封面（multipart/form-data） */
  updateBuiltin(id: string, formData: FormData) {
    return client.put<ApiResponse<Rom>>(`/admin/roms/${id}`, formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
      timeout: 60000,
    })
  },

  /** DELETE /api/admin/roms/:id — 删除内置 ROM */
  deleteBuiltin(id: string) {
    return client.delete<ApiResponse<{ deleted: string }>>(`/admin/roms/${id}`)
  },
}
