export type MessageRole = 'user' | 'assistant' | 'system' | 'tool'

export interface Message {
  id: string
  sessionId: string
  role: MessageRole
  content: string
  reasoning?: string
  timestamp: string
  tokenUsage?: TokenUsage
  toolCall?: ToolCall
  images?: string[]
}

export interface SendMessageRequest {
  content: string
  images?: string[]
}

export interface SendMessageResponse {
  id: string
  content: string
  role: MessageRole
  timestamp: string
}

export interface StreamChunk {
  content: string
  isComplete: boolean
}

import type { TokenUsage } from './session'
import type { ToolCall } from './tool'