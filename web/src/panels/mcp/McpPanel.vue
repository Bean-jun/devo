<script setup lang="ts">
import AppIcon from '@/components/common/AppIcon.vue'
import ConfirmDeleteDialog from '@/components/modal/ConfirmDeleteDialog.vue'
import { useMcpPanel } from './McpPanelController'

const {
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
} = useMcpPanel()
</script>

<template>
  <div class="mcp-panel">
    <div class="mcp-header">
      <h3 class="mcp-title">MCP 服务器</h3>
      <div class="header-actions">
        <button class="reload-btn" :class="{ spinning: mcpStore.isLoading }" :disabled="mcpStore.isLoading" @click="handleRefresh" title="刷新服务器列表">
          <AppIcon name="arrow-clockwise" :size="16" />
        </button>
        <button class="add-btn" @click="showAddForm = !showAddForm">
          {{ showAddForm ? '取消' : '添加' }}
        </button>
      </div>
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
            >{{ mcpStore.removingServerId === server.server_id ? '...' : '' }}<template v-if="!mcpStore.removingServerId || mcpStore.removingServerId !== server.server_id"><AppIcon name="x" :size="12" /></template></button>
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

<style scoped src="./McpPanel.css">
</style>