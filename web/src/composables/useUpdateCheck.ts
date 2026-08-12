import { ref } from 'vue'
import type { UpdateCheckResult } from '@/types/api'
import { API_BASE } from '@/utils/constants'

const updateInfo = ref<UpdateCheckResult | null>(null)
const isLoading = ref(false)
const error = ref<string | null>(null)

export function useUpdateCheck() {
  async function checkUpdate(): Promise<UpdateCheckResult | null> {
    if (isLoading.value) return updateInfo.value
    isLoading.value = true
    error.value = null
    try {
      const res = await fetch(`${API_BASE}/update/check`)
      if (!res.ok) {
        throw new Error(`HTTP ${res.status}`)
      }
      const data = (await res.json()) as UpdateCheckResult
      updateInfo.value = data
      return data
    } catch (err) {
      error.value = (err as Error).message || '检查更新失败'
      return null
    } finally {
      isLoading.value = false
    }
  }

  return { updateInfo, isLoading, error, checkUpdate }
}