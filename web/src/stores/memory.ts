import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import type { Memory, MemoryCreateRequest } from '@/types/memory'
import { API_BASE } from '@/utils/constants'

export const useMemoryStore = defineStore('memory', () => {
  const memories = ref<Memory[]>([])
  const isLoading = ref(false)

  const globalMemories = computed(() =>
    memories.value.filter(m => m.scope === 'global')
  )

  function workspaceMemories(workspaceId: string): Memory[] {
    return memories.value.filter(m => m.scope === `workspace:${workspaceId}`)
  }

  async function fetchMemories(): Promise<void> {
    isLoading.value = true
    try {
      const res = await fetch(`${API_BASE}/memory`)
      if (!res.ok) throw new Error(`Failed to fetch memories: ${res.status}`)
      const data = await res.json()
      const list = Array.isArray(data) ? data : (data.memories || [])
      memories.value = list
    } catch {
      throw new Error('获取记忆列表失败')
    } finally {
      isLoading.value = false
    }
  }

  async function createMemory(request: MemoryCreateRequest): Promise<void> {
    const res = await fetch(`${API_BASE}/memory`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(request),
    })
    if (!res.ok) throw new Error('创建记忆失败')
    await fetchMemories()
  }

  async function updateMemory(key: string, value: string): Promise<void> {
    const res = await fetch(`${API_BASE}/memory/${encodeURIComponent(key)}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ value }),
    })
    if (!res.ok) throw new Error('更新记忆失败')
    const mem = memories.value.find(m => m.key === key)
    if (mem) {
      mem.value = value
      mem.updatedAt = new Date().toISOString()
    }
  }

  async function deleteMemory(key: string, scope: string): Promise<void> {
    const res = await fetch(`${API_BASE}/memory/${encodeURIComponent(key)}`, {
      method: 'DELETE',
    })
    if (!res.ok) throw new Error('删除记忆失败')
    memories.value = memories.value.filter(m => !(m.key === key && m.scope === scope))
  }

  function updateMemoryFromEvent(memory: Memory): void {
    const idx = memories.value.findIndex(m => m.id === memory.id)
    if (idx >= 0) {
      memories.value[idx] = { ...memories.value[idx], ...memory }
    } else {
      memories.value.push(memory)
    }
  }

  return {
    memories,
    isLoading,
    globalMemories,
    workspaceMemories,
    fetchMemories,
    createMemory,
    updateMemory,
    deleteMemory,
    updateMemoryFromEvent,
  }
})