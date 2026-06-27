<script setup lang="ts">
import { computed, ref, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { useSessionStore } from '@/stores/session'
import { useUiStore } from '@/stores/ui'
import { API_BASE } from '@/utils/constants'

const props = defineProps<{
  collapsed: boolean
}>()

const router = useRouter()
const sessionStore = useSessionStore()
const uiStore = useUiStore()

const workspaces = computed(() => uiStore.workspaceList)

const currentWorkspace = computed(() =>
  workspaces.value.find(w => w.id === uiStore.activeWorkspace)
)

const sessions = computed(() => {
  return sessionStore.sessions
})

const currentSessionId = computed(() => sessionStore.currentSession?.id)

const deleteTarget = ref<{ id: string; name: string; path: string } | null>(null)
const confirmInput = ref('')
const confirmError = ref('')

function openDeleteConfirm(ws: { id: string; name: string; path: string }) {
  deleteTarget.value = ws
  confirmInput.value = ''
  confirmError.value = ''
  nextTick(() => {
    const input = document.querySelector('.confirm-input') as HTMLInputElement
    input?.focus()
  })
}

function confirmDelete() {
  if (!deleteTarget.value) return
  if (confirmInput.value !== deleteTarget.value.path) {
    confirmError.value = '输入的路径不匹配'
    return
  }
  uiStore.removeWorkspace(deleteTarget.value.id)
  deleteTarget.value = null
  confirmInput.value = ''
  confirmError.value = ''
}

function cancelDelete() {
  deleteTarget.value = null
  confirmInput.value = ''
  confirmError.value = ''
}

const sessionDeleteTarget = ref<{ id: string; title: string } | null>(null)

function openSessionDeleteConfirm(sess: { id: string; title: string }) {
  sessionDeleteTarget.value = sess
}

async function confirmSessionDelete() {
  if (!sessionDeleteTarget.value) return
  await sessionStore.deleteSession(sessionDeleteTarget.value.id)
  sessionDeleteTarget.value = null
}

function cancelSessionDelete() {
  sessionDeleteTarget.value = null
}

function selectWorkspace(ws: { id: string; name: string; path: string }) {
  uiStore.setActiveWorkspace(ws.id)
  uiStore.setActiveRightTab('files')
  fetch(`${API_BASE}/current-workspace`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ working_directory: ws.path }),
  }).catch(() => {})
}

function selectSession(sessionId: string) {
  sessionStore.switchSessionById(sessionId)
  router.push('/chat')
}

function isActiveWorkspace(wsId: string): boolean {
  return uiStore.activeWorkspace === wsId
}

const showNewSessionDialog = ref(false)
const newSessionTitle = ref('')

function newSession() {
  newSessionTitle.value = ''
  showNewSessionDialog.value = true
  nextTick(() => {
    const input = document.querySelector('.new-session-input') as HTMLInputElement
    input?.focus()
  })
}

async function confirmNewSession() {
  const dir = currentWorkspace.value?.path || sessionStore.workingDirectory
  await sessionStore.createSession({
    workingDirectory: dir,
    title: newSessionTitle.value.trim() || undefined,
  })
  showNewSessionDialog.value = false
  router.push('/chat')
}

function cancelNewSession() {
  showNewSessionDialog.value = false
}
</script>

