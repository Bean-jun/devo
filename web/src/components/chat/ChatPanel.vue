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
  openPalette()
}
</script>

<template>
  <div class="chat-panel" data-test="chat-panel">
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
    />
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