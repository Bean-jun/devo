<script setup lang="ts">
import { onMounted, watch, ref } from 'vue'
import { useSessionStore } from '@/stores/session'
import { useChatStore } from '@/stores/chat'
import { useUiStore } from '@/stores/ui'
import { useCommandStore } from '@/stores/command'
import { useApprovalStore } from '@/stores/approval'
import { useSkillsStore } from '@/stores/skills'
import { useMemoryStore } from '@/stores/memory'
import { useMcpStore } from '@/stores/mcp'
import { useSSE } from '@/composables/useSSE'
import { useKeyboard } from '@/composables/useKeyboard'
import { useCommand } from '@/composables/useCommand'
import { usePlatform } from '@/composables/usePlatform'
import { useThemeTransition } from '@/composables/useThemeTransition'
import { API_BASE } from '@/utils/constants'
import VscodeLayout from '@/layouts/VscodeLayout.vue'
import BrowserLayout from '@/layouts/BrowserLayout.vue'

const sessionStore = useSessionStore()
const chatStore = useChatStore()
const uiStore = useUiStore()
const commandStore = useCommandStore()
const approvalStore = useApprovalStore()
const skillsStore = useSkillsStore()
const memoryStore = useMemoryStore()
const mcpStore = useMcpStore()
const { detectMode, isVscodeMode } = usePlatform()
const { startTransition } = useThemeTransition()

const initialized = ref(false)

watch(
  () => uiStore.theme,
  (theme) => {
    document.documentElement.setAttribute('data-theme', theme)
  },
  { immediate: true }
)

const { connect, disconnect, onEvent, onStatusChange } = useSSE()
const { openPalette } = useCommand()

onStatusChange((connected) => {
  uiStore.setConnectionStatus(connected ? 'connected' : 'disconnected')
})

onMounted(async () => {
  detectMode()

  uiStore.registerThemeTransition((x, y, cb) => {
    startTransition(x, y, cb)
  })

  await sessionStore.fetchWorkspace()
  await uiStore.fetchWorkspaceList()

  const currentDir = sessionStore.workingDirectory
  if (currentDir) {
    uiStore.setActiveWorkspace(currentDir)
  }

  await sessionStore.fetchSessions(currentDir || uiStore.activeWorkspace || '')

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

  initialized.value = true
})

