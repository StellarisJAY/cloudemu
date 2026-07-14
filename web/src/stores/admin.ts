import { defineStore } from 'pinia'
import { ref } from 'vue'
import { adminApi } from '@/api/admin'
import type { Rom } from '@/types/api'

/** 管理员：平台内置 ROM 管理状态 */
export const useAdminStore = defineStore('admin', () => {
  const builtinRoms = ref<Rom[]>([])
  const loading = ref(false)

  async function fetchBuiltinRoms(): Promise<void> {
    loading.value = true
    try {
      const res = await adminApi.listBuiltin()
      builtinRoms.value = res.data.data?.roms ?? []
    } catch {
      builtinRoms.value = []
    } finally {
      loading.value = false
    }
  }

  async function uploadBuiltin(formData: FormData): Promise<string | null> {
    try {
      const res = await adminApi.uploadBuiltin(formData)
      if (res.data.code !== 0) {
        return res.data.message || '上传失败'
      }
      await fetchBuiltinRoms()
      return null
    } catch (e: unknown) {
      return extractError(e, '网络错误，上传失败')
    }
  }

  async function updateBuiltin(id: string, formData: FormData): Promise<string | null> {
    try {
      const res = await adminApi.updateBuiltin(id, formData)
      if (res.data.code !== 0) {
        return res.data.message || '更新失败'
      }
      await fetchBuiltinRoms()
      return null
    } catch (e: unknown) {
      return extractError(e, '网络错误，更新失败')
    }
  }

  async function deleteBuiltin(id: string): Promise<string | null> {
    try {
      const res = await adminApi.deleteBuiltin(id)
      if (res.data.code !== 0) {
        return res.data.message || '删除失败'
      }
      await fetchBuiltinRoms()
      return null
    } catch (e: unknown) {
      return extractError(e, '网络错误，删除失败')
    }
  }

  function extractError(e: unknown, fallback: string): string {
    if (e && typeof e === 'object' && 'response' in e) {
      const err = e as { response?: { data?: { message?: string } } }
      return err.response?.data?.message || fallback
    }
    return fallback
  }

  return { builtinRoms, loading, fetchBuiltinRoms, uploadBuiltin, updateBuiltin, deleteBuiltin }
})
