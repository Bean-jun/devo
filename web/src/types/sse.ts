export type SSEEventType =
  | 'thinking'
  | 'reasoning_token'
  | 'reasoning_complete'
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
  | 'background_output'
  | 'error'

export interface SSEEvent {
  type: SSEEventType
  data: SSEEventData
}

export interface SSEEventData {
  content?: string
  token?: string
  reasoning?: string
  fullReasoning?: string
  full_reasoning?: string
  reasoning_tokens?: number
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
  /** background_output: OS PID of the background process emitting the output */
  pid?: number
  /** background_output: which stream the chunk came from - "stdout" or "stderr" */
  stream?: string
  /** background_output: the raw text chunk */
  data?: string
  [key: string]: unknown
}