export type SSEEventType =
  | 'thinking'
  | 'streaming_token'
  | 'streaming_complete'
  | 'tool_call_request'
  | 'tool_result'
  | 'tool_progress'
  | 'tool_chunk'
  | 'message_complete'
  | 'approval_required'
  | 'approval_auto'
  | 'approval_resolved'
  | 'session_state_change'
  | 'token_usage'
  | 'context_compressed'
  | 'file_state_warning'
  | 'skill_solidified'
  | 'memory_updated'
  | 'mcp_tool_discovered'
  | 'loop.completed_with_reason'
  | 'error'

export interface SSEEvent {
  type: SSEEventType
  data: SSEEventData
}

export interface SSEEventData {
  content?: string
  token?: string
  messageId?: string
  toolCall?: import('./tool').ToolCall
  tool_name?: string
  approval?: import('./approval').ApprovalRequest
  approval_id?: string
  sessionState?: import('./session').SessionState
  tokenUsage?: import('./session').TokenUsage
  rollbackTo?: string
  error?: string
  message?: string
  [key: string]: unknown
}