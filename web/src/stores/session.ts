import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Session, CreateSessionRequest } from '@/types/session'
import { API_BASE } from '@/utils/constants'

export const useSessionStore = defineStore('session', () => {
  const currentSession = ref<Session | null>(null)
  const sessions = ref<Session[]>([])
  const isLoading = ref(false)
  const workingDirectory = ref('')

  const isProcessing = computed(() => currentSession.value?.state?.toLowerCase() === 'processing')
  const isAwaitingApproval = computed(() => currentSession.value?.state?.toLowerCase() === 'awaiting_approval')
  const isPaused = computed(() => currentSession.value?.state?.toLowerCase() === 'paused')
  const isArchived = computed(() => currentSession.value?.state?.toLowerCase() === 'archived')
  const sessionStatus = computed(() => (currentSession.value?.state ?? 'idle').toLowerCase())
  const isSessionActive = computed(() =>
    currentSession.value !== null &&
    currentSession.value.state?.toLowerCase() !== 'archived'
  )

  const canPause = computed(() => currentSession.value?.state?.toLowerCase() === 'processing')
  const canResume = computed(() => currentSession.value?.state?.toLowerCase() === 'paused')
  const canCancel = computed(() => {
    const s = currentSession.value?.state?.toLowerCase()
    return s === 'processing' || s === 'awaiting_approval'
  })

  function defaultTitle(): string {
    const now = new Date()
    const pad = (n: number) => String(n).padStart(2, '0')
    return `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())} ${pad(now.getHours())}:${pad(now.getMinutes())}:${pad(now.getSeconds())}`
  }

  async function createSession(request?: CreateSessionRequest): Promise<Session> {
    isLoading.value = true
    try {
      const body: Record<string, unknown> = {}
      body.title = request?.title || defaultTitle()
      if (request?.workingDirectory) body.working_directory = request.workingDirectory
      const res = await fetch(`${API_BASE}/sessions`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })
      if (!res.ok) throw new Error(`Failed to create session: ${res.status}`)
      const data = await res.json()
      const session: Session = {
        id: data.id,
        title: data.title || '未命名会话',
        state: data.state || 'idle',
        workingDirectory: data.working_directory || data.project_path || '',
        createdAt: data.created_at || new Date().toISOString(),
        lastActiveAt: data.last_active_at || new Date().toISOString(),
        messageCount: data.message_count || 0,
        tokenUsage: data.token_usage || { input: 0, output: 0 },
        trustLevel: data.trust_level || 'always_ask',
        approvalPolicy: data.approval_policy || 'always_ask',
        maxContextTokens: data.max_context_tokens,
      }
      currentSession.value = session
      sessions.value.unshift(session)
      return session
    } finally {
      isLoading.value = false
    }
  }

  async function fetchWorkspace(): Promise<string> {
    try {
      const res = await fetch(`${API_BASE}/workspace`)
      if (!res.ok) return ''
      const data = await res.json()
      workingDirectory.value = data.working_directory || ''
      return workingDirectory.value
    } catch {
      return ''
    }
  }

  async function fetchSessions(project?: string): Promise<void> {
    isLoading.value = true
    try {
      const params = new URLSearchParams()
      if (project) params.set('project', project)

      const url = `${API_BASE}/sessions${params.toString() ? '?' + params.toString() : ''}`
      const res = await fetch(url)
      if (!res.ok) throw new Error(`Failed to fetch sessions: ${res.status}`)
      const data = await res.json()
      sessions.value = (Array.isArray(data) ? data : (data.sessions || data.data || [])).map(mapSession)
    } finally {
      isLoading.value = false
    }
  }

  async function fetchSession(id: string): Promise<Session | null> {
    try {
      const res = await fetch(`${API_BASE}/sessions/${id}`)
      if (!res.ok) return null
      const data = await res.json()
      const session = mapSession(data)
      return session
    } catch {
      return null
    }
  }

  function switchSession(session: Session): void {
    currentSession.value = session
  }

  async function switchSessionById(id: string): Promise<boolean> {
    const existing = sessions.value.find(s => s.id === id)
    if (existing) {
      currentSession.value = existing
      return true
    }
    const fetched = await fetchSession(id)
    if (fetched) {
      currentSession.value = fetched
      sessions.value.unshift(fetched)
      return true
    }
    return false
  }

  async function renameSession(id: string, title: string): Promise<void> {
    try {
      await fetch(`${API_BASE}/sessions/${id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ title }),
      })
      if (currentSession.value?.id === id) {
        currentSession.value = { ...currentSession.value, title }
      }
      const idx = sessions.value.findIndex(s => s.id === id)
      if (idx !== -1) {
        sessions.value[idx] = { ...sessions.value[idx], title }
      }
    } catch {
      throw new Error('重命名失败')
    }
  }

  async function archiveSession(id: string): Promise<void> {
    try {
      await fetch(`${API_BASE}/sessions/${id}/archive`, { method: 'POST' })
      if (currentSession.value?.id === id) {
        currentSession.value = { ...currentSession.value, state: 'archived' }
      }
      const idx = sessions.value.findIndex(s => s.id === id)
      if (idx !== -1) {
        sessions.value[idx] = { ...sessions.value[idx], state: 'archived' }
      }
    } catch {
      throw new Error('归档失败')
    }
  }

  function updateSessionState(id: string, state: string): void {
    if (currentSession.value?.id === id) {
      currentSession.value = { ...currentSession.value, state: state as Session['state'] }
    }
    const idx = sessions.value.findIndex(s => s.id === id)
    if (idx !== -1) {
      sessions.value[idx] = { ...sessions.value[idx], state: state as Session['state'] }
    }
  }

  function updateTokenUsage(id: string, usage: { input: number; output: number }): void {
    if (currentSession.value?.id === id) {
      currentSession.value = { ...currentSession.value, tokenUsage: usage }
    }
  }

  function mapSession(data: any): Session {
    return {
      id: data.id,
      title: data.title || '未命名会话',
      state: data.state || 'idle',
      workingDirectory: data.working_directory || data.project_path || '',
      createdAt: data.created_at || new Date().toISOString(),
      lastActiveAt: data.last_active_at || new Date().toISOString(),
      messageCount: data.message_count || 0,
      tokenUsage: data.token_usage || { input: 0, output: 0 },
      trustLevel: data.trust_level || 'always_ask',
      approvalPolicy: data.approval_policy || 'always_ask',
      maxContextTokens: data.max_context_tokens,
    }
  }

  return {
    currentSession,
    sessions,
    isLoading,
    workingDirectory,
    isProcessing,
    isAwaitingApproval,
    isPaused,
    isArchived,
    canPause,
    canResume,
    canCancel,
    sessionStatus,
    isSessionActive,
    createSession,
    fetchWorkspace,
    fetchSessions,
    fetchSession,
    switchSession,
    switchSessionById,
    renameSession,
    archiveSession,
    updateSessionState,
    updateTokenUsage,
  }
})