watch(() => uiStore.activeWorkspace, async (ws) => {
  if (!initialized.value || !ws) return
  await sessionStore.fetchSessions(ws)
  if (sessionStore.sessions.length > 0) {
    await sessionStore.switchSessionById(sessionStore.sessions[0].id)
  } else {
    await sessionStore.createSession({ workingDirectory: ws })
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

  onEvent('streaming_token', (data: any) => {
    chatStore.appendStreamChunk(data.content || data.token || '')
  })

  onEvent('streaming_complete', (_data: any) => {
    // streaming_complete arrives before message_complete
    // Actual message finalization is handled in message_complete
  })

  onEvent('token_usage', (data: any) => {
    if (sessionStore.currentSession) {
      sessionStore.updateTokenUsage(sessionStore.currentSession.id, {
        input: data.input_tokens ?? 0,
        output: data.output_tokens ?? 0,
        session_input_tokens: data.session_input_tokens,
        session_output_tokens: data.session_output_tokens,
        currentContextTokens: data.current_context_tokens,
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
    const diff = data.diff || ''
    const msgs = chatStore.messages
    for (let i = msgs.length - 1; i >= 0; i--) {
      const msg = msgs[i]
      if (msg.toolCall?.name === toolName) {
        chatStore.updateToolCallStatus(msg.toolCall!.id, success ? 'success' : 'failed', {
          stdout: summary,
          error: success ? undefined : summary,
          success,
          diff,
        })
        break
      }
    }
  })

  onEvent('tool_progress', (data: any) => {
    const toolName = data.tool_name || ''
    const msgs = chatStore.messages
    for (let i = msgs.length - 1; i >= 0; i--) {
      const msg = msgs[i]
      if (msg.toolCall && msg.toolCall.name === toolName && msg.toolCall.status === 'pending') {
        chatStore.updateToolCallStatus(msg.toolCall.id, 'executing')
        break
      }
    }
  })

  onEvent('approval_required', (data: any) => {
    const details = data.details || {}
    approvalStore.setApproval({
      id: data.approval_id || `appr-${Date.now()}`,
      sessionId: sessionStore.currentSession?.id || '',
      toolCallId: '',
      toolName: data.operation_type || 'unknown',
      riskLevel: data.risk_level || 'medium',
      parameters: {},
      command: details.command || '',
      diff: details.diff || '',
      filePath: details.path || '',
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

  onEvent('approval_resolved', (_data: any) => {
    approvalStore.clearApproval()
    if (uiStore.activeModal === 'approval') {
      uiStore.setActiveModal(null)
    }
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

  onEvent('file_state_warning', (data: any) => {
    chatStore.appendSystemMessage(`文件状态警告：${data.message || '文件可能已被外部修改'}`)
  })

  onEvent('skill_solidified', (data: any) => {
    skillsStore.updateSkillFromEvent({
      name: data.skill_name || '',
      source: 'project',
      priority: data.priority || 0,
      enabled: true,
      location: data.location || '',
      installedAt: new Date().toISOString(),
    })
  })

  onEvent('memory_updated', (data: any) => {
    memoryStore.updateMemoryFromEvent({
      id: data.memory_id || '',
      type: data.type || 'user',
      key: data.key || '',
      content: data.value || '',
      source: data.source || 'user',
      updatedAt: new Date().toISOString(),
      createdAt: data.created_at || new Date().toISOString(),
    })
  })

  onEvent('mcp_tool_discovered', (data: any) => {
    const tools = Array.isArray(data.tools) ? data.tools : (data.tools ? [data.tools] : [])
    mcpStore.updateToolsFromEvent(tools.map((t: any) => ({
      tool_name: t.tool_name || t.name || '',
      server_id: t.server_id || t.server_name || '',
      description: t.description || '',
      input_schema: t.input_schema || t.parameters || {},
    })))
  })

  onEvent('error', (data: any) => {
    uiStore.showToast('error', data.message || '发生错误')
  })
}

useKeyboard([
  {
    key: 'k',
    ctrl: true,
    handler: () => commandStore.isOpen ? commandStore.close() : openPalette((cmd) => {
      uiStore.setPendingCommand(cmd.name + ' ')
    }),
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
    shift: true,
    handler: async () => {
      if (!sessionStore.currentSession) return
      if (sessionStore.isPaused) {
        if (!sessionStore.canResume) {
          uiStore.showToast('error', `当前状态为 ${sessionStore.currentSession.state}，无法恢复`)
          return
        }
        try {
          const res = await fetch(`${API_BASE}/sessions/${sessionStore.currentSession.id}/resume`, {
            method: 'POST',
          })
          if (!res.ok) {
            const data = await res.json().catch(() => ({}))
            throw new Error(data.message || '恢复失败')
          }
          sessionStore.updateSessionState(sessionStore.currentSession.id, 'processing')
          uiStore.showToast('info', '会话已恢复')
        } catch (e: any) {
          uiStore.showToast('error', e.message || '恢复失败')
        }
      } else {
        if (!sessionStore.canPause) {
          uiStore.showToast('error', `当前状态为 ${sessionStore.currentSession.state}，无法暂停`)
          return
        }
        try {
          const res = await fetch(`${API_BASE}/sessions/${sessionStore.currentSession.id}/pause`, {
            method: 'POST',
          })
          if (!res.ok) {
            const data = await res.json().catch(() => ({}))
            throw new Error(data.message || '暂停失败')
          }
          sessionStore.updateSessionState(sessionStore.currentSession.id, 'paused')
          uiStore.showToast('info', '会话已暂停')
        } catch (e: any) {
          uiStore.showToast('error', e.message || '暂停失败')
        }
      }
    },
  },
  {
    key: 'r',
    ctrl: true,
    shift: true,
    handler: async () => {
      if (!sessionStore.currentSession) return
      if (!sessionStore.canResume) {
        uiStore.showToast('error', `当前状态为 ${sessionStore.currentSession.state}，无法恢复`)
        return
      }
      try {
        const res = await fetch(`${API_BASE}/sessions/${sessionStore.currentSession.id}/resume`, {
          method: 'POST',
        })
        if (!res.ok) {
          const data = await res.json().catch(() => ({}))
          throw new Error(data.message || '恢复失败')
        }
        sessionStore.updateSessionState(sessionStore.currentSession.id, 'processing')
        uiStore.showToast('info', '会话已恢复')
      } catch (e: any) {
        uiStore.showToast('error', e.message || '恢复失败')
      }
    },
  },
  {
    key: 'c',
    ctrl: true,
    shift: true,
    handler: async () => {
      if (!sessionStore.currentSession) return
      if (!sessionStore.canCancel) {
        uiStore.showToast('error', `当前状态为 ${sessionStore.currentSession.state}，无法取消`)
        return
      }
      try {
        const res = await fetch(`${API_BASE}/sessions/${sessionStore.currentSession.id}/cancel`, {
          method: 'POST',
        })
        if (!res.ok) {
          const data = await res.json().catch(() => ({}))
          throw new Error(data.message || '取消失败')
        }
        sessionStore.updateSessionState(sessionStore.currentSession.id, 'idle')
        uiStore.showToast('info', '操作已取消')
      } catch (e: any) {
        uiStore.showToast('error', e.message || '取消失败')
      }
    },
  },
])
</script>

<template>
  <VscodeLayout v-if="isVscodeMode" />
  <BrowserLayout v-else />
</template>