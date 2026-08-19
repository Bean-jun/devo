export type SessionState =
  | 'idle'
  | 'thinking'
  | 'tool_executing'
  | 'processing' // deprecated: replaced by thinking + tool_executing
  | 'awaiting_approval'
  | 'paused'
  | 'completed'
  | 'archived'

export type TrustLevel = 'low' | 'normal' | 'elevated'

export type ApprovalPolicyLevel = 'always_ask' | 'session_trust' | 'full_trust' | 'auto_approve'

export type ApprovalPolicy = Record<string, ApprovalPolicyLevel>

export interface TokenUsage {
  input: number
  output: number
}

export interface Session {
  id: string
  title: string
  state: SessionState
  workingDirectory: string
  agentId: string
  createdAt: string
  lastActiveAt: string
  messageCount: number
  tokenUsage: TokenUsage
  trustLevel: TrustLevel
  approvalPolicy: ApprovalPolicy
  currentContextTokens?: number
  lastMessageContent?: string
  lastMessageTime?: string
}

export interface CreateSessionRequest {
  title?: string
  workingDirectory?: string
  agent_id?: string
}

export interface CreateSessionResponse {
  id: string
  title: string
  state: SessionState
  createdAt: string
}

export interface SessionStatus {
  state: SessionState
  messageCount: number
  tokenUsage: TokenUsage
}