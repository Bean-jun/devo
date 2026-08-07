import { ref, computed, onMounted, watch } from 'vue'
import { useSessionStore } from '@/stores/session'
import { useUiStore } from '@/stores/ui'
import { useSession } from '@/composables/useSession'
import { formatDateTime } from '@/utils/formatters'
import { STATUS_LABELS } from '@/utils/constants'
import type { TokenUsage } from '@/types/session'

export function useSessionPicker() {
  const sessionStore = useSessionStore()
  const uiStore = useUiStore()
  const { createAndSwitch, switchTo } = useSession()

  const searchQuery = ref('')
  const searchInput = ref<HTMLInputElement | null>(null)
  const selectedIndex = ref(0)

  function formatTokenUsage(usage?: TokenUsage): string {
    if (!usage || (usage.input === 0 && usage.output === 0)) return '0 token'
    const total = usage.input + usage.output
    if (total >= 1000) return `${(total / 1000).toFixed(1)}k token`
    return `${total} token`
  }

  function truncateText(text: string, maxLen: number): string {
    if (!text) return ''
    if (text.length <= maxLen) return text
    return text.slice(0, maxLen) + '...'
  }

  function formatLastMessageTime(time?: string): string {
    if (!time) return ''
    return formatDateTime(time)
  }

  const isOpen = computed(() => uiStore.activeModal === 'session-picker')
  const filteredSessions = computed(() => {
    if (!searchQuery.value) return sessionStore.sessions
    const q = searchQuery.value.toLowerCase()
    return sessionStore.sessions.filter(
      s => s.title.toLowerCase().includes(q) || s.id.toLowerCase().includes(q)
    )
  })

onMounted(async () => {
  await sessionStore.fetchSessions(sessionStore.workingDirectory)
})

watch(searchQuery, () => {
  selectedIndex.value = 0
})

watch(() => uiStore.activeModal, (val) => {
  if (val === 'session-picker') {
    selectedIndex.value = 0
  }
})

async function handleSelect(sessionId: string) {
  await switchTo(sessionId)
  uiStore.setActiveModal(null)
}

async function handleCreate() {
  const name = searchQuery.value.trim() || undefined
  await createAndSwitch(name)
  uiStore.setActiveModal(null)
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    e.stopPropagation()
    uiStore.setActiveModal(null)
    return
  }
  if (e.key === 'ArrowDown') {
    e.preventDefault()
    selectedIndex.value = Math.min(selectedIndex.value + 1, filteredSessions.value.length - 1)
    return
  }
  if (e.key === 'ArrowUp') {
    e.preventDefault()
    selectedIndex.value = Math.max(selectedIndex.value - 1, 0)
    return
  }
  if (e.key === 'Enter') {
    e.preventDefault()
    const session = filteredSessions.value[selectedIndex.value]
    if (session) {
      handleSelect(session.id)
    }
    return
  }
}

  return {
    sessionStore,
    uiStore,
    searchQuery,
    searchInput,
    selectedIndex,
    formatTokenUsage,
    truncateText,
    formatLastMessageTime,
    isOpen,
    filteredSessions,
    handleSelect,
    handleCreate,
    handleKeydown,
    STATUS_LABELS,
    formatDateTime,
  }
}