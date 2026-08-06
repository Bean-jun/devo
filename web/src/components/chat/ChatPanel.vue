<script setup lang="ts">
import { computed, ref } from 'vue'
import { useChatStore } from '@/stores/chat'
import { useSessionStore } from '@/stores/session'
import { useUiStore } from '@/stores/ui'
import { API_BASE } from '@/utils/constants'
import { useCommand } from '@/composables/useCommand'
import AppIcon from '@/components/common/AppIcon.vue'
import MessageList from './MessageList.vue'
import InputArea from './InputArea.vue'
import FloatingNavPanel from './FloatingNavPanel.vue'

const chatStore = useChatStore()
const sessionStore = useSessionStore()
const uiStore = useUiStore()
const { openPalette } = useCommand()

const props = withDefaults(defineProps<{
  hideInput?: boolean
}>(), {
  hideInput: false,
})

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
    uiStore.setPendingCommand(cmd.name + ' ')
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
    default: {
      uiStore.showToast('error', `未知命令: /${cmd}`)
      break
    }
  }
}
</script>

<template>
  <div v-if="sessionStore.currentSession" class="chat-panel" data-test="chat-panel">
    <div class="chat-body">
      <MessageList ref="messageListRef" />
      <FloatingNavPanel
        v-if="chatStore.messages.length > 0"
        :scroll-to-message="handleScrollToMessage"
      />
    </div>

    <InputArea
      v-if="!props.hideInput"
      :is-disabled="isDisabled"
      :is-processing="isProcessing"
      @send="handleSend"
      @stop="handleStop"
      @clear="handleClear"
      @open-command="handleOpenCommand"
      @execute-command="handleExecuteCommand"
    />
  </div>
  <div v-else class="chat-empty">
    <AppIcon name="chat-dots" :size="48" class="chat-empty-icon" />
    <div class="chat-empty-title">请选择或新建一个会话</div>
    <div class="chat-empty-desc">在左侧选择一个已有会话，或点击 + 新建会话开始对话</div>
  </div>
</template>

<style scoped>
.chat-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
}

.chat-body {
  flex: 1;
  display: flex;
  min-height: 0;
  position: relative;
  padding-bottom: var(--space-lg);
}

.chat-empty {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  color: var(--color-text-tertiary);
  user-select: none;
}

.chat-empty-icon {
  font-size: 48px;
  opacity: 0.3;
}

.chat-empty-title {
  font-size: 16px;
  font-weight: 500;
  color: var(--color-text-secondary);
}

.chat-empty-desc {
  font-size: 13px;
}
</style>