<template>
  <nav class="app-sidebar" :class="{ collapsed: props.collapsed }">
    <template v-if="props.collapsed">
      <div class="collapsed-icons">
        <button
          v-for="ws in workspaces"
          :key="ws.id"
          class="collapsed-icon-btn"
          :class="{ active: isActiveWorkspace(ws.id) }"
          :title="ws.name + '\n' + ws.path"
          @click="selectWorkspace(ws)"
        >
          📁
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
          💬
        </button>
        <button class="collapsed-icon-btn collapsed-add-btn" title="新建会话" @click="newSession">
          +
        </button>
      </div>
      <button class="sidebar-toggle-btn" @click="uiStore.toggleSidebar()" title="展开侧边栏">
        ▶
      </button>
    </template>

    <template v-else>
    <div class="sidebar-section">
      <div class="sidebar-section-title">工作区</div>
      <button
        v-for="ws in workspaces"
        :key="ws.id"
        class="sidebar-item workspace-item"
        :class="{ active: isActiveWorkspace(ws.id) }"
        @click="selectWorkspace(ws)"
      >
        <span class="sidebar-item-icon">📁</span>
        <div class="workspace-info">
          <span class="workspace-name">{{ ws.name }}</span>
          <span class="workspace-path">{{ ws.path }}</span>
        </div>
        <span class="workspace-delete" @click.stop="openDeleteConfirm(ws)" title="移除">✕</span>
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
            <span class="sidebar-item-icon">💬</span>
            <span class="sidebar-item-label session-title">{{ sess.title }}</span>
            <span class="session-delete" @click.stop="openSessionDeleteConfirm(sess)" title="删除">✕</span>
          </button>
        </div>
        <div v-else class="sidebar-empty">
          暂无会话
        </div>
      </div>
      <button class="sidebar-item sidebar-add-btn" @click="newSession">
        <span class="sidebar-item-icon">+</span>
        <span class="sidebar-item-label">新建会话</span>
      </button>
    </div>
    <button class="sidebar-toggle-btn sidebar-toggle-expand" @click="uiStore.toggleSidebar()" title="折叠侧边栏">
      ◀
    </button>
    </template>
  </nav>

  <Teleport to="body">
    <div v-if="deleteTarget" class="confirm-overlay" @click.self="cancelDelete">
      <div class="confirm-dialog">
        <h3 class="confirm-title">⚠️ 删除工作区</h3>
        <p class="confirm-desc">
          此操作将删除 <strong>{{ deleteTarget.name }}</strong> 下的所有会话和记录，不可恢复。
        </p>
        <p class="confirm-path-label">请输入以下路径以确认删除：</p>
        <code class="confirm-path">{{ deleteTarget.path }}</code>
        <input
          v-model="confirmInput"
          class="confirm-input"
          :class="{ 'confirm-input-error': confirmError }"
          placeholder="输入路径以确认..."
          @keydown.enter="confirmDelete"
          @keydown.escape="cancelDelete"
        />
        <p v-if="confirmError" class="confirm-error">{{ confirmError }}</p>
        <div class="confirm-actions">
          <button class="confirm-btn confirm-btn-cancel" @click="cancelDelete">取消</button>
          <button
            class="confirm-btn confirm-btn-danger"
            :disabled="confirmInput !== deleteTarget.path"
            @click="confirmDelete"
          >
            删除
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
          @keydown.escape="cancelNewSession"
        />
        <div class="confirm-actions">
          <button class="confirm-btn confirm-btn-cancel" @click="cancelNewSession">取消</button>
          <button class="confirm-btn confirm-btn-primary" @click="confirmNewSession">创建</button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.app-sidebar {
  width: 100%;
  height: 100vh;
  background: var(--color-bg-secondary);
  border-right: 1px solid var(--color-border);
  display: flex;
  flex-direction: column;
  user-select: none;
  overflow: hidden;
}

.sidebar-section {
  padding: 4px 8px;
}

.sidebar-section:last-child {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
}

.sidebar-section-title {
  padding: 8px 12px 4px;
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: var(--color-text-tertiary);
}

.sidebar-item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 8px 12px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all var(--transition-fast);
  font-size: 13px;
  text-align: left;
}

.sidebar-item:hover {
  background: var(--color-bg-hover);
  color: var(--color-text-primary);
}

.sidebar-item.active {
  background: var(--color-accent);
  color: white;
}

.sidebar-item-icon {
  font-size: 16px;
  flex-shrink: 0;
  width: 20px;
  text-align: center;
}

.workspace-delete {
  margin-left: auto;
  font-size: 11px;
  opacity: 0;
  color: var(--color-text-tertiary);
  padding: 2px 4px;
  border-radius: 3px;
  transition: opacity 0.15s, color 0.15s;
}

.sidebar-item:hover .workspace-delete {
  opacity: 1;
}

.workspace-delete:hover {
  color: var(--color-error);
  background: rgba(255, 0, 0, 0.1);
}

.session-delete {
  margin-left: auto;
  font-size: 11px;
  opacity: 0;
  color: var(--color-text-tertiary);
  padding: 2px 4px;
  border-radius: 3px;
  transition: opacity 0.15s, color 0.15s;
}

.sidebar-item:hover .session-delete {
  opacity: 1;
}

.session-delete:hover {
  color: var(--color-error);
  background: rgba(255, 0, 0, 0.1);
}

.workspace-info {
  display: flex;
  flex-direction: column;
  min-width: 0;
  overflow: hidden;
  gap: 1px;
}

