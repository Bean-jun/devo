import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import type { Memory, MemoryCreateRequest, MemoryType } from '@/types/memory'
import { API_BASE } from '@/utils/constants'
import { useSessionStore } from '@/stores/session'

export const useMemoryStore = defineStore('memory', () => {
  const memories = ref<Memory[]>([])
  const isLoading = ref(false)

  const userMemories = computed(() =>
    memories.value.filter(m => m.type === 'user')
  )

  const projectMemories = computed(() =>
    memories.value.filter(m => m.type === 'project')
  )

  function getSessionId(): string | undefined {
    const sessionStore = useSessionStore()
    return sessionStore.currentSession?.id
  }

  async function fetchMemories(type?: MemoryType): Promise<void> {
    const sessionId = getSessionId()
    if (!sessionId) {
      memories.value = []
      return
    }
    isLoading.value = true
    try {
      const url = type
        ? `${API_BASE}/sessions/${sessionId}/memory?type=${type}`
        : `${API_BASE}/sessions/${sessionId}/memory?type=user`
      const res = await fetch(url)
      if (!res.ok) throw new Error(`Failed to fetch memories: ${res.status}`)
      const data = await res.json()
      memories.value = data.memories || []
    } catch (e) {
      console.error('获取记忆列表失败', e)
    } finally {
      isLoading.value = false
    }
  }

  async function createMemory(request: MemoryCreateRequest): Promise<void> {
    const sessionId = getSessionId()
    if (!sessionId) throw new Error('无当前会话')
    const res = await fetch(`${API_BASE}/sessions/${sessionId}/memory`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ...request, action: 'upsert' }),
    })
    if (!res.ok) throw new Error('创建记忆失败')
    await fetchMemories()
  }

  async function updateMemory(id: string, content: string): Promise<void> {
    const sessionId = getSessionId()
    if (!sessionId) throw new Error('无当前会话')
    const mem = memories.value.find(m => m.id === id)
    if (!mem) throw new Error('记忆不存在')
    const res = await fetch(`${API_BASE}/sessions/${sessionId}/memory`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        type: mem.type,
        key: mem.key,
        content,
        action: 'upsert',
      }),
    })
    if (!res.ok) throw new Error('更新记忆失败')
    mem.content = content
    mem.updatedAt = new Date().toISOString()
  }

  async function deleteMemory(id: string): Promise<void> {
    const sessionId = getSessionId()
    if (!sessionId) throw new Error('无当前会话')
    const res = await fetch(`${API_BASE}/sessions/${sessionId}/memory/${encodeURIComponent(id)}`, {
      method: 'DELETE',
    })
    if (!res.ok) throw new Error('删除记忆失败')
    memories.value = memories.value.filter(m => m.id !== id)
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
    userMemories,
    projectMemories,
    fetchMemories,
    createMemory,
    updateMemory,
    deleteMemory,
    updateMemoryFromEvent,
  }
})