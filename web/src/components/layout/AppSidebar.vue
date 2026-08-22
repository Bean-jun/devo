<script setup lang="ts">
import AppIcon from '@/components/common/AppIcon.vue'
import { useAppSidebar } from './AppSidebarController'

const props = defineProps<{
  collapsed: boolean
}>()

const {
  uiStore,
  sessionStore,
  workspaces,
  currentWorkspace,
  sessions,
  currentSessionId,
  deleteTarget,
  confirmInput,
  confirmError,
  openDeleteConfirm,
  confirmDelete,
  cancelDelete,
  sessionDeleteTarget,
  openSessionDeleteConfirm,
  confirmSessionDelete,
  cancelSessionDelete,
  selectWorkspace,
  selectSession,
  isActiveWorkspace,
  showNewSessionDialog,
  newSessionTitle,
  newSession,
  confirmNewSession,
  cancelNewSession,
} = useAppSidebar(props)
</script>

<template>
  <nav class="app-sidebar" :class="{ collapsed: props.collapsed }">
    <template v-if="props.collapsed">
      <div class="collapsed-icons">
        <button
          v-for="ws in workspaces"
          :key="ws.id"
          class="collapsed-icon-btn"
          :class="{
            active: isActiveWorkspace(ws.id),
            removed: !ws.exists,
          }"
          :disabled="!ws.exists"
          :title="ws.exists ? (ws.name + '\n' + ws.path) : (ws.name + '\n已移除')"
          @click="ws.exists && selectWorkspace(ws)"
        >
          <AppIcon :name="ws.exists ? 'folder' : 'trash'" :size="16" />
        </button>
        <div class="collapsed-separator"></div>
        <button
          v-for="sess in sessions"
          :key="sess.id"
          class="collapsed-icon-btn"
          :class="{ active: currentSessionId === sess.id }"
          :title="sess.title"
          @click="selectSession(sess.id)"
        >
          <AppIcon name="chat-dots" :size="16" />
        </button>
        <button class="collapsed-icon-btn collapsed-add-btn" title="新建会话" @click="newSession">
          <AppIcon name="plus" :size="16" />
        </button>
      </div>
      <button class="sidebar-toggle-btn" @click="uiStore.toggleSidebar()" title="展开侧边栏">
        <AppIcon name="caret-right" :size="16" />
      </button>
    </template>

    <template v-else>
    <div class="sidebar-section">
      <div class="sidebar-section-title">工作区</div>
      <button
        v-for="ws in workspaces"
        :key="ws.id"
        class="sidebar-item workspace-item"
        :class="{
          active: isActiveWorkspace(ws.id),
          removed: !ws.exists,
        }"
        :disabled="!ws.exists"
        @click="ws.exists && selectWorkspace(ws)"
      >
        <AppIcon :name="ws.exists ? 'folder' : 'trash'" :size="16" class="sidebar-item-icon" />
        <div class="workspace-info">
          <span class="workspace-name">{{ ws.name }}</span>
          <span class="workspace-path">{{ ws.exists ? ws.path : '已移除' }}</span>
        </div>
        <span class="workspace-delete" @click.stop="openDeleteConfirm(ws)" title="移除"><AppIcon name="x" :size="12" /></span>
      </button>
      <div v-if="workspaces.length === 0" class="sidebar-empty">
        暂无工作区
      </div>
    </div>

    <div class="sidebar-section sidebar-sessions">
      <div class="sidebar-section-title">会话</div>
      <div class="sessions-body">
        <div class="sessions-list" v-if="sessions.length > 0">
          <button
            v-for="sess in sessions"
            :key="sess.id"
            class="sidebar-item session-item"
            :class="{ active: currentSessionId === sess.id }"
            :title="sess.title"
            @click="selectSession(sess.id)"
          >
            <AppIcon name="chat-dots" :size="16" class="sidebar-item-icon" />
            <span class="sidebar-item-label session-title">{{ sess.title }}</span>
            <span class="session-delete" @click.stop="openSessionDeleteConfirm(sess)" title="删除"><AppIcon name="x" :size="12" /></span>
          </button>
        </div>
        <div v-else class="sidebar-empty">
          暂无会话
        </div>
      </div>
      <button class="sidebar-item sidebar-add-btn" @click="newSession">
        <AppIcon name="plus" :size="16" class="sidebar-item-icon" />
        <span class="sidebar-item-label">新建会话</span>
      </button>
    </div>
    <button class="sidebar-toggle-btn sidebar-toggle-expand" @click="uiStore.toggleSidebar()" title="折叠侧边栏">
      <AppIcon name="caret-left" :size="16" />
    </button>
    </template>
  </nav>

  <Teleport to="body">
    <div v-if="deleteTarget" class="confirm-overlay" @click.self="cancelDelete">
      <div class="confirm-dialog" :class="{ 'confirm-dialog-sm': !deleteTarget.exists }">
        <h3 class="confirm-title">
          <template v-if="deleteTarget.exists">
            <AppIcon name="warning" :size="16" style="display: inline-block; vertical-align: middle; margin-right: 6px;" />
            删除工作区
          </template>
          <template v-else>
            移除工作区记录
          </template>
        </h3>
        <p class="confirm-desc">
          <template v-if="deleteTarget.exists">
            此操作将删除 <strong>{{ deleteTarget.name }}</strong> 下的所有会话和记录，不可恢复。
          </template>
          <template v-else>
            该目录已不存在，将仅从列表中移除 <strong>{{ deleteTarget.name }}</strong> 的记录。
          </template>
        </p>
        <template v-if="deleteTarget.exists">
          <p class="confirm-path-label">请输入以下路径以确认删除：</p>
          <code class="confirm-path">{{ deleteTarget.path }}</code>
          <input
            v-model="confirmInput"
            class="confirm-input"
            :class="{ 'confirm-input-error': confirmError }"
            placeholder="输入路径以确认..."
            @keydown.enter="confirmDelete"
            @keydown.escape.stop="cancelDelete"
          />
          <p v-if="confirmError" class="confirm-error">{{ confirmError }}</p>
        </template>
        <div class="confirm-actions">
          <button class="confirm-btn confirm-btn-cancel" @click="cancelDelete">取消</button>
          <button
            class="confirm-btn confirm-btn-danger"
            :disabled="deleteTarget.exists && confirmInput !== deleteTarget.path"
            @click="confirmDelete"
          >
            {{ deleteTarget.exists ? '删除' : '确认移除' }}
          </button>
        </div>
      </div>
    </div>
  </Teleport>

  <Teleport to="body">
    <div v-if="sessionDeleteTarget" class="confirm-overlay" @click.self="cancelSessionDelete">
      <div class="confirm-dialog confirm-dialog-sm">
        <h3 class="confirm-title">删除会话</h3>
        <p class="confirm-desc">
          确定要删除会话 <strong>{{ sessionDeleteTarget.title }}</strong> 吗？此操作不可恢复。
        </p>
        <div class="confirm-actions">
          <button class="confirm-btn confirm-btn-cancel" @click="cancelSessionDelete">取消</button>
          <button class="confirm-btn confirm-btn-danger" @click="confirmSessionDelete">删除</button>
        </div>
      </div>
    </div>
  </Teleport>

  <Teleport to="body">
    <div v-if="showNewSessionDialog" class="confirm-overlay" @click.self="cancelNewSession">
      <div class="confirm-dialog confirm-dialog-sm">
        <h3 class="confirm-title">新建会话</h3>
        <label class="new-session-label">会话名称</label>
        <input
          v-model="newSessionTitle"
          class="new-session-input"
          placeholder="输入名称（可选）"
          @keydown.enter="confirmNewSession"
          @keydown.escape.stop="cancelNewSession"
        />
        <label v-if="sessionStore.agents.length > 0" class="new-session-label" style="margin-top: 12px;">选择 Agent</label>
        <select
          v-if="sessionStore.agents.length > 0"
          v-model="sessionStore.selectedAgentId"
          class="new-session-input agent-select"
        >
          <option
            v-for="agent in sessionStore.agents"
            :key="agent.id"
            :value="agent.id"
          >{{ agent.name }}</option>
        </select>
        <div class="confirm-actions">
          <button class="confirm-btn confirm-btn-cancel" @click="cancelNewSession">取消</button>
          <button class="confirm-btn confirm-btn-primary" @click="confirmNewSession">创建</button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped src="./AppSidebar.css">
</style>