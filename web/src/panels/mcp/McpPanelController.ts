import { ref, onMounted } from 'vue'
import { useMcpStore } from '@/stores/mcp'
import type { McpServer } from '@/types/mcp'
import ConfirmDeleteDialog from '@/components/modal/ConfirmDeleteDialog.vue'
import AppIcon from '@/components/common/AppIcon.vue'

function statusLabel(status: string): string {
  switch (status) {
    case 'connected': return '已连接'
    case 'error': return '错误'
    default: return '未连接'
  }
}

function statusClass(status: string): string {
  return `status-${status}`
}

function sourceLabel(source: string): string {
  return source === 'global' ? '全局' : '项目'
}

export function useMcpPanel() {
  const mcpStore = useMcpStore()

  const error = ref<string | null>(null)
  const expandedServers = ref<Set<string>>(new Set())

  const showDeleteDialog = ref(false)
  const deletingServer = ref<McpServer | null>(null)

  const showAddForm = ref(false)
  const addForm = ref({
    server_id: '',
    endpoint: '',
    transport: 'sse',
    scope: 'project' as 'project' | 'global',
  })
  const addError = ref<string | null>(null)
  const adding = ref(false)

  function resetAddForm() {
    addForm.value = { server_id: '', endpoint: '', transport: 'sse', scope: 'project' }
    addError.value = null
  }

  function toggleServerExpand(serverId: string) {
    if (expandedServers.value.has(serverId)) {
      expandedServers.value.delete(serverId)
    } else {
      expandedServers.value.add(serverId)
    }
  }

  async function handleToggleServer(server: McpServer) {
    try {
      const isConnected = server.status === 'connected'
      await mcpStore.toggleServer(server.server_id, !isConnected)
    } catch (e: any) {
      error.value = e.message || '切换失败'
    }
  }

  async function handleAddServer() {
    if (!addForm.value.server_id.trim() || !addForm.value.endpoint.trim()) {
      addError.value = '请填写服务器 ID 和端点地址'
      return
    }
    adding.value = true
    addError.value = null
    try {
      await mcpStore.addServer({
        server_id: addForm.value.server_id.trim(),
        endpoint: addForm.value.endpoint.trim(),
        transport: addForm.value.transport,
        scope: addForm.value.scope,
      })
      showAddForm.value = false
      resetAddForm()
    } catch (e: any) {
      addError.value = e.message || '添加失败'
    } finally {
      adding.value = false
    }
  }

  async function handleRemoveServer(server: McpServer) {
    deletingServer.value = server
    showDeleteDialog.value = true
  }

  async function confirmDelete() {
    if (!deletingServer.value) return
    try {
      await mcpStore.removeServer(deletingServer.value.server_id, deletingServer.value.source)
      showDeleteDialog.value = false
      deletingServer.value = null
    } catch (e: any) {
      error.value = e.message || '删除失败'
    }
  }

  async function handleRefresh() {
    try {
      await mcpStore.fetchServers()
    } catch (e: any) {
      error.value = e.message || '刷新失败'
    }
  }

  function cancelDelete() {
    showDeleteDialog.value = false
    deletingServer.value = null
  }

  onMounted(async () => {
    try {
      await mcpStore.fetchServers()
    } catch (e: any) {
      error.value = e.message
    }
  })

  return {
    mcpStore,
    error,
    expandedServers,
    showDeleteDialog,
    deletingServer,
    showAddForm,
    addForm,
    addError,
    adding,
    resetAddForm,
    toggleServerExpand,
    handleToggleServer,
    handleAddServer,
    handleRemoveServer,
    confirmDelete,
    handleRefresh,
    cancelDelete,
    statusLabel,
    statusClass,
    sourceLabel,
  }
}