export type SessionState =
  | 'idle'
  | 'processing'
  | 'awaiting_approval'
  | 'paused'
  | 'completed'
  | 'archived'

export type TrustLevel = 'always_ask' | 'session_trust' | 'full_trust'

export type ApprovalPolicy = 'always_ask' | 'session_trust' | 'full_trust'

export interface TokenUsage {
  input: number
  output: number
}

export interface Session {
  id: string
  title: string
  state: SessionState
  workingDirectory: string
  createdAt: string
  lastActiveAt: string
  messageCount: number
  tokenUsage: TokenUsage
  trustLevel: TrustLevel
  approvalPolicy: ApprovalPolicy
  maxContextTokens?: number
}

export interface CreateSessionRequest {
  title?: string
  workingDirectory?: string
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