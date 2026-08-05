import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Session, CreateSessionRequest, TrustLevel } from '@/types/session'
import { API_BASE } from '@/utils/constants'
import { useUiStore } from '@/stores/ui'

export const useSessionStore = defineStore('session', () => {
  const currentSession = ref<Session | null>(null)
  const sessions = ref<Session[]>([])
  const isLoading = ref(false)
  const workingDirectory = ref('')

  const isProcessing = computed(() => {
    const s = currentSession.value?.state?.toLowerCase()
    return s === 'processing' || s === 'thinking' || s === 'tool_executing'
  })
  const isThinking = computed(() => currentSession.value?.state?.toLowerCase() === 'thinking')
  const isToolExecuting = computed(() => currentSession.value?.state?.toLowerCase() === 'tool_executing')
  const isAwaitingApproval = computed(() => currentSession.value?.state?.toLowerCase() === 'awaiting_approval')
  const isPaused = computed(() => currentSession.value?.state?.toLowerCase() === 'paused')
  const isArchived = computed(() => currentSession.value?.state?.toLowerCase() === 'archived')
  const sessionStatus = computed(() => (currentSession.value?.state ?? 'idle').toLowerCase())
  const isSessionActive = computed(() =>
    currentSession.value !== null &&
    currentSession.value.state?.toLowerCase() !== 'archived'
  )

  const yoloEnabled = computed(() => currentSession.value?.trustLevel === 'elevated')

  const canPause = computed(() => currentSession.value?.state?.toLowerCase() === 'tool_executing')
  const canResume = computed(() => currentSession.value?.state?.toLowerCase() === 'paused')
  const canCancel = computed(() => {
    const s = currentSession.value?.state?.toLowerCase()
    return s === 'thinking' || s === 'tool_executing' || s === 'processing' || s === 'awaiting_approval' || s === 'paused'
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
        trustLevel: data.trust_level || 'normal',
        approvalPolicy: data.approval_policy || {},
        toolCallLimit: data.tool_call_limit,
        keepRecent: data.keep_recent,
        maxContextTokens: data.max_context_tokens,
        currentContextTokens: data.current_context_tokens,
      }
      currentSession.value = session
      sessions.value.unshift(session)
      if (request?.workingDirectory) {
        useUiStore().addWorkspace(request.workingDirectory)
      }
      return session
    } finally {
      isLoading.value = false
    }
  }

  async function fetchWorkspace(): Promise<string> {
    try {
      const res = await fetch(`${API_BASE}/current-workspace`)
      if (!res.ok) return ''
      const data = await res.json()
      workingDirectory.value = data.working_directory || ''
      if (workingDirectory.value) {
        useUiStore().addWorkspace(workingDirectory.value)
      }
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
    const fetched = await fetchSession(id)
    if (fetched) {
      currentSession.value = fetched
      const idx = sessions.value.findIndex(s => s.id === id)
      if (idx !== -1) {
        sessions.value[idx] = fetched
      } else {
        sessions.value.unshift(fetched)
      }
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

  async function deleteSession(id: string): Promise<void> {
    const res = await fetch(`${API_BASE}/sessions/${id}`, { method: 'DELETE' })
    if (!res.ok) throw new Error('删除失败')
    if (currentSession.value?.id === id) {
      currentSession.value = null
    }
    sessions.value = sessions.value.filter(s => s.id !== id)
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

  function updateTokenUsage(id: string, usage: {
    input: number
    output: number
    session_total_tokens?: number
    session_input_tokens?: number
    session_output_tokens?: number
    currentContextTokens?: number
  }): void {
    if (currentSession.value?.id === id) {
      const updates: any = {}

      if (usage.session_input_tokens != null || usage.session_output_tokens != null) {
        updates.tokenUsage = {
          input: usage.session_input_tokens ?? 0,
          output: usage.session_output_tokens ?? 0,
        }
      } else {
        updates.tokenUsage = {
          input: (currentSession.value.tokenUsage?.input ?? 0) + usage.input,
          output: (currentSession.value.tokenUsage?.output ?? 0) + usage.output,
        }
      }

      if (usage.currentContextTokens != null) {
        updates.currentContextTokens = usage.currentContextTokens
      }

      currentSession.value = { ...currentSession.value, ...updates }
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
      trustLevel: data.trust_level || 'normal',
      approvalPolicy: data.approval_policy || {},
      toolCallLimit: data.tool_call_limit,
      keepRecent: data.keep_recent,
      maxContextTokens: data.max_context_tokens,
      currentContextTokens: data.current_context_tokens,
      lastMessageContent: data.last_message_content,
      lastMessageTime: data.last_message_time,
    }
  }

  async function toggleYolo(): Promise<void> {
    const session = currentSession.value
    if (!session) return
    const newLevel: TrustLevel = session.trustLevel === 'elevated' ? 'normal' : 'elevated'
    const res = await fetch(`${API_BASE}/sessions/${session.id}/trust`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ trust_level: newLevel }),
    })
    if (!res.ok) throw new Error('切换 YOLO 模式失败')
    currentSession.value = { ...session, trustLevel: newLevel }
  }

  async function setApprovalPolicy(updates: Record<string, string>): Promise<void> {
    const session = currentSession.value
    if (!session) return
    const res = await fetch(`${API_BASE}/sessions/${session.id}/approval-policy`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(updates),
    })
    if (!res.ok) throw new Error('保存审批策略失败')
    currentSession.value = {
      ...session,
      approvalPolicy: { ...session.approvalPolicy, ...updates } as Record<string, any>,
    }
  }

  async function setProjectApprovalPolicy(updates: Record<string, string>): Promise<void> {
    const res = await fetch(`${API_BASE}/project/config`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        approval_policy: updates,
      }),
    })
    if (!res.ok) throw new Error('保存项目审批策略失败')
  }

  async function setGlobalApprovalPolicy(updates: Record<string, string>): Promise<void> {
    const res = await fetch(`${API_BASE}/global/config`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ approval_policy: updates }),
    })
    if (!res.ok) throw new Error('保存全局审批策略失败')
  }

  async function fetchProjectConfig(): Promise<{ skills: string[]; mcp: string[]; approval_policy: Record<string, string> }> {
    const res = await fetch(`${API_BASE}/project/config`)
    if (!res.ok) return { skills: [], mcp: [], approval_policy: {} }
    const data = await res.json()
    return {
      skills: data.skills || [],
      mcp: data.mcp || [],
      approval_policy: data.approval_policy || {},
    }
  }

  async function fetchGlobalApprovalPolicy(): Promise<Record<string, string>> {
    const res = await fetch(`${API_BASE}/global/config`)
    if (!res.ok) return {}
    const data = await res.json()
    return data.approval_policy || {}
  }

  return {
    currentSession,
    sessions,
    isLoading,
    workingDirectory,
    isProcessing,
    isThinking,
    isToolExecuting,
    isAwaitingApproval,
    isPaused,
    isArchived,
    canPause,
    canResume,
    canCancel,
    sessionStatus,
    isSessionActive,
    yoloEnabled,
    createSession,
    fetchWorkspace,
    fetchSessions,
    fetchSession,
    switchSession,
    switchSessionById,
    renameSession,
    archiveSession,
    deleteSession,
    updateSessionState,
    updateTokenUsage,
    toggleYolo,
    setApprovalPolicy,
    setProjectApprovalPolicy,
    setGlobalApprovalPolicy,
    fetchProjectConfig,
    fetchGlobalApprovalPolicy,
  }
})