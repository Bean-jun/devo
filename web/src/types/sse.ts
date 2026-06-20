export type SSEEventType =
  | 'thinking'
  | 'message_chunk'
  | 'tool_call_request'
  | 'tool_result'
  | 'message_complete'
  | 'approval_required'
  | 'approval_auto'
  | 'session_state_change'
  | 'token_usage'
  | 'context_compressed'
  | 'error'

export interface SSEEvent {
  type: SSEEventType
  data: SSEEventData
}

export interface SSEEventData {
  content?: string
  messageId?: string
  toolCall?: import('./tool').ToolCall
  approval?: import('./approval').ApprovalRequest
  sessionState?: import('./session').SessionState
  tokenUsage?: import('./session').TokenUsage
  rollbackTo?: string
  error?: string
  [key: string]: unknown
}