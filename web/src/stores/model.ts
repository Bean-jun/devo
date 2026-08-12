import { defineStore } from 'pinia'
import { ref } from 'vue'
import { API_BASE } from '@/utils/constants'

export interface ModelItem {
  id: string
  name: string
  provider?: string
  api_key: string
  base_url: string
  model: string
  enable_reasoning?: boolean
  reasoning_effort?: string
  max_tokens?: number
}

export const useModelStore = defineStore('model', () => {
  const models = ref<ModelItem[]>([])
  const activeModelId = ref<string>('')
  const isLoading = ref(false)

  async function fetchModels(): Promise<void> {
    isLoading.value = true
    try {
      const res = await fetch(`${API_BASE}/global/config/models`)
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const data = await res.json()
      models.value = data.models ?? []
      activeModelId.value = data.active_model_id ?? ''
    } catch (e) {
      console.error('Failed to fetch models:', e)
    } finally {
      isLoading.value = false
    }
  }

  async function activateModel(id: string): Promise<void> {
    const res = await fetch(`${API_BASE}/global/config/models/${encodeURIComponent(id)}/activate`, {
      method: 'PUT',
    })
    if (!res.ok) {
      const data = await res.json().catch(() => ({}))
      throw new Error((data as any).error || `HTTP ${res.status}`)
    }
    activeModelId.value = id
  }

  async function addModel(model: Partial<ModelItem>): Promise<void> {
    const res = await fetch(`${API_BASE}/global/config/models`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(model),
    })
    if (!res.ok) {
      const data = await res.json().catch(() => ({}))
      throw new Error((data as any).error || `HTTP ${res.status}`)
    }
    const created = await res.json()
    models.value.push(created)
    if (models.value.length === 1) {
      activeModelId.value = created.id
    }
  }

  async function deleteModel(id: string): Promise<void> {
    const res = await fetch(`${API_BASE}/global/config/models/${encodeURIComponent(id)}`, {
      method: 'DELETE',
    })
    if (!res.ok) {
      const data = await res.json().catch(() => ({}))
      throw new Error((data as any).error || `HTTP ${res.status}`)
    }
    models.value = models.value.filter(m => m.id !== id)
    if (activeModelId.value === id) {
      activeModelId.value = models.value.length > 0 ? models.value[0].id : ''
    }
  }

  async function testModel(id: string): Promise<{ success: boolean; error?: string }> {
    const res = await fetch(`${API_BASE}/global/config/models/${encodeURIComponent(id)}/test`, {
      method: 'POST',
    })
    const data = await res.json()
    return data as { success: boolean; error?: string }
  }

  const activeModel = (): ModelItem | undefined => {
    return models.value.find(m => m.id === activeModelId.value)
  }

  return {
    models,
    activeModelId,
    isLoading,
    fetchModels,
    activateModel,
    addModel,
    deleteModel,
    testModel,
    activeModel,
  }
})