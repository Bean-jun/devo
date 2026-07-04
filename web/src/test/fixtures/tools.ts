import type { ToolCall } from '@/types/tool'

export const mockToolCallPending: ToolCall = {
  id: 'tool-001',
  name: 'write_file',
  parameters: {
    path: '/tmp/test/file.txt',
    content: 'Hello, world!',
  },
  status: 'pending',
  riskLevel: 'medium',
}

export const mockToolCallExecuting: ToolCall = {
  ...mockToolCallPending,
  status: 'executing',
  stage: 'running',
  streamingOutput: 'Writing file...\n',
}

export const mockToolCallSuccess: ToolCall = {
  ...mockToolCallPending,
  status: 'success',
  result: {
    success: true,
    bytesWritten: 13,
    path: '/tmp/test/file.txt',
  },
  duration: 234,
}

export const mockToolCallFailed: ToolCall = {
  ...mockToolCallPending,
  status: 'failed',
  result: {
    success: false,
    error: 'Permission denied',
  },
  duration: 50,
}

export const mockToolCallRejected: ToolCall = {
  ...mockToolCallPending,
  status: 'rejected',
}