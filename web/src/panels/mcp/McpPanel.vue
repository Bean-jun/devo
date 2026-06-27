<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useMcpStore } from '@/stores/mcp'
import type { McpServer } from '@/types/mcp'
import ConfirmDeleteDialog from '@/components/modal/ConfirmDeleteDialog.vue'

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

function cancelDelete() {
  showDeleteDialog.value = false
  deletingServer.value = null
}

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

onMounted(async () => {
  try {
    await mcpStore.fetchServers()
  } catch (e: any) {
    error.value = e.message
  }
})
</script>

<template>
  <div class="mcp-panel">
    <div class="mcp-header">
      <h3 class="mcp-title">MCP 服务器</h3>
      <button class="add-btn" @click="showAddForm = !showAddForm">
        {{ showAddForm ? '取消' : '添加' }}
      </button>
    </div>

    <div v-if="showAddForm" class="add-form">
      <div class="form-field">
        <label>服务器 ID</label>
        <input
          v-model="addForm.server_id"
          type="text"
          placeholder="my-mcp-server"
          class="form-input"
        />
      </div>
      <div class="form-field">
        <label>端点地址</label>
        <input
          v-model="addForm.endpoint"
          type="text"
          placeholder="http://localhost:8080"
          class="form-input"
        />
      </div>
      <div class="form-row">
        <div class="form-field">
          <label>传输协议</label>
          <select v-model="addForm.transport" class="form-select">
            <option value="sse">SSE</option>
            <option value="stdio">STDIO</option>
          </select>
        </div>
        <div class="form-field">
          <label>作用域</label>
          <select v-model="addForm.scope" class="form-select">
            <option value="project">项目</option>
            <option value="global">全局</option>
          </select>
        </div>
      </div>
      <div v-if="addError" class="form-error">{{ addError }}</div>
      <button class="submit-btn" :disabled="adding" @click="handleAddServer">
        {{ adding ? '添加中...' : '添加服务器' }}
      </button>
    </div>

    <div v-if="error" class="mcp-error">{{ error }}</div>
    <div v-else-if="mcpStore.isLoading" class="mcp-loading">加载中...</div>
    <div v-else-if="mcpStore.servers.length === 0" class="mcp-empty">
      <p>暂无 MCP 服务器</p>
      <p class="mcp-hint">点击"添加"按钮添加 MCP 服务器</p>
    </div>

    <div v-else class="mcp-server-list">
      <div
        v-for="server in mcpStore.servers"
        :key="server.server_id"
        class="mcp-server-card"
      >
        <div class="server-header" @click="toggleServerExpand(server.server_id)">
          <div class="server-info">
            <span class="server-name">{{ server.server_id }}</span>
          </div>
          <div class="server-actions">
            <span class="source-tag" :class="'source-' + server.source">{{ sourceLabel(server.source) }}</span>
            <span class="tool-count">{{ server.tool_count }} 工具</span>
            <span class="server-status" :class="statusClass(server.status)">
              {{ statusLabel(server.status) }}
            </span>
            <button
              class="server-toggle"
              :class="{ active: server.status === 'connected' }"
              @click.stop="handleToggleServer(server)"
            >
              {{ server.status === 'connected' ? 'ON' : 'OFF' }}
            </button>
            <button
              class="server-remove"
              :disabled="mcpStore.removingServerId === server.server_id"
              @click.stop="handleRemoveServer(server)"
              :title="mcpStore.removingServerId === server.server_id ? '删除中...' : '删除服务器'"
            >{{ mcpStore.removingServerId === server.server_id ? '...' : '✕' }}</button>
          </div>
        </div>

        <div v-if="server.error_msg" class="server-error">
          <span class="error-icon">!</span>
          {{ server.error_msg }}
        </div>

        <div v-if="expandedServers.has(server.server_id)" class="server-tools">
          <div class="server-meta-row">
            <span class="meta-label">端点:</span>
            <code class="meta-value">{{ server.endpoint }}</code>
          </div>
          <div
            v-for="tool in server.tools"
            :key="tool.tool_name"
            class="tool-item"
          >
            <div class="tool-name">{{ tool.tool_name }}</div>
            <div class="tool-desc">{{ tool.description }}</div>
          </div>
          <div v-if="server.tools.length === 0" class="tool-empty">
            暂无工具
          </div>
        </div>
      </div>
    </div>
  </div>

    <ConfirmDeleteDialog
      :visible="showDeleteDialog"
      :server-name="deletingServer?.server_id ?? ''"
      :deleting="mcpStore.removingServerId === deletingServer?.server_id"
      @confirm="confirmDelete"
      @cancel="cancelDelete"
    />
</template>

<style scoped>
.mcp-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.mcp-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  border-bottom: 1px solid var(--color-border);
}

.mcp-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin: 0;
}

.add-btn {
  height: 28px;
  padding: 0 8px;
  border: 1px solid var(--color-border);
  border-radius: 4px;
  background: transparent;
  color: var(--color-text-secondary);
  cursor: pointer;
  font-size: 11px;
  transition: all var(--transition-fast);
}

.add-btn:hover {
  background: var(--color-bg-hover);
  color: var(--color-accent);
  border-color: var(--color-accent);
}

