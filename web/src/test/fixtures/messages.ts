import type { Message } from '@/types/message'

export const mockUserMessage: Message = {
  id: 'msg-001',
  sessionId: 'sess-001',
  role: 'user',
  content: 'Hello, can you help me write a function?',
  timestamp: '2026-01-01T12:00:00Z',
}

export const mockAssistantMessage: Message = {
  id: 'msg-002',
  sessionId: 'sess-001',
  role: 'assistant',
  content: 'Sure! Here is a function:\n\n```typescript\nfunction hello() {\n  return "Hello";\n}\n```',
  timestamp: '2026-01-01T12:00:05Z',
  tokenUsage: { input: 50, output: 30 },
}

export const mockSystemMessage: Message = {
  id: 'msg-003',
  sessionId: 'sess-001',
  role: 'system',
  content: 'Session created',
  timestamp: '2026-01-01T12:00:00Z',
}

export const mockToolCallMessage: Message = {
  id: 'msg-004',
  sessionId: 'sess-001',
  role: 'tool',
  content: 'File written successfully',
  timestamp: '2026-01-01T12:00:03Z',
  toolCall: {
    id: 'tool-001',
    name: 'write_file',
    parameters: { path: '/tmp/test/file.txt', content: 'Hello' },
    result: { success: true, bytesWritten: 5 },
    status: 'success',
    duration: 234,
  },
}