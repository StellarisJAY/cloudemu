import { defineStore } from 'pinia'
import { ref } from 'vue'
import { romApi } from '@/api/rom'
import type { Rom } from '@/types/api'

export const useRomStore = defineStore('rom', () => {
  const roms = ref<Rom[]>([])
  const loading = ref(false)

  async function fetchRoms(): Promise<void> {
    loading.value = true
    try {
      const res = await romApi.list()
      roms.value = res.data.data?.roms ?? []
    } catch {
      roms.value = []
    } finally {
      loading.value = false
    }
  }

  async function uploadRom(formData: FormData): Promise<string | null> {
    try {
      const res = await romApi.upload(formData)
      if (res.data.code !== 0) {
        return res.data.message || '上传失败'
      }
      await fetchRoms()
      return null
    } catch (e: unknown) {
      if (e && typeof e === 'object' && 'response' in e) {
        const err = e as { response?: { data?: { message?: string } } }
        return err.response?.data?.message || '网络错误，上传失败'
      }
      return '网络错误，上传失败'
    }
  }

  async function updateRom(id: string, formData: FormData): Promise<string | null> {
    try {
      const res = await romApi.update(id, formData)
      if (res.data.code !== 0) {
        return res.data.message || '更新失败'
      }
      await fetchRoms()
      return null
    } catch (e: unknown) {
      if (e && typeof e === 'object' && 'response' in e) {
        const err = e as { response?: { data?: { message?: string } } }
        return err.response?.data?.message || '网络错误，更新失败'
      }
      return '网络错误，更新失败'
    }
  }

  return {
    roms,
    loading,
    fetchRoms,
    uploadRom,
    updateRom,
  }
})
