import { computed, ref, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { useSessionStore } from '@/stores/session'
import { useUiStore } from '@/stores/ui'
import { API_BASE } from '@/utils/constants'

export interface AppSidebarProps {
  collapsed: boolean
}

export function useAppSidebar(props: AppSidebarProps) {
  const router = useRouter()
  const sessionStore = useSessionStore()
  const uiStore = useUiStore()

const workspaces = computed(() => uiStore.workspaceList)

const currentWorkspace = computed(() =>
  workspaces.value.find(w => w.id === uiStore.activeWorkspace)
)

const sessions = computed(() => {
  return sessionStore.sessions
})

const currentSessionId = computed(() => sessionStore.currentSession?.id)

const deleteTarget = ref<{ id: string; name: string; path: string; exists: boolean } | null>(null)
const confirmInput = ref('')
const confirmError = ref('')

function openDeleteConfirm(ws: { id: string; name: string; path: string; exists: boolean }) {
  deleteTarget.value = { id: ws.id, name: ws.name, path: ws.path, exists: ws.exists }
  confirmInput.value = ''
  confirmError.value = ''
  nextTick(() => {
    const input = document.querySelector('.confirm-input') as HTMLInputElement
    input?.focus()
  })
}

function confirmDelete() {
  if (!deleteTarget.value) return
  if (deleteTarget.value.exists && confirmInput.value !== deleteTarget.value.path) {
    confirmError.value = '输入的路径不匹配'
    return
  }
  uiStore.removeWorkspace(deleteTarget.value.id)
  deleteTarget.value = null
  confirmInput.value = ''
  confirmError.value = ''
}

function cancelDelete(e?: Event) {
  e?.stopPropagation()
  deleteTarget.value = null
  confirmInput.value = ''
  confirmError.value = ''
}

const sessionDeleteTarget = ref<{ id: string; title: string } | null>(null)

function openSessionDeleteConfirm(sess: { id: string; title: string }) {
  sessionDeleteTarget.value = sess
}

async function confirmSessionDelete() {
  if (!sessionDeleteTarget.value) return
  await sessionStore.deleteSession(sessionDeleteTarget.value.id)
  sessionDeleteTarget.value = null
}

function cancelSessionDelete(e?: Event) {
  e?.stopPropagation()
  sessionDeleteTarget.value = null
}

function selectWorkspace(ws: { id: string; name: string; path: string }) {
  uiStore.setActiveWorkspace(ws.id)
  uiStore.setActiveRightTab('files')
  fetch(`${API_BASE}/current-workspace`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ working_directory: ws.path }),
  })
    .then(() => {
      sessionStore.workingDirectory = ws.path
    })
    .catch(() => {})
}

async function selectSession(sessionId: string) {
  const ok = await sessionStore.switchSessionById(sessionId)
  if (ok && sessionStore.currentSession?.workingDirectory) {
    fetch(`${API_BASE}/current-workspace`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ working_directory: sessionStore.currentSession.workingDirectory }),
    }).catch(() => {})
  }
  router.push('/chat')
}

function isActiveWorkspace(wsId: string): boolean {
  return uiStore.activeWorkspace === wsId
}

const showNewSessionDialog = ref(false)
const newSessionTitle = ref('')

function newSession() {
  newSessionTitle.value = ''
  showNewSessionDialog.value = true
  sessionStore.fetchAgents()
  nextTick(() => {
    const input = document.querySelector('.new-session-input') as HTMLInputElement
    input?.focus()
  })
}

async function confirmNewSession() {
  const dir = currentWorkspace.value?.path || sessionStore.workingDirectory
  await sessionStore.createSession({
    workingDirectory: dir,
    title: newSessionTitle.value.trim() || undefined,
    agent_id: sessionStore.selectedAgentId || undefined,
  })
  showNewSessionDialog.value = false
  router.push('/chat')
}

function cancelNewSession(e?: Event) {
  e?.stopPropagation()
  showNewSessionDialog.value = false
}

  return {
    uiStore,
    sessionStore,
    workspaces,
    currentWorkspace,
    sessions,
    currentSessionId,
    deleteTarget,
    confirmInput,
    confirmError,
    openDeleteConfirm,
    confirmDelete,
    cancelDelete,
    sessionDeleteTarget,
    openSessionDeleteConfirm,
    confirmSessionDelete,
    cancelSessionDelete,
    selectWorkspace,
    selectSession,
    isActiveWorkspace,
    showNewSessionDialog,
    newSessionTitle,
    newSession,
    confirmNewSession,
    cancelNewSession,
  }
}