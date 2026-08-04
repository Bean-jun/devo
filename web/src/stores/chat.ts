import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Message } from '@/types/message'
import type { ToolCall } from '@/types/tool'
import { generateId } from '@/utils/formatters'
import { API_BASE } from '@/utils/constants'

export const useChatStore = defineStore('chat', () => {
  const messages = ref<Message[]>([])
  const isStreaming = ref(false)
  const isReasoningActive = ref(false)
  const streamingContent = ref('')
  const streamingReasoning = ref('')
  const streamingMessageId = ref<string | null>(null)
  const initialFetchDone = ref(false)
  const fetchError = ref<string | null>(null)

  const lastMessage = computed(() => messages.value[messages.value.length - 1] ?? null)
  const messageCount = computed(() => messages.value.length)
  const canRollback = computed(() => messages.value.length > 0)
  const hasStreamingMessage = computed(() => isStreaming.value && streamingMessageId.value !== null)
  const hasStreamingReasoning = computed(() => streamingReasoning.value.length > 0)

  function appendMessage(message: Message): void {
    messages.value.push(message)
  }

  function appendUserMessage(content: string): Message {
    const msg: Message = {
      id: generateId(),
      sessionId: '',
      role: 'user',
      content,
      timestamp: new Date().toISOString(),
    }
    messages.value.push(msg)
    return msg
  }

  function appendAssistantMessage(content: string, tokenUsage?: { input: number; output: number }, reasoning?: string): Message {
    const msg: Message = {
      id: generateId(),
      sessionId: '',
      role: 'assistant',
      content,
      reasoning,
      timestamp: new Date().toISOString(),
      tokenUsage,
    }
    messages.value.push(msg)
    return msg
  }

  function appendSystemMessage(content: string): Message {
    const msg: Message = {
      id: generateId(),
      sessionId: '',
      role: 'system',
      content,
      timestamp: new Date().toISOString(),
    }
    messages.value.push(msg)
    return msg
  }

  function appendToolCallMessage(toolCall: ToolCall): Message {
    const msg: Message = {
      id: generateId(),
      sessionId: '',
      role: 'tool',
      content: toolCall.result?.stdout ?? toolCall.result?.error ?? '',
      timestamp: new Date().toISOString(),
      toolCall,
    }
    messages.value.push(msg)
    return msg
  }

  function startStreaming(): void {
    isStreaming.value = true
    streamingContent.value = ''
    streamingMessageId.value = generateId()
  }

  function startReasoning(): void {
    isReasoningActive.value = true
    streamingReasoning.value = ''
  }

  function appendStreamChunk(content: string): void {
    streamingContent.value += content
  }

  function appendReasoningChunk(content: string): void {
    streamingReasoning.value += content
  }

  function finishReasoning(): void {
    isReasoningActive.value = false
  }

  function commitStreamingSegment(): void {
    if (streamingContent.value.trim() || streamingReasoning.value.trim()) {
      const msg: Message = {
        id: streamingMessageId.value ?? generateId(),
        sessionId: '',
        role: 'assistant',
        content: streamingContent.value,
        reasoning: streamingReasoning.value || undefined,
        timestamp: new Date().toISOString(),
      }
      messages.value.push(msg)
    }
    streamingContent.value = ''
    streamingReasoning.value = ''
    streamingMessageId.value = generateId()
  }

  function finishStreaming(tokenUsage?: { input: number; output: number }, reasoning?: string): void {
    if (streamingContent.value.trim() || reasoning || streamingReasoning.value.trim()) {
      const msg: Message = {
        id: streamingMessageId.value ?? generateId(),
        sessionId: '',
        role: 'assistant',
        content: streamingContent.value,
        reasoning: reasoning || streamingReasoning.value || undefined,
        timestamp: new Date().toISOString(),
        tokenUsage,
      }
      messages.value.push(msg)
    }
    isStreaming.value = false
    isReasoningActive.value = false
    streamingContent.value = ''
    streamingReasoning.value = ''
    streamingMessageId.value = null
  }

  function updateToolCallStatus(toolCallId: string, status: ToolCall['status'], result?: ToolCall['result']): void {
    for (let i = messages.value.length - 1; i >= 0; i--) {
      const msg = messages.value[i]
      if (msg.toolCall?.id === toolCallId) {
        messages.value[i] = {
          ...msg,
          toolCall: {
            ...msg.toolCall!,
            status,
            result: result ?? msg.toolCall!.result,
          },
        }
        break
      }
    }
  }

  function updateToolProgress(toolCallId: string, stage: string, message?: string): void {
    for (let i = messages.value.length - 1; i >= 0; i--) {
      const msg = messages.value[i]
      if (msg.toolCall?.id === toolCallId) {
        messages.value[i] = {
          ...msg,
          toolCall: {
            ...msg.toolCall!,
            status: 'executing',
            stage,
          },
        }
        break
      }
    }
  }

  function appendToolStreamChunk(toolCallId: string, chunk: string): void {
    for (let i = messages.value.length - 1; i >= 0; i--) {
      const msg = messages.value[i]
      if (msg.toolCall?.id === toolCallId) {
        messages.value[i] = {
          ...msg,
          toolCall: {
            ...msg.toolCall!,
            status: 'executing',
            streamingOutput: (msg.toolCall?.streamingOutput ?? '') + chunk,
          },
        }
        break
      }
    }
  }

  function clearMessages(): void {
    messages.value = []
    isStreaming.value = false
    isReasoningActive.value = false
    streamingContent.value = ''
    streamingReasoning.value = ''
    streamingMessageId.value = null
    initialFetchDone.value = false
    fetchError.value = null
  }

  function rollbackTo(messageIndex: number): void {
    if (messageIndex < 0 || messageIndex >= messages.value.length) return
    messages.value = messages.value.slice(0, messageIndex)
  }

  async function fetchMessages(sessionId: string): Promise<void> {
    fetchError.value = null
    try {
      const res = await fetch(`${API_BASE}/sessions/${sessionId}/messages`)
      if (!res.ok) {
        fetchError.value = `Failed to load messages (${res.status})`
        initialFetchDone.value = true
        return
      }
      const data = await res.json()
      const list = (data.messages || []) as any[]

      const toolCallMap = new Map<string, ToolCall>()

      // 第一遍：从 assistant 消息中收集 tool_calls
      for (const m of list) {
        if (m.role === 'assistant' && Array.isArray(m.tool_calls)) {
          for (const tc of m.tool_calls) {
            toolCallMap.set(tc.id, {
              id: tc.id,
              name: tc.tool_name || 'unknown',
              parameters: tc.params || {},
              status: 'pending',
            })
          }
        }
      }

      // 第二遍：处理 tool 消息，更新 tool call 状态
      for (const m of list) {
        if (m.role === 'tool' && m.tool_call_id && toolCallMap.has(m.tool_call_id)) {
          const tc = toolCallMap.get(m.tool_call_id)!
          const content = m.content || ''
          const isError = content.startsWith('错误:') || content.includes('failed')
          // 替换对象引用，确保 Vue 能检测到变化
          toolCallMap.set(m.tool_call_id, {
            ...tc,
            status: isError ? 'failed' : 'success',
            result: {
              success: !isError,
              stdout: content,
              error: isError ? content : undefined,
            },
          })
        }
      }

      // 构建最终消息列表
      messages.value = list.map((m: any) => {
        const baseMsg = {
          id: m.id || generateId(),
          sessionId,
          role: m.role || 'user',
          content: m.content || '',
          reasoning: m.reasoning || undefined,
          timestamp: m.created_at || new Date().toISOString(),
        }

        if (m.role === 'tool' && m.tool_call_id && toolCallMap.has(m.tool_call_id)) {
          return {
            ...baseMsg,
            toolCall: toolCallMap.get(m.tool_call_id)!,
          }
        }

        return baseMsg
      })

      initialFetchDone.value = true
    } catch (err) {
      console.error('Failed to fetch messages:', err)
      fetchError.value = (err as Error).message || 'Failed to load messages'
      initialFetchDone.value = true
    }
  }

  return {
    messages,
    isStreaming,
    isReasoningActive,
    streamingContent,
    streamingReasoning,
    streamingMessageId,
    initialFetchDone,
    fetchError,
    lastMessage,
    messageCount,
    canRollback,
    hasStreamingMessage,
    hasStreamingReasoning,
    appendMessage,
    appendUserMessage,
    appendAssistantMessage,
    appendSystemMessage,
    appendToolCallMessage,
    startStreaming,
    startReasoning,
    appendStreamChunk,
    appendReasoningChunk,
    finishReasoning,
    commitStreamingSegment,
    finishStreaming,
    updateToolCallStatus,
    updateToolProgress,
    appendToolStreamChunk,
    clearMessages,
    rollbackTo,
    fetchMessages,
  }
})