<script setup lang="ts">
import { onMounted, watch } from 'vue'
import { useSessionStore } from '@/stores/session'
import { useChatStore } from '@/stores/chat'
import { useUiStore } from '@/stores/ui'
import { useCommandStore } from '@/stores/command'
import { useApprovalStore } from '@/stores/approval'
import { useSSE } from '@/composables/useSSE'
import { useKeyboard } from '@/composables/useKeyboard'
import { useCommand } from '@/composables/useCommand'
import { API_BASE } from '@/utils/constants'
import StatusBar from '@/components/layout/StatusBar.vue'
import ToastContainer from '@/components/layout/ToastContainer.vue'
import ChatPanel from '@/components/chat/ChatPanel.vue'
import CommandPalette from '@/components/command/CommandPalette.vue'
import ApprovalModal from '@/components/modal/ApprovalModal.vue'
import SessionPicker from '@/components/modal/SessionPicker.vue'
import RollbackPicker from '@/components/modal/RollbackPicker.vue'
import HelpPanel from '@/components/modal/HelpPanel.vue'
import InputModal from '@/components/modal/InputModal.vue'

const sessionStore = useSessionStore()
const chatStore = useChatStore()
const uiStore = useUiStore()
const commandStore = useCommandStore()
const approvalStore = useApprovalStore()
const { connect, disconnect, onEvent, onStatusChange } = useSSE()
const { openPalette } = useCommand()

onStatusChange((connected) => {
  uiStore.setConnectionStatus(connected ? 'connected' : 'disconnected')
})

onMounted(async () => {
  await sessionStore.fetchWorkspace()
  await sessionStore.fetchSessions(sessionStore.workingDirectory)

  const params = new URLSearchParams(window.location.search)
  const sessionIdFromUrl = params.get('session')
  if (sessionIdFromUrl && sessionStore.sessions.some(s => s.id === sessionIdFromUrl)) {
    await sessionStore.switchSessionById(sessionIdFromUrl)
  } else if (sessionStore.sessions.length > 0) {
    const latest = sessionStore.sessions[0]
    await sessionStore.switchSessionById(latest.id)
  } else {
    await sessionStore.createSession({ workingDirectory: sessionStore.workingDirectory })
  }
})

watch(
  () => sessionStore.currentSession?.id,
  (newId, oldId) => {
    if (newId) {
      const url = new URL(window.location.href)
      url.searchParams.set('session', newId)
      history.replaceState(null, '', url.toString())
    }
    if (newId && newId !== oldId) {
      disconnect()
      chatStore.clearMessages()
      chatStore.fetchMessages(newId)
      connectSSE(newId)
    }
  }
)

