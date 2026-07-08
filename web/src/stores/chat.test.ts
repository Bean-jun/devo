import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useChatStore } from '@/stores/chat'
import { mockUserMessage, mockAssistantMessage, mockSystemMessage, mockToolCallMessage } from '@/test/fixtures/messages'
import { mockToolCallPending, mockToolCallSuccess } from '@/test/fixtures/tools'

describe('ChatStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  describe('appendMessage', () => {
    it('should append a message', () => {
      const store = useChatStore()

      store.appendMessage({ ...mockUserMessage })

      expect(store.messages).toHaveLength(1)
      expect(store.messages[0].role).toBe('user')
    })
  })

  describe('appendUserMessage', () => {
    it('should append user message', () => {
      const store = useChatStore()

      const msg = store.appendUserMessage('Hello')

      expect(msg.role).toBe('user')
      expect(msg.content).toBe('Hello')
      expect(store.messages).toHaveLength(1)
    })
  })

  describe('appendAssistantMessage', () => {
    it('should append assistant message', () => {
      const store = useChatStore()

      const msg = store.appendAssistantMessage('Hi there!')

      expect(msg.role).toBe('assistant')
      expect(msg.content).toBe('Hi there!')
    })

    it('should include token usage when provided', () => {
      const store = useChatStore()

      const msg = store.appendAssistantMessage('Hi', { input: 10, output: 5 })

      expect(msg.tokenUsage).toEqual({ input: 10, output: 5 })
    })
  })

  describe('appendSystemMessage', () => {
    it('should append system message', () => {
      const store = useChatStore()

      const msg = store.appendSystemMessage('Session started')

      expect(msg.role).toBe('system')
      expect(msg.content).toBe('Session started')
    })
  })

  describe('appendToolCallMessage', () => {
    it('should append tool call message', () => {
      const store = useChatStore()

      const msg = store.appendToolCallMessage(mockToolCallPending)

      expect(msg.role).toBe('tool')
      expect(msg.toolCall?.name).toBe('write_file')
    })
  })

  describe('streaming', () => {
    it('should start streaming', () => {
      const store = useChatStore()

      store.startStreaming()

      expect(store.isStreaming).toBe(true)
      expect(store.streamingMessageId).toBeTruthy()
    })

    it('should append stream chunks', () => {
      const store = useChatStore()

      store.startStreaming()
      store.appendStreamChunk('Hello ')
      store.appendStreamChunk('World')

      expect(store.streamingContent).toBe('Hello World')
    })

    it('should finish streaming and add message', () => {
      const store = useChatStore()

      store.startStreaming()
      store.appendStreamChunk('Final content')
      store.finishStreaming({ input: 20, output: 10 })

      expect(store.isStreaming).toBe(false)
      expect(store.streamingContent).toBe('')
      expect(store.streamingMessageId).toBeNull()
      expect(store.messages).toHaveLength(1)
      expect(store.messages[0].content).toBe('Final content')
      expect(store.messages[0].tokenUsage).toEqual({ input: 20, output: 10 })
    })
  })

  describe('updateToolCallStatus', () => {
    it('should update tool call status', () => {
      const store = useChatStore()

      store.appendToolCallMessage(mockToolCallPending)
      store.updateToolCallStatus('tool-001', 'success', { success: true })

      const msg = store.messages[0]
      expect(msg.toolCall?.status).toBe('success')
      expect(msg.toolCall?.result?.success).toBe(true)
    })
  })

  describe('updateToolProgress', () => {
    it('should update tool stage and status', () => {
      const store = useChatStore()

      store.appendToolCallMessage(mockToolCallPending)
      store.updateToolProgress('tool-001', 'running')

      const msg = store.messages[0]
      expect(msg.toolCall?.status).toBe('executing')
      expect(msg.toolCall?.stage).toBe('running')
    })

    it('should update stage for existing executing tool', () => {
      const store = useChatStore()

      store.appendToolCallMessage(mockToolCallPending)
      store.updateToolProgress('tool-001', 'starting')
      store.updateToolProgress('tool-001', 'running')

      const msg = store.messages[0]
      expect(msg.toolCall?.stage).toBe('running')
    })
  })

  describe('appendToolStreamChunk', () => {
    it('should append streaming chunks', () => {
      const store = useChatStore()

      store.appendToolCallMessage(mockToolCallPending)
      store.appendToolStreamChunk('tool-001', 'Line 1\n')
      store.appendToolStreamChunk('tool-001', 'Line 2\n')

      const msg = store.messages[0]
      expect(msg.toolCall?.status).toBe('executing')
      expect(msg.toolCall?.streamingOutput).toBe('Line 1\nLine 2\n')
    })

    it('should handle empty initial streamingOutput', () => {
      const store = useChatStore()

      store.appendToolCallMessage(mockToolCallPending)
      store.appendToolStreamChunk('tool-001', 'Hello')

      const msg = store.messages[0]
      expect(msg.toolCall?.streamingOutput).toBe('Hello')
    })
  })

  describe('clearMessages', () => {
    it('should clear all messages', () => {
      const store = useChatStore()

      store.appendUserMessage('Hello')
      store.appendAssistantMessage('Hi')
      store.clearMessages()

      expect(store.messages).toHaveLength(0)
    })
  })

  describe('rollbackTo', () => {
    it('should rollback to specific message index', () => {
      const store = useChatStore()

      store.appendUserMessage('A')
      store.appendAssistantMessage('B')
      store.appendUserMessage('C')
      store.appendAssistantMessage('D')

      store.rollbackTo(1)

      expect(store.messages).toHaveLength(1)
      expect(store.messages[0].content).toBe('A')
    })

    it('should not rollback beyond message count', () => {
      const store = useChatStore()

      store.appendUserMessage('A')
      store.rollbackTo(10)

      expect(store.messages).toHaveLength(1)
    })
  })

  describe('computed', () => {
    it('should return last message', () => {
      const store = useChatStore()

      store.appendUserMessage('First')
      store.appendAssistantMessage('Last')

      expect(store.lastMessage?.content).toBe('Last')
    })

    it('should return null for lastMessage when empty', () => {
      const store = useChatStore()
      expect(store.lastMessage).toBeNull()
    })

    it('should return message count', () => {
      const store = useChatStore()

      store.appendUserMessage('A')
      store.appendAssistantMessage('B')

      expect(store.messageCount).toBe(2)
    })

    it('should detect canRollback', () => {
      const store = useChatStore()

      expect(store.canRollback).toBe(false)

      store.appendUserMessage('A')
      expect(store.canRollback).toBe(true)
    })
  })
})