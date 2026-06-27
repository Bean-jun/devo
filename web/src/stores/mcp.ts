import { ref } from 'vue'
import { defineStore } from 'pinia'
import type { McpTool, McpServer, AddServerRequest } from '@/types/mcp'
import { API_BASE } from '@/utils/constants'

export const useMcpStore = defineStore('mcp', () => {
  const tools = ref<McpTool[]>([])
  const servers = ref<McpServer[]>([])
  const isLoading = ref(false)
  const removingServerId = ref<string | null>(null)

  async function fetchTools(): Promise<void> {
    isLoading.value = true
    try {
      const res = await fetch(`${API_BASE}/mcp/tools`)
      if (!res.ok) throw new Error(`Failed to fetch MCP tools: ${res.status}`)
      const data = await res.json()
      const list = Array.isArray(data) ? data : (data.tools || [])
      tools.value = list.map((t: any) => ({
        tool_name: t.tool_name || t.name || '',
        server_id: t.server_id || t.server_name || '',
        description: t.description || '',
        input_schema: t.input_schema || t.parameters || {},
      }))
    } catch {
      throw new Error('获取 MCP 工具列表失败')
    } finally {
      isLoading.value = false
    }
  }

  async function fetchServers(): Promise<void> {
    isLoading.value = true
    try {
      const res = await fetch(`${API_BASE}/mcp/servers`)
      if (!res.ok) throw new Error(`Failed to fetch MCP servers: ${res.status}`)
      const data = await res.json()
      servers.value = (data.servers || []) as McpServer[]
    } catch {
      throw new Error('获取 MCP 服务器列表失败')
    } finally {
      isLoading.value = false
    }
  }

  async function toggleServer(serverId: string, enabled: boolean): Promise<void> {
    const res = await fetch(`${API_BASE}/mcp/servers/${serverId}/toggle`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ enabled }),
    })
    if (!res.ok) throw new Error('切换 MCP 服务器失败')
    await fetchServers()
    await fetchTools()
  }

  async function addServer(req: AddServerRequest): Promise<void> {
    const res = await fetch(`${API_BASE}/mcp/servers`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(req),
    })
    if (!res.ok) {
      const err = await res.json().catch(() => ({ error: '添加失败' }))
      throw new Error(err.error || '添加 MCP 服务器失败')
    }
    await fetchServers()
    await fetchTools()
  }

  async function removeServer(serverId: string, scope: string = 'project'): Promise<void> {
    removingServerId.value = serverId
    try {
      const res = await fetch(`${API_BASE}/mcp/servers/${serverId}?scope=${scope}`, {
        method: 'DELETE',
      })
      if (!res.ok) {
        const err = await res.json().catch(() => ({ error: '删除失败' }))
        throw new Error(err.error || '删除 MCP 服务器失败')
      }
      await fetchServers()
      await fetchTools()
    } finally {
      removingServerId.value = null
    }
  }

  function updateToolsFromEvent(newTools: McpTool[]): void {
    for (const tool of newTools) {
      const idx = tools.value.findIndex(t => t.tool_name === tool.tool_name)
      if (idx >= 0) {
        tools.value[idx] = { ...tools.value[idx], ...tool }
      } else {
        tools.value.push(tool)
      }
    }
  }

  return {
    tools,
    servers,
    isLoading,
    removingServerId,
    fetchTools,
    fetchServers,
    toggleServer,
    addServer,
    removeServer,
    updateToolsFromEvent,
  }
})