.add-form {
  padding: 10px 12px;
  border-bottom: 1px solid var(--color-border);
  background: var(--color-bg-secondary);
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.form-field {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.form-field label {
  font-size: 11px;
  color: var(--color-text-secondary);
  font-weight: 500;
}

.form-input,
.form-select {
  padding: 5px 8px;
  border: 1px solid var(--color-border);
  border-radius: 4px;
  background: var(--color-bg-primary);
  color: var(--color-text-primary);
  font-size: 12px;
  font-family: var(--font-mono);
}

.form-input:focus,
.form-select:focus {
  outline: none;
  border-color: var(--color-accent);
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.15);
}

.form-row {
  display: flex;
  gap: 8px;
}

.form-row .form-field {
  flex: 1;
}

.form-error {
  font-size: 11px;
  color: var(--color-error);
}

.submit-btn {
  padding: 6px 0;
  border: none;
  border-radius: 4px;
  background: var(--color-accent);
  color: white;
  cursor: pointer;
  font-size: 12px;
  font-weight: 500;
  transition: opacity var(--transition-fast);
}

.submit-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.submit-btn:hover:not(:disabled) {
  opacity: 0.9;
}

.mcp-error {
  padding: 16px;
  color: var(--color-error);
  font-size: 12px;
}

.mcp-loading {
  padding: 16px;
  color: var(--color-text-tertiary);
  font-size: 12px;
  text-align: center;
}

.mcp-empty {
  padding: 24px 16px;
  text-align: center;
  color: var(--color-text-tertiary);
  font-size: 12px;
}

.mcp-hint {
  margin-top: 8px;
  font-size: 11px;
  color: var(--color-text-tertiary);
  opacity: 0.7;
}

.mcp-server-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}

.mcp-server-card {
  border: 1px solid var(--color-border);
  border-radius: 6px;
  margin-bottom: 6px;
  overflow: hidden;
  transition: border-color var(--transition-fast);
}

.mcp-server-card:hover {
  border-color: var(--color-accent);
}

.server-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 10px;
  cursor: pointer;
  user-select: none;
}

.server-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.server-name {
  font-size: 13px;
  font-weight: 500;
  color: var(--color-text-primary);
}

.source-tag {
  font-size: 10px;
  padding: 0 5px;
  border-radius: 3px;
  border: 1px solid;
  font-weight: 500;
  flex-shrink: 0;
  line-height: 1.6;
}

.source-project {
  background: rgba(59, 130, 246, 0.1);
  border-color: rgba(59, 130, 246, 0.25);
  color: #60a5fa;
}

.source-global {
  background: rgba(139, 92, 246, 0.1);
  border-color: rgba(139, 92, 246, 0.25);
  color: #a78bfa;
}

.server-actions {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}

.tool-count {
  font-size: 10px;
  color: var(--color-text-tertiary);
  padding: 2px 6px;
  background: var(--color-bg-tertiary);
  border-radius: 3px;
}

.server-status {
  font-size: 10px;
  padding: 2px 6px;
  border-radius: 3px;
  font-weight: 500;
}

.status-connected {
  background: rgba(34, 197, 94, 0.15);
  color: #22c55e;
}

.status-disconnected {
  background: var(--color-bg-tertiary);
  color: var(--color-text-tertiary);
}

.status-error {
  background: rgba(239, 68, 68, 0.15);
  color: #ef4444;
}

.server-toggle {
  width: 36px;
  height: 24px;
  border: 1px solid var(--color-border);
  border-radius: 12px;
  background: var(--color-bg-tertiary);
  color: var(--color-text-tertiary);
  cursor: pointer;
  font-size: 10px;
  font-weight: 600;
  transition: all var(--transition-fast);
  flex-shrink: 0;
}

.server-toggle.active {
  background: var(--color-accent);
  border-color: var(--color-accent);
  color: white;
}

.server-remove {
  background: none;
  border: none;
  color: var(--color-text-tertiary);
  cursor: pointer;
  font-size: 12px;
  padding: 2px 4px;
  border-radius: 4px;
  transition: all var(--transition-fast);
}

.server-remove:hover {
  color: var(--color-error);
  background: var(--color-bg-hover);
}

.server-remove:disabled {
  color: var(--color-text-tertiary);
  opacity: 0.5;
  cursor: not-allowed;
}

.server-error {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 10px;
  background: rgba(239, 68, 68, 0.08);
  color: var(--color-error);
  font-size: 11px;
  border-top: 1px solid var(--color-border);
}

.error-icon {
  font-weight: 700;
  font-size: 12px;
}

.server-tools {
  border-top: 1px solid var(--color-border);
  padding: 6px 10px;
  background: var(--color-bg-secondary);
}

.server-meta-row {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 8px;
  font-size: 11px;
}

.meta-label {
  color: var(--color-text-tertiary);
}

.meta-value {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--color-text-secondary);
  background: var(--color-bg-tertiary);
  padding: 1px 5px;
  border-radius: 3px;
}

.tool-item {
  padding: 6px 0;
  border-bottom: 1px solid var(--color-border);
}

.tool-item:last-child {
  border-bottom: none;
}

.tool-name {
  font-size: 12px;
  font-weight: 500;
  color: var(--color-text-primary);
  font-family: var(--font-mono);
}

.tool-desc {
  font-size: 11px;
  color: var(--color-text-tertiary);
  margin-top: 2px;
  line-height: 1.4;
}

.tool-empty {
  font-size: 11px;
  color: var(--color-text-tertiary);
  text-align: center;
  padding: 8px 0;
}
</style>