import { ref, computed, watch } from 'vue'
import { useChatStore } from '@/stores/chat'
import { useUiStore } from '@/stores/ui'
import { useSessionStore } from '@/stores/session'
import { formatTime } from '@/utils/formatters'
import { API_BASE } from '@/utils/constants'

export function useRollbackPicker() {
  const chatStore = useChatStore()
  const uiStore = useUiStore()
  const sessionStore = useSessionStore()

  const selectedIndex = ref(0)
  const isLoading = ref(false)

  const isOpen = computed(() => uiStore.activeModal === 'rollback-picker')

  const userMessages = computed(() => {
    const msgs = chatStore.messages
      .map((msg, originalIndex) => ({ msg, originalIndex }))
      .filter(({ msg }) => msg.role === 'user')
      .reverse()
    return msgs
  })

watch(isOpen, async (val) => {
  if (val && sessionStore.currentSession) {
    isLoading.value = true
    try {
      await chatStore.fetchMessages(sessionStore.currentSession.id)
    } catch {
      uiStore.showToast('error', '加载消息失败')
    } finally {
      isLoading.value = false
    }
    if (userMessages.value.length > 0) {
      selectedIndex.value = userMessages.value[0].originalIndex
    }
  }
})

function selectMessage(entry: { msg: any; originalIndex: number }) {
  selectedIndex.value = entry.originalIndex
}

async function confirmRollback() {
  if (selectedIndex.value < 0 || !sessionStore.currentSession) return
  const targetMsg = userMessages.value.find(
    (e) => e.originalIndex === selectedIndex.value
  )
  if (!targetMsg) return

  try {
    const res = await fetch(
      `${API_BASE}/sessions/${sessionStore.currentSession.id}/rollback`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ target_message_id: targetMsg.msg.id }),
      }
    )
    if (!res.ok) throw new Error('回滚失败')
    chatStore.rollbackTo(selectedIndex.value)
    uiStore.setPendingCommand(targetMsg.msg.content)
    uiStore.showToast('info', '消息已回滚')
    uiStore.setActiveModal(null)
  } catch (err) {
    uiStore.showToast('error', `回滚失败: ${(err as Error).message}`)
  }
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    e.stopPropagation()
    uiStore.setActiveModal(null)
    return
  }
  if (e.key === 'ArrowDown') {
    e.preventDefault()
    const next = userMessages.value.findIndex(
      (entry) => entry.originalIndex === selectedIndex.value
    )
    if (next >= 0 && next < userMessages.value.length - 1) {
      selectedIndex.value = userMessages.value[next + 1].originalIndex
    }
    return
  }
  if (e.key === 'ArrowUp') {
    e.preventDefault()
    const curr = userMessages.value.findIndex(
      (entry) => entry.originalIndex === selectedIndex.value
    )
    if (curr > 0) {
      selectedIndex.value = userMessages.value[curr - 1].originalIndex
    }
    return
  }
  if (e.key === 'Enter') {
    e.preventDefault()
    confirmRollback()
    return
  }
}

  return {
    chatStore,
    uiStore,
    sessionStore,
    selectedIndex,
    isLoading,
    isOpen,
    userMessages,
    selectMessage,
    confirmRollback,
    handleKeydown,
    formatTime,
    API_BASE,
  }
}