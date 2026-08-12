import { computed, ref } from 'vue'
import { useChatStore } from '@/stores/chat'
import { useSessionStore } from '@/stores/session'
import { useUiStore } from '@/stores/ui'
import { useModelStore } from '@/stores/model'
import { API_BASE } from '@/utils/constants'
import { useCommand } from '@/composables/useCommand'
import MessageList from './MessageList.vue'

export function useChatPanel() {
  const chatStore = useChatStore()
  const sessionStore = useSessionStore()
  const uiStore = useUiStore()
  const modelStore = useModelStore()
  const { openPalette } = useCommand()

  const messageListRef = ref<InstanceType<typeof MessageList> | null>(null)

function handleScrollToMessage(messageId: string): void {
  messageListRef.value?.scrollToMessage(messageId)
}

const isDisabled = computed(() =>
  sessionStore.isProcessing ||
  sessionStore.isAwaitingApproval ||
  sessionStore.isArchived
)

const isProcessing = computed(() => sessionStore.isProcessing)

async function handleSend(text: string, images?: string[]) {
  if (!sessionStore.currentSession) return
  uiStore.clearErrorToasts()
  chatStore.appendUserMessage(text, images)

  try {
    const body: { content: string; images?: string[] } = { content: text }
    if (images && images.length > 0) {
      body.images = images
    }
    const res = await fetch(`${API_BASE}/sessions/${sessionStore.currentSession.id}/messages`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    })

    if (!res.ok) {
      const err = await res.json().catch(() => ({}))
      throw new Error(err.error || `发送失败 (${res.status})`)
    }
  } catch (err) {
    uiStore.showToast('error', `发送失败: ${(err as Error).message}`)
  }
}

async function handleStop() {
  if (!sessionStore.currentSession) return
  try {
    await fetch(`${API_BASE}/sessions/${sessionStore.currentSession.id}/cancel`, {
      method: 'POST',
    })
    sessionStore.updateSessionState(sessionStore.currentSession.id, 'idle')
    chatStore.isStreaming = false
  } catch {}
}

function handleClear() {
  chatStore.clearMessages()
}

function handleOpenCommand() {
  openPalette((cmd) => {
    if (cmd.id === 'model') {
      uiStore.setActiveModal('model-picker')
    } else {
      uiStore.setPendingCommand(cmd.name + ' ')
    }
  })
}

