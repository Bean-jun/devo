import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
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

  describe('reasoning streaming', () => {
    it('should start reasoning', () => {
      const store = useChatStore()

      store.startReasoning()

      expect(store.isReasoningActive).toBe(true)
      expect(store.streamingReasoning).toBe('')
    })

    it('should append reasoning chunks', () => {
      const store = useChatStore()

      store.startReasoning()
      store.appendReasoningChunk('thinking ')
      store.appendReasoningChunk('about it')

      expect(store.streamingReasoning).toBe('thinking about it')
      expect(store.hasStreamingReasoning).toBe(true)
    })

    it('should finish reasoning but keep streaming content', () => {
      const store = useChatStore()

      store.startStreaming()
      store.startReasoning()
      store.appendReasoningChunk('done thinking')
      store.appendStreamChunk('answer')
      store.finishReasoning()

      expect(store.isReasoningActive).toBe(false)
      expect(store.isStreaming).toBe(true)
      expect(store.streamingReasoning).toBe('done thinking')
      expect(store.streamingContent).toBe('answer')
    })

    it('should persist reasoning into message when finishing', () => {
      const store = useChatStore()

      store.startStreaming()
      store.startReasoning()
      store.appendReasoningChunk('思考过程')
      store.finishReasoning()
      store.appendStreamChunk('最终答案')
      store.finishStreaming({ input: 5, output: 3 })

      expect(store.messages).toHaveLength(1)
      expect(store.messages[0].content).toBe('最终答案')
      expect(store.messages[0].reasoning).toBe('思考过程')
      expect(store.messages[0].tokenUsage).toEqual({ input: 5, output: 3 })
    })

    it('should accept reasoning argument to finishStreaming overriding streamed reasoning', () => {
      const store = useChatStore()

      store.startStreaming()
      store.startReasoning()
      store.appendReasoningChunk('流式思考')
      store.finishReasoning()
      store.finishStreaming({ input: 1, output: 1 }, '完整思考')

      expect(store.messages[0].reasoning).toBe('完整思考')
    })

    it('should not persist message if neither content nor reasoning exists', () => {
      const store = useChatStore()

      store.startStreaming()
      store.startReasoning()
      store.finishReasoning()
      store.finishStreaming()

      expect(store.messages).toHaveLength(0)
    })

    it('should still persist message if only reasoning exists without content', () => {
      const store = useChatStore()

      store.startStreaming()
      store.startReasoning()
      store.appendReasoningChunk('只有思考没有正文')
      store.finishReasoning()
      store.finishStreaming()

      expect(store.messages).toHaveLength(1)
      expect(store.messages[0].content).toBe('')
      expect(store.messages[0].reasoning).toBe('只有思考没有正文')
    })

    it('should clear reasoning state on clearMessages', () => {
      const store = useChatStore()

      store.startStreaming()
      store.startReasoning()
      store.appendReasoningChunk('abc')
      store.clearMessages()

      expect(store.isStreaming).toBe(false)
      expect(store.isReasoningActive).toBe(false)
      expect(store.streamingReasoning).toBe('')
      expect(store.streamingContent).toBe('')
    })

    it('should include reasoning in appendAssistantMessage', () => {
      const store = useChatStore()

      const msg = store.appendAssistantMessage('hi', { input: 1, output: 1 }, 'reasoning here')

      expect(msg.reasoning).toBe('reasoning here')
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

  describe('fetchMessages', () => {
    beforeEach(() => {
      vi.mocked(globalThis.fetch).mockReset()
    })

    afterEach(() => {
      vi.mocked(globalThis.fetch).mockReset()
    })

    const mockApiResponse = (data: unknown, ok = true, status = 200) => {
      vi.mocked(globalThis.fetch).mockResolvedValueOnce({
        ok,
        status,
        json: () => Promise.resolve(data),
        text: () => Promise.resolve(JSON.stringify(data)),
      } as Response)
    }

    const mockApiError = (error: Error) => {
      vi.mocked(globalThis.fetch).mockRejectedValueOnce(error)
    }

    it('should fetch messages and populate the list', async () => {
      const store = useChatStore()
      const messages = [
        {
          id: 'msg-1',
          role: 'user',
          content: 'Hello',
          created_at: '2026-01-01T12:00:00Z',
        },
        {
          id: 'msg-2',
          role: 'assistant',
          content: 'Hi there!',
          created_at: '2026-01-01T12:00:01Z',
        },
      ]

      mockApiResponse({ messages, total: 2 })

      await store.fetchMessages('sess-001')

      expect(store.messages).toHaveLength(2)
      expect(store.messages[0].role).toBe('user')
      expect(store.messages[0].content).toBe('Hello')
      expect(store.messages[1].role).toBe('assistant')
      expect(store.messages[1].content).toBe('Hi there!')
      expect(store.initialFetchDone).toBe(true)
      expect(store.fetchError).toBeNull()
    })

    it('should handle empty messages list', async () => {
      const store = useChatStore()

      mockApiResponse({ messages: [], total: 0 })

      await store.fetchMessages('sess-001')

      expect(store.messages).toHaveLength(0)
      expect(store.initialFetchDone).toBe(true)
      expect(store.fetchError).toBeNull()
    })

    it('should handle missing messages field', async () => {
      const store = useChatStore()

      mockApiResponse({ total: 0 })

      await store.fetchMessages('sess-001')

      expect(store.messages).toHaveLength(0)
      expect(store.initialFetchDone).toBe(true)
    })

    it('should reconstruct tool call with parameters from assistant message', async () => {
      const store = useChatStore()
      const messages = [
        {
          id: 'msg-1',
          role: 'user',
          content: 'Read the file config.yaml',
          created_at: '2026-01-01T12:00:00Z',
        },
        {
          id: 'msg-2',
          role: 'assistant',
          content: 'I will read the file.',
          created_at: '2026-01-01T12:00:01Z',
          tool_calls: [
            {
              id: 'tc-001',
              tool_name: 'read_file',
              params: { path: '/tmp/config.yaml' },
            },
          ],
        },
        {
          id: 'msg-3',
          role: 'tool',
          content: 'server:\n  port: 8080\n',
          created_at: '2026-01-01T12:00:02Z',
          tool_call_id: 'tc-001',
        },
      ]

      mockApiResponse({ messages, total: 3 })

      await store.fetchMessages('sess-001')

      expect(store.messages).toHaveLength(3)
      // 用户消息
      expect(store.messages[0].role).toBe('user')
      expect(store.messages[0].toolCall).toBeUndefined()
      // 助手消息
      expect(store.messages[1].role).toBe('assistant')
      expect(store.messages[1].toolCall).toBeUndefined()
      // 工具消息应包含 toolCall 信息
      const toolMsg = store.messages[2]
      expect(toolMsg.role).toBe('tool')
      expect(toolMsg.toolCall).toBeDefined()
      expect(toolMsg.toolCall!.id).toBe('tc-001')
      expect(toolMsg.toolCall!.name).toBe('read_file')
      expect(toolMsg.toolCall!.parameters).toEqual({ path: '/tmp/config.yaml' })
      expect(toolMsg.toolCall!.status).toBe('success')
      expect(toolMsg.toolCall!.result).toBeDefined()
      expect(toolMsg.toolCall!.result!.success).toBe(true)
      expect(toolMsg.toolCall!.result!.stdout).toBe('server:\n  port: 8080\n')
    })

    it('should detect failed tool call from content', async () => {
      const store = useChatStore()
      const messages = [
        {
          id: 'msg-1',
          role: 'user',
          content: 'Delete a file',
          created_at: '2026-01-01T12:00:00Z',
        },
        {
          id: 'msg-2',
          role: 'assistant',
          content: 'I will delete the file.',
          created_at: '2026-01-01T12:00:01Z',
          tool_calls: [
            {
              id: 'tc-001',
              tool_name: 'delete_file',
              params: { path: '/tmp/test.txt' },
            },
          ],
        },
        {
          id: 'msg-3',
          role: 'tool',
          content: '错误: Permission denied',
          created_at: '2026-01-01T12:00:02Z',
          tool_call_id: 'tc-001',
        },
      ]

      mockApiResponse({ messages, total: 3 })

      await store.fetchMessages('sess-001')

      const toolMsg = store.messages[2]
      expect(toolMsg.toolCall).toBeDefined()
      expect(toolMsg.toolCall!.status).toBe('failed')
      expect(toolMsg.toolCall!.result!.success).toBe(false)
      expect(toolMsg.toolCall!.result!.error).toBe('错误: Permission denied')
    })

    it('should detect failed tool call with "failed" keyword', async () => {
      const store = useChatStore()
      const messages = [
        {
          id: 'msg-1',
          role: 'user',
          content: 'Run a command',
          created_at: '2026-01-01T12:00:00Z',
        },
        {
          id: 'msg-2',
          role: 'assistant',
          content: '',
          created_at: '2026-01-01T12:00:01Z',
          tool_calls: [
            {
              id: 'tc-001',
              tool_name: 'execute_command',
              params: { command: 'invalid-cmd' },
            },
          ],
        },
        {
          id: 'msg-3',
          role: 'tool',
          content: 'Command execution failed: command not found',
          created_at: '2026-01-01T12:00:02Z',
          tool_call_id: 'tc-001',
        },
      ]

      mockApiResponse({ messages, total: 3 })

      await store.fetchMessages('sess-001')

      const toolMsg = store.messages[2]
      expect(toolMsg.toolCall!.status).toBe('failed')
    })

    it('should handle multiple tool calls in one response', async () => {
      const store = useChatStore()
      const messages = [
        {
          id: 'msg-1',
          role: 'user',
          content: 'Read two files',
          created_at: '2026-01-01T12:00:00Z',
        },
        {
          id: 'msg-2',
          role: 'assistant',
          content: '',
          created_at: '2026-01-01T12:00:01Z',
          tool_calls: [
            { id: 'tc-001', tool_name: 'read_file', params: { path: '/tmp/a.txt' } },
            { id: 'tc-002', tool_name: 'read_file', params: { path: '/tmp/b.txt' } },
          ],
        },
        {
          id: 'msg-3',
          role: 'tool',
          content: 'Content of a.txt',
          created_at: '2026-01-01T12:00:02Z',
          tool_call_id: 'tc-001',
        },
        {
          id: 'msg-4',
          role: 'tool',
          content: 'Content of b.txt',
          created_at: '2026-01-01T12:00:03Z',
          tool_call_id: 'tc-002',
        },
      ]

      mockApiResponse({ messages, total: 4 })

      await store.fetchMessages('sess-001')

      expect(store.messages).toHaveLength(4)
      expect(store.messages[2].toolCall!.id).toBe('tc-001')
      expect(store.messages[2].toolCall!.parameters).toEqual({ path: '/tmp/a.txt' })
      expect(store.messages[3].toolCall!.id).toBe('tc-002')
      expect(store.messages[3].toolCall!.parameters).toEqual({ path: '/tmp/b.txt' })
    })

    it('should handle tool call with missing tool_call_id in tool message', async () => {
      const store = useChatStore()
      const messages = [
        {
          id: 'msg-1',
          role: 'user',
          content: 'Write file',
          created_at: '2026-01-01T12:00:00Z',
        },
        {
          id: 'msg-2',
          role: 'assistant',
          content: '',
          created_at: '2026-01-01T12:00:01Z',
          tool_calls: [
            { id: 'tc-001', tool_name: 'write_file', params: { path: '/tmp/test.txt', content: 'Hello' } },
          ],
        },
        {
          id: 'msg-3',
          role: 'tool',
          content: 'File written',
          created_at: '2026-01-01T12:00:02Z',
        },
      ]

      mockApiResponse({ messages, total: 3 })

      await store.fetchMessages('sess-001')

      expect(store.messages).toHaveLength(3)
      // 没有 tool_call_id 的 tool 消息不应被关联到 tool call
      const toolMsg = store.messages[2]
      expect(toolMsg.toolCall).toBeUndefined()
    })

    it('should handle API error gracefully', async () => {
      const store = useChatStore()

      mockApiResponse({ error: 'Not found' }, false, 404)

      await store.fetchMessages('sess-001')

      expect(store.messages).toHaveLength(0)
      expect(store.initialFetchDone).toBe(true)
      expect(store.fetchError).toContain('404')
    })

    it('should handle network error gracefully', async () => {
      const store = useChatStore()

      mockApiError(new Error('Network failure'))

      await store.fetchMessages('sess-001')

      expect(store.messages).toHaveLength(0)
      expect(store.initialFetchDone).toBe(true)
      expect(store.fetchError).toBe('Network failure')
    })

    it('should handle null data gracefully', async () => {
      const store = useChatStore()

      mockApiResponse(null)

      await store.fetchMessages('sess-001')

      expect(store.messages).toHaveLength(0)
      expect(store.initialFetchDone).toBe(true)
    })

    it('should reset fetchError on subsequent successful fetch', async () => {
      const store = useChatStore()

      // First call fails
      mockApiResponse({ error: 'Not found' }, false, 500)
      await store.fetchMessages('sess-001')
      expect(store.fetchError).toContain('500')

      // Second call succeeds
      mockApiResponse({ messages: [
        { id: 'msg-1', role: 'user', content: 'Hello', created_at: '2026-01-01T12:00:00Z' },
      ], total: 1 })
      await store.fetchMessages('sess-001')
      expect(store.fetchError).toBeNull()
      expect(store.messages).toHaveLength(1)
    })

    it('should set correct sessionId on messages', async () => {
      const store = useChatStore()
      const messages = [
        { id: 'msg-1', role: 'user', content: 'Test', created_at: '2026-01-01T12:00:00Z' },
      ]

      mockApiResponse({ messages, total: 1 })

      await store.fetchMessages('sess-abc-123')

      expect(store.messages[0].sessionId).toBe('sess-abc-123')
    })

    it('should parse reasoning from assistant messages', async () => {
      const store = useChatStore()
      const messages = [
        { id: 'msg-1', role: 'user', content: 'Q', created_at: '2026-01-01T12:00:00Z' },
        {
          id: 'msg-2',
          role: 'assistant',
          content: 'A',
          reasoning: '思考过程',
          created_at: '2026-01-01T12:00:01Z',
        },
      ]

      mockApiResponse({ messages, total: 2 })

      await store.fetchMessages('sess-001')

      expect(store.messages[1].reasoning).toBe('思考过程')
    })

    it('should set reasoning to undefined when backend message has no reasoning field', async () => {
      const store = useChatStore()
      const messages = [
        {
          id: 'msg-1',
          role: 'assistant',
          content: 'no thinking',
          created_at: '2026-01-01T12:00:00Z',
        },
      ]

      mockApiResponse({ messages, total: 1 })

      await store.fetchMessages('sess-001')

      expect(store.messages[0].reasoning).toBeUndefined()
    })
  })
})