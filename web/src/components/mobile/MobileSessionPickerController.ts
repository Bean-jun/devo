import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useSessionStore } from '@/stores/session'
import { useUiStore } from '@/stores/ui'
import { API_BASE } from '@/utils/constants'
import { formatDateTime } from '@/utils/formatters'

export function useMobileSessionPicker(emit: (e: string, ...args: any[]) => void) {
  const router = useRouter()
  const sessionStore = useSessionStore()
  const uiStore = useUiStore()

const sessions = computed(() => sessionStore.sessions)
const currentSessionId = computed(() => sessionStore.currentSession?.id)

function truncateText(text: string, maxLen: number): string {
  if (!text) return ''
  if (text.length <= maxLen) return text
  return text.slice(0, maxLen) + '...'
}

function formatLastMessageTime(time?: string): string {
  if (!time) return ''
  return formatDateTime(time)
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
  emit('close')
}

function newSession() {
  emit('newSession')
  emit('close')
}

function onBackdropClick() {
  emit('close')
}

function onSheetClick(e: Event) {
  e.stopPropagation()
}

let startY = 0

function onTouchStart(e: TouchEvent) {
  startY = e.touches[0].clientY
}

function onTouchMove(e: TouchEvent) {
  const delta = e.touches[0].clientY - startY
  if (delta > 80) {
    emit('close')
  }
}

  return {
    sessionStore,
    uiStore,
    sessions,
    currentSessionId,
    truncateText,
    formatLastMessageTime,
    selectSession,
    newSession,
    onBackdropClick,
    onSheetClick,
    onTouchStart,
    onTouchMove,
  }
}