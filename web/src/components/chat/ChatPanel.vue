<script setup lang="ts">
import { computed } from 'vue'
import { useChatStore } from '@/stores/chat'
import { useSessionStore } from '@/stores/session'
import { useAutoScroll } from '@/composables/useAutoScroll'
import { API_BASE } from '@/utils/constants'
import { useUiStore } from '@/stores/ui'
import { useCommand } from '@/composables/useCommand'
import MessageList from './MessageList.vue'
import InputArea from './InputArea.vue'

const chatStore = useChatStore()
const sessionStore = useSessionStore()
const uiStore = useUiStore()
const { containerRef, showScrollToBottom, scrollToBottom, onScroll } = useAutoScroll()
const { openPalette } = useCommand()

const isDisabled = computed(() =>
  sessionStore.isProcessing ||
  sessionStore.isAwaitingApproval ||
  sessionStore.isArchived
)

const isProcessing = computed(() => sessionStore.isProcessing)

async function handleSend(text: string) {
  if (!sessionStore.currentSession) return
  chatStore.appendUserMessage(text)
  scrollToBottom(false)

  try {
    const res = await fetch(`${API_BASE}/sessions/${sessionStore.currentSession.id}/messages`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ content: text }),
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
        sessionStore.updateSessionState(sid, 'idle')
        uiStore.showToast('info', '操作已取消')
      } catch (e: any) {
        uiStore.showToast('error', e.message || '取消失败')
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
    <div
      ref="containerRef"
      class="message-area"
      @scroll="onScroll"
      @click.self="uiStore.requestFocusInput()"
    >
      <MessageList :scroll-to-bottom="scrollToBottom" />

      <button
        v-if="showScrollToBottom"
        class="scroll-to-bottom"
        @click="scrollToBottom()"
      >
        ↓ 回到底部
      </button>
    </div>

    <InputArea
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
    <div class="chat-empty-icon">💬</div>
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

.message-area {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  position: relative;
  padding: var(--space-lg) 0;
}

.scroll-to-bottom {
  position: sticky;
  bottom: var(--space-md);
  left: 50%;
  transform: translateX(-50%);
  padding: var(--space-xs) var(--space-md);
  background: var(--color-accent);
  color: var(--color-text-inverse);
  border-radius: var(--radius-full);
  font-size: var(--font-size-xs);
  box-shadow: var(--shadow-md);
  z-index: 10;
  animation: slideInUp var(--transition-fast) ease;
}

.scroll-to-bottom:hover {
  background: var(--color-accent-hover);
}
</style>