async function handleExecuteCommand(text: string) {
  const parts = text.trim().split(/\s+/)
  const cmd = parts[0].slice(1)
  const arg = parts.slice(1).join(' ') || undefined

  switch (cmd) {
    case 'new': {
      await sessionStore.createSession({
        title: arg || undefined,
        workingDirectory: sessionStore.workingDirectory,
      })
      uiStore.showToast('success', '会话已创建')
      break
    }
    case 'switch': {
      uiStore.setActiveModal('session-picker')
      break
    }
    case 'rename': {
      if (!sessionStore.currentSession) {
        uiStore.showToast('error', '没有当前会话')
        return
      }
      try {
        await sessionStore.renameSession(sessionStore.currentSession.id, arg || '')
        uiStore.showToast('success', '会话已重命名')
      } catch (e: any) {
        uiStore.showToast('error', e.message || '重命名失败')
      }
      break
    }
    case 'export': {
      if (!sessionStore.currentSession) {
        uiStore.showToast('error', '没有当前会话')
        return
      }
      try {
        const sid = sessionStore.currentSession.id
        await fetch(`${API_BASE}/sessions/${sid}/sync-archive`, { method: 'POST' })
        const url = `${API_BASE}/sessions/${sid}/archive`
        const filename = `${sid}.md`
        const a = document.createElement('a')
        a.href = url
        a.download = filename
        document.body.appendChild(a)
        a.click()
        document.body.removeChild(a)
      } catch {
        uiStore.showToast('error', '导出失败')
      }
      break
    }
    case 'rollback': {
      uiStore.setActiveModal('rollback-picker')
      break
    }
    case 'pause': {
      const sid = sessionStore.currentSession?.id
      if (!sid) return
      if (!sessionStore.canPause) {
        uiStore.showToast('error', `当前状态为 ${sessionStore.currentSession?.state}，无法暂停`)
        return
      }
      try {
        const res = await fetch(`${API_BASE}/sessions/${sid}/pause`, { method: 'POST' })
        if (!res.ok) {
          const data = await res.json().catch(() => ({}))
          throw new Error(data.message || `HTTP ${res.status}`)
        }
        sessionStore.updateSessionState(sid, 'paused')
        uiStore.showToast('info', '会话已暂停')
      } catch (e: any) {
        uiStore.showToast('error', e.message || '暂停失败')
      }
      break
    }
    case 'resume': {
      const sid = sessionStore.currentSession?.id
      if (!sid) return
      if (!sessionStore.canResume) {
        uiStore.showToast('error', `当前状态为 ${sessionStore.currentSession?.state}，无法恢复`)
        return
      }
      try {
        const res = await fetch(`${API_BASE}/sessions/${sid}/resume`, { method: 'POST' })
        if (!res.ok) {
          const data = await res.json().catch(() => ({}))
          throw new Error(data.message || `HTTP ${res.status}`)
        }
        sessionStore.updateSessionState(sid, 'tool_executing')
        uiStore.showToast('info', '会话已恢复')
      } catch (e: any) {
        uiStore.showToast('error', e.message || '恢复失败')
      }
      break
    }
    case 'cancel': {
      const sid = sessionStore.currentSession?.id
      if (!sid) return
      if (!sessionStore.canCancel) {
        uiStore.showToast('error', `当前状态为 ${sessionStore.currentSession?.state}，无法取消`)
        return
      }
      try {
        const res = await fetch(`${API_BASE}/sessions/${sid}/cancel`, { method: 'POST' })
        if (!res.ok) {
          const data = await res.json().catch(() => ({}))
          throw new Error(data.message || `HTTP ${res.status}`)
        }
        sessionStore.updateSessionState(sid, 'cancelled')
        uiStore.showToast('info', '操作已取消')
      } catch (e: any) {
        uiStore.showToast('error', e.message || '取消失败')
      }
      break
    }
    case 'compact': {
      const sid = sessionStore.currentSession?.id
      if (!sid) return
      try {
        const res = await fetch(`${API_BASE}/sessions/${sid}/compact`, { method: 'POST' })
        const data = await res.json().catch(() => ({}))
        if (!res.ok) {
          throw new Error(data.error || `HTTP ${res.status}`)
        }
        const compressedCount = data.compressed_count ?? 0
        const tokensRemoved = data.tokens_removed ?? 0
        if (compressedCount > 0) {
          uiStore.showToast('success', `上下文已压缩，压缩了 ${compressedCount} 条消息，减少约 ${tokensRemoved} tokens`)
        } else {
          uiStore.showToast('info', '当前上下文无需压缩')
        }
      } catch (e: any) {
        uiStore.showToast('error', e.message || '压缩失败')
      }
      break
    }
    case 'help': {
      uiStore.setActiveModal('help')
      break
    }
    case 'model': {
      if (arg) {
        if (modelStore.models.length === 0) {
          await modelStore.fetchModels()
        }
        const match = modelStore.models.find(
          m => m.id === arg || m.name === arg || m.model === arg
        )
        if (match) {
          try {
            await modelStore.activateModel(match.id)
            uiStore.showToast('success', `已切换到 ${match.name}`)
          } catch (e: any) {
            uiStore.showToast('error', e.message || '切换失败')
          }
        } else {
          uiStore.showToast('error', `未找到模型: ${arg}`)
        }
      } else {
        uiStore.setActiveModal('model-picker')
      }
      break
    }
    default: {
      uiStore.showToast('error', `未知命令: /${cmd}`)
      break
    }
  }
}

  return {
    sessionStore,
    chatStore,
    messageListRef,
    isDisabled,
    isProcessing,
    handleSend,
    handleStop,
    handleClear,
    handleOpenCommand,
    handleExecuteCommand,
    handleScrollToMessage,
  }
}