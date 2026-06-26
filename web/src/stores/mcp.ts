import { ref } from 'vue'
import { defineStore } from 'pinia'
import { API_BASE } from '@/utils/constants'

export interface McpTool {
  name: string
  description: string
  parameters: Record<string, unknown>
  serverName: string
  enabled: boolean
}

export const useMcpStore = defineStore('mcp', () => {
  const tools = ref<McpTool[]>([])
  const isLoading = ref(false)

  async function fetchTools(): Promise<void> {
    isLoading.value = true
    try {
      const res = await fetch(`${API_BASE}/mcp/tools`)
      if (!res.ok) throw new Error(`Failed to fetch MCP tools: ${res.status}`)
      const data = await res.json()
      const list = Array.isArray(data) ? data : (data.tools || [])
      tools.value = list.map((t: any) => ({
        name: t.tool_name || t.name || '',
        description: t.description || '',
        parameters: t.input_schema || t.parameters || {},
        serverName: t.server_id || t.server_name || '',
        enabled: t.enabled !== false,
      }))
    } catch {
      throw new Error('获取 MCP 工具列表失败')
    } finally {
      isLoading.value = false
    }
  }

  function updateToolsFromEvent(newTools: McpTool[]): void {
    for (const tool of newTools) {
      const idx = tools.value.findIndex(t => t.name === tool.name)
      if (idx >= 0) {
        tools.value[idx] = { ...tools.value[idx], ...tool }
      } else {
        tools.value.push(tool)
      }
    }
  }

  return {
    tools,
    isLoading,
    fetchTools,
    updateToolsFromEvent,
  }
})