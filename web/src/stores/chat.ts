import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Message } from '@/types/message'
import type { ToolCall } from '@/types/tool'
import { generateId } from '@/utils/formatters'
import { API_BASE } from '@/utils/constants'

export const useChatStore = defineStore('chat', () => {
  const messages = ref<Message[]>([])
  const isStreaming = ref(false)
  const streamingContent = ref('')
  const streamingMessageId = ref<string | null>(null)

  const lastMessage = computed(() => messages.value[messages.value.length - 1] ?? null)
  const messageCount = computed(() => messages.value.length)
  const canRollback = computed(() => messages.value.length > 0)
  const hasStreamingMessage = computed(() => isStreaming.value && streamingMessageId.value !== null)

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

  function appendAssistantMessage(content: string, tokenUsage?: { input: number; output: number }): Message {
    const msg: Message = {
      id: generateId(),
      sessionId: '',
      role: 'assistant',
      content,
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

  function appendStreamChunk(content: string): void {
    streamingContent.value += content
  }

  function finishStreaming(tokenUsage?: { input: number; output: number }): void {
    if (streamingContent.value) {
      const msg: Message = {
        id: streamingMessageId.value ?? generateId(),
        sessionId: '',
        role: 'assistant',
        content: streamingContent.value,
        timestamp: new Date().toISOString(),
        tokenUsage,
      }
      messages.value.push(msg)
    }
    isStreaming.value = false
    streamingContent.value = ''
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

  function clearMessages(): void {
    messages.value = []
    isStreaming.value = false
    streamingContent.value = ''
    streamingMessageId.value = null
  }

  function rollbackTo(messageIndex: number): void {
    if (messageIndex < 0 || messageIndex >= messages.value.length) return
    messages.value = messages.value.slice(0, messageIndex + 1)
  }

  async function fetchMessages(sessionId: string): Promise<void> {
    try {
      const res = await fetch(`${API_BASE}/sessions/${sessionId}/messages`)
      if (!res.ok) return
      const data = await res.json()
      const list = (data.messages || []) as any[]
      messages.value = list.map((m: any) => ({
        id: m.id || generateId(),
        sessionId,
        role: m.role || 'user',
        content: m.content || '',
        timestamp: m.created_at || new Date().toISOString(),
      }))
    } catch {
      // Ignore fetch errors
    }
  }

  return {
    messages,
    isStreaming,
    streamingContent,
    streamingMessageId,
    lastMessage,
    messageCount,
    canRollback,
    hasStreamingMessage,
    appendMessage,
    appendUserMessage,
    appendAssistantMessage,
    appendSystemMessage,
    appendToolCallMessage,
    startStreaming,
    appendStreamChunk,
    finishStreaming,
    updateToolCallStatus,
    clearMessages,
    rollbackTo,
    fetchMessages,
  }
})