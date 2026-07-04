export type ToolStatus = 'pending' | 'executing' | 'success' | 'failed' | 'rejected'
export type RiskLevel = 'low' | 'medium' | 'high'

export interface ToolCall {
  id: string
  name: string
  parameters: Record<string, unknown>
  result?: ToolResult
  status: ToolStatus
  riskLevel?: RiskLevel
  duration?: number
  approvalId?: string
  streamingOutput?: string
  stage?: string
}

export interface ToolResult {
  success: boolean
  error?: string
  bytesWritten?: number
  path?: string
  stdout?: string
  stderr?: string
  exitCode?: number
  [key: string]: unknown
}

export interface ToolDefinition {
  name: string
  description: string
  parameters: ToolParameter[]
}

export interface ToolParameter {
  name: string
  type: string
  description: string
  required: boolean
}