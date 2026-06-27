export interface McpTool {
  tool_name: string
  server_id: string
  description: string
  input_schema: Record<string, unknown>
}

export interface McpServer {
  server_id: string
  source: string
  endpoint: string
  transport: string
  status: 'connected' | 'disconnected' | 'error'
  tool_count: number
  tools: McpTool[]
  error_msg?: string
}

export interface McpServersResponse {
  servers: McpServer[]
}

export interface McpToolsResponse {
  tools: McpTool[]
}

export interface ToggleServerRequest {
  enabled: boolean
}

export interface ToggleServerResponse {
  server_id: string
  status: string
}

export interface AddServerRequest {
  server_id: string
  endpoint: string
  transport: string
  scope: 'project' | 'global'
}

export interface AddServerResponse {
  server_id: string
  status: string
}

export interface RemoveServerResponse {
  server_id: string
  status: string
}