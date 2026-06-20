export interface ApprovalRequest {
  id: string
  sessionId: string
  toolCallId: string
  toolName: string
  riskLevel: 'low' | 'medium' | 'high'
  parameters: Record<string, unknown>
  diff?: string
  command?: string
  filePath?: string
  timeout: number
  createdAt: string
}

export interface ApprovalResponse {
  approved: boolean
  trustPolicy?: 'session_trust' | 'full_trust'
}

export interface ApprovalDecision {
  approvalId: string
  approved: boolean
  trustAllInSession?: boolean
}