function connectSSE(sessionId: string) {
  uiStore.setConnectionStatus('connecting')
  connect(sessionId)

  onEvent('thinking', (_data: any) => {
    chatStore.startStreaming()
  })

  onEvent('token_usage', (data: any) => {
    if (sessionStore.currentSession) {
      sessionStore.updateTokenUsage(sessionStore.currentSession.id, {
        input: data.input_tokens ?? 0,
        output: data.output_tokens ?? 0,
      })
    }
  })

  onEvent('message_complete', (data: any) => {
    const hadStreamingContent = chatStore.streamingContent !== ''
    chatStore.finishStreaming({
      input: data.input_tokens ?? 0,
      output: data.output_tokens ?? 0,
    })
    const fullText = data.full_text || ''
    if (fullText && !hadStreamingContent) {
      chatStore.appendAssistantMessage(fullText, {
        input: data.input_tokens ?? 0,
        output: data.output_tokens ?? 0,
      })
    }
    if (sessionStore.currentSession) {
      sessionStore.updateSessionState(sessionStore.currentSession.id, 'idle')
    }
  })

  onEvent('tool_call_request', (data: any) => {
    chatStore.appendToolCallMessage({
      id: `tool-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
      name: data.tool_name || 'unknown',
      parameters: data.params || {},
      status: 'pending',
      riskLevel: data.risk_level || 'medium',
    })
  })

  onEvent('tool_result', (data: any) => {
    const toolName = data.tool_name || ''
    const success = data.success === true
    const summary = data.summary || ''
    const msgs = chatStore.messages
    for (let i = msgs.length - 1; i >= 0; i--) {
      const msg = msgs[i]
      if (msg.toolCall?.name === toolName) {
        chatStore.updateToolCallStatus(msg.toolCall!.id, success ? 'success' : 'failed', {
          stdout: summary,
          error: success ? undefined : summary,
          success,
        })
        break
      }
    }
  })

  onEvent('approval_required', (data: any) => {
    approvalStore.setApproval({
      id: data.approval_id || `appr-${Date.now()}`,
      sessionId: sessionStore.currentSession?.id || '',
      toolCallId: '',
      toolName: data.operation_type || 'unknown',
      riskLevel: data.risk_level || 'medium',
      parameters: {},
      command: data.command_preview || '',
      diff: data.diff || '',
      filePath: '',
      timeout: 30,
      createdAt: new Date().toISOString(),
    })
    uiStore.setActiveModal('approval')
  })

  onEvent('approval_auto', (data: any) => {
    const summary = data.summary || ''
    const policy = data.policy_level || ''
    chatStore.appendSystemMessage(`已根据信任策略（${policy}）自动批准：${summary}`)
  })

  onEvent('session_state_change', (data: any) => {
    if (sessionStore.currentSession) {
      const newState = data.new_state || 'idle'
      sessionStore.updateSessionState(sessionStore.currentSession.id, newState)
      if (data.reason === 'completed') {
        chatStore.appendSystemMessage('任务完成')
      } else if (data.reason === 'cancelled') {
        chatStore.appendSystemMessage('操作已取消')
      } else if (data.reason === 'tool_limit_reached') {
        chatStore.appendSystemMessage('已达到工具调用上限，输入新消息继续')
      } else if (data.reason === 'error') {
        chatStore.appendSystemMessage('发生错误，请重试')
      }
    }
  })

  onEvent('context_compressed', (data: any) => {
    const count = data.compressed_count ?? 0
    const tokens = data.tokens_removed ?? 0
    chatStore.appendSystemMessage(`上下文已压缩：${count} 条消息，释放约 ${tokens} tokens`)
  })

  onEvent('error', (data: any) => {
    uiStore.showToast('error', data.message || '发生错误')
  })
}

useKeyboard([
  {
    key: 'k',
    ctrl: true,
    handler: () => commandStore.isOpen ? commandStore.close() : openPalette(),
  },
  {
    key: 'Escape',
    handler: () => {
      if (commandStore.isOpen) commandStore.close()
      else if (uiStore.activeModal) uiStore.setActiveModal(null)
    },
  },
  {
    key: 'p',
    ctrl: true,
    handler: async () => {
      if (!sessionStore.currentSession) return
      const action = sessionStore.isPaused ? 'resume' : 'pause'
      try {
        await fetch(`${API_BASE}/sessions/${sessionStore.currentSession.id}/${action}`, {
          method: 'POST',
        })
        sessionStore.updateSessionState(
          sessionStore.currentSession.id,
          action === 'pause' ? 'paused' : 'idle'
        )
      } catch {}
    },
  },
  {
    key: 'l',
    ctrl: true,
    handler: () => chatStore.clearMessages(),
  },
])
</script>

<template>
  <div class="app-shell">
    <StatusBar />
    <ChatPanel />
    <Teleport to="body">
      <ToastContainer />
      <CommandPalette />
      <ApprovalModal />
      <SessionPicker />
      <RollbackPicker />
      <HelpPanel />
      <InputModal />
    </Teleport>
  </div>
</template>

<style scoped>
.app-shell {
  display: flex;
  flex-direction: column;
  height: 100vh;
  width: 100vw;
  overflow: hidden;
  background: var(--color-bg-primary);
}
</style>