.workspace-name {
  font-size: 13px;
  line-height: 1.3;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.workspace-path {
  font-size: 10px;
  line-height: 1.2;
  color: var(--color-text-tertiary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sidebar-item.active .workspace-path {
  color: rgba(255, 255, 255, 0.6);
}

.sidebar-item-label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sidebar-sessions {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
}

.sessions-body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}

.sessions-list {
  min-height: 0;
}

.sidebar-empty {
  padding: 12px;
  font-size: 12px;
  color: var(--color-text-tertiary);
  text-align: center;
}

.sidebar-add-btn {
  margin-top: 4px;
  border-top: 1px solid var(--color-border);
  border-radius: 0;
  padding: 10px 12px;
  color: var(--color-accent);
}

.sidebar-add-btn:hover {
  background: var(--color-bg-hover);
  color: var(--color-accent);
}

.confirm-overlay {
  position: fixed;
  inset: 0;
  z-index: 9999;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  backdrop-filter: blur(2px);
}

.confirm-dialog {
  background: var(--color-bg-primary);
  border: 1px solid var(--color-border);
  border-radius: 12px;
  padding: 24px;
  width: 440px;
  max-width: 90vw;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
}

.confirm-dialog-sm {
  width: 360px;
}

.confirm-title {
  margin: 0 0 12px;
  font-size: 16px;
  font-weight: 600;
  color: var(--color-text-primary);
}

.confirm-desc {
  margin: 0 0 16px;
  font-size: 13px;
  line-height: 1.6;
  color: var(--color-text-secondary);
}

.confirm-desc strong {
  color: var(--color-text-primary);
  font-weight: 600;
}

.confirm-path-label {
  margin: 0 0 6px;
  font-size: 12px;
  color: var(--color-text-tertiary);
}

.confirm-path {
  display: block;
  padding: 8px 12px;
  margin-bottom: 12px;
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: 6px;
  font-size: 12px;
  font-family: 'SF Mono', 'Cascadia Code', 'Consolas', monospace;
  color: var(--color-accent);
  word-break: break-all;
}

.confirm-input {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  background: var(--color-bg-primary);
  color: var(--color-text-primary);
  font-size: 13px;
  font-family: 'SF Mono', 'Cascadia Code', 'Consolas', monospace;
  outline: none;
  transition: border-color 0.15s;
  box-sizing: border-box;
}

.confirm-input:focus {
  border-color: var(--color-accent);
}

.confirm-input-error {
  border-color: var(--color-error) !important;
}

.confirm-error {
  margin: 6px 0 0;
  font-size: 12px;
  color: var(--color-error);
}

.confirm-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 16px;
}

.confirm-btn {
  padding: 8px 20px;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s;
}

.confirm-btn-cancel {
  background: var(--color-bg-secondary);
  color: var(--color-text-secondary);
}

.confirm-btn-cancel:hover {
  background: var(--color-bg-hover);
  color: var(--color-text-primary);
}

.confirm-btn-danger {
  background: var(--color-error);
  color: white;
  border-color: var(--color-error);
}

.confirm-btn-danger:hover:not(:disabled) {
  opacity: 0.9;
}

.confirm-btn-danger:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.confirm-btn-primary {
  background: var(--color-accent);
  color: white;
  border-color: var(--color-accent);
}

.confirm-btn-primary:hover {
  opacity: 0.9;
}

.new-session-label {
  display: block;
  margin: 0 0 6px;
  font-size: 12px;
  color: var(--color-text-tertiary);
}

.new-session-input {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  background: var(--color-bg-primary);
  color: var(--color-text-primary);
  font-size: 13px;
  outline: none;
  transition: border-color 0.15s;
  box-sizing: border-box;
}

.new-session-input:focus {
  border-color: var(--color-accent);
}

.new-session-input::placeholder {
  color: var(--color-text-tertiary);
}

.app-sidebar.collapsed {
  align-items: center;
  padding: 4px 0;
}

.collapsed-icons {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  padding: 8px 4px;
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
}

.collapsed-icon-btn {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  border-radius: 6px;
  background: transparent;
  font-size: 16px;
  cursor: pointer;
  color: var(--color-text-secondary);
  transition: all var(--transition-fast);
  flex-shrink: 0;
}

.collapsed-icon-btn:hover {
  background: var(--color-bg-hover);
  color: var(--color-text-primary);
}

.collapsed-icon-btn.active {
  background: var(--color-accent);
  color: white;
}

.collapsed-separator {
  width: 20px;
  height: 1px;
  background: var(--color-border);
  margin: 4px 0;
}

.collapsed-add-btn {
  border: 1px dashed var(--color-border);
  font-size: 14px;
  color: var(--color-text-tertiary);
}

.collapsed-add-btn:hover {
  border-color: var(--color-accent);
  color: var(--color-accent);
}

.sidebar-toggle-btn {
  width: 100%;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  border-top: 1px solid var(--color-border);
  background: transparent;
  color: var(--color-text-tertiary);
  font-size: 10px;
  cursor: pointer;
  transition: all var(--transition-fast);
  flex-shrink: 0;
}

.sidebar-toggle-btn:hover {
  background: var(--color-bg-hover);
  color: var(--color-accent);
}

.sidebar-toggle-expand {
  margin-top: auto;
}
</style>