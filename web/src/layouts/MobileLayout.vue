<script setup lang="ts">
import { ref, computed, watch, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { useSessionStore } from '@/stores/session'
import { useChatStore } from '@/stores/chat'
import { useUiStore } from '@/stores/ui'
import { useApprovalStore } from '@/stores/approval'
import { API_BASE } from '@/utils/constants'
import { formatTokenCount } from '@/utils/formatters'
import { useFps } from '@/composables/useFps'
import type { Command } from '@/stores/command'
import StatusBar from '@/components/layout/StatusBar.vue'
import ChatPanel from '@/components/chat/ChatPanel.vue'
import GlobalModals from '@/components/layout/GlobalModals.vue'
import MobileInputBar from '@/components/mobile/MobileInputBar.vue'
import MobileCommandSheet from '@/components/mobile/MobileCommandSheet.vue'
import MobilePanelDrawer from '@/components/mobile/MobilePanelDrawer.vue'
import MobileWorkspacePicker from '@/components/mobile/MobileWorkspacePicker.vue'
import MobileSessionPicker from '@/components/mobile/MobileSessionPicker.vue'

const router = useRouter()
const sessionStore = useSessionStore()
const chatStore = useChatStore()
const uiStore = useUiStore()
const approvalStore = useApprovalStore()

const showCommandSheet = ref(false)
const showPanelDrawer = ref(false)
const showWorkspacePicker = ref(false)
const showSessionPicker = ref(false)

const showNewSessionDialog = ref(false)
const newSessionTitle = ref('')

const showInfoDialog = ref(false)
const infoDialogTitle = ref('')
const infoDialogContent = ref('')

const isProcessing = computed(() => sessionStore.isProcessing)

const appVersion = import.meta.env.VITE_APP_VERSION || 'dev'
const { fps } = useFps()

function openCommandSheet() {
  showCommandSheet.value = true
}

function closeCommandSheet() {
  showCommandSheet.value = false
}

function openPanelDrawer() {
  showPanelDrawer.value = true
}

function closePanelDrawer() {
  showPanelDrawer.value = false
}

function openWorkspacePicker() {
  showWorkspacePicker.value = true
}

function closeWorkspacePicker() {
  showWorkspacePicker.value = false
}

function openSessionPicker() {
  showSessionPicker.value = true
}

function closeSessionPicker() {
  showSessionPicker.value = false
}

function showInfo(title: string, content: string) {
  infoDialogTitle.value = title
  infoDialogContent.value = content
  showInfoDialog.value = true
}

function closeInfoDialog() {
  showInfoDialog.value = false
}

function handleCommandSelect(cmd: Command) {
  closeCommandSheet()

  switch (cmd.id) {
    case 'files':
    case 'skills':
    case 'mcp':
    case 'memory':
    case 'dashboard':
    case 'settings': {
      const tabMap: Record<string, string> = {
        files: 'files',
        skills: 'skills',
        mcp: 'mcp',
        memory: 'memory',
        dashboard: 'dashboard',
        settings: 'settings',
      }
      const tab = (tabMap[cmd.id] || 'files') as 'files' | 'skills' | 'mcp' | 'memory' | 'dashboard' | 'settings'
      uiStore.setActiveRightTab(tab)
      openPanelDrawer()
      break
    }
    case 'new':
      handleNewSession()
      break
    case 'switch':
      openSessionPicker()
      break
    case 'rename':
      handleRename()
      break
    case 'workspace-switch':
      openWorkspacePicker()
      break
    case 'toggle-theme':
      uiStore.toggleTheme()
      uiStore.showToast('success', '主题已切换')
      break
    case 'pause':
      handlePauseSession()
      break
    case 'resume':
      handleResumeSession()
      break
    case 'cancel':
      handleCancelSession()
      break
    case 'help':
      uiStore.setActiveModal('help')
      break
    case 'export':
      handleExport()
      break
    case 'rollback':
      uiStore.setActiveModal('rollback-picker')
      break
    case 'status':
      handleStatus()
      break
    case 'version':
      handleVersion()
      break
    default:
      break
  }
}

function handleSend(text: string) {
  if (!sessionStore.currentSession) return
  chatStore.appendUserMessage(text)

  fetch(`${API_BASE}/sessions/${sessionStore.currentSession.id}/messages`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ content: text }),
  }).catch((err) => {
    uiStore.showToast('error', `发送失败: ${(err as Error).message}`)
  })
}

async function handleStop() {
  if (!sessionStore.currentSession) return
  try {
    await fetch(`${API_BASE}/sessions/${sessionStore.currentSession.id}/cancel`, {
      method: 'POST',
    })
    sessionStore.updateSessionState(sessionStore.currentSession.id, 'idle')
    chatStore.isStreaming = false
  } catch {}
}

async function handleNewSession() {
  newSessionTitle.value = ''
  showNewSessionDialog.value = true
  await nextTick()
  const input = document.querySelector('.new-session-mobile-input') as HTMLInputElement
  input?.focus()
}

async function confirmNewSession() {
  showNewSessionDialog.value = false
  const dir = uiStore.activeWorkspace || sessionStore.workingDirectory
  await sessionStore.createSession({
    workingDirectory: dir,
    title: newSessionTitle.value.trim() || undefined,
  })
  router.push('/chat')
}

function cancelNewSession(e?: Event) {
  e?.stopPropagation()
  showNewSessionDialog.value = false
}

function handleRename() {
  const statusbarEl = document.querySelector('.statusbar')
  if (statusbarEl) {
    const renameEl = statusbarEl.querySelector('.session-name') as HTMLElement
    if (renameEl) {
      renameEl.dispatchEvent(new MouseEvent('dblclick', { bubbles: true }))
    }
  }
}

async function handlePauseSession() {
  const session = sessionStore.currentSession
  if (!session) return
  try {
    const res = await fetch(`${API_BASE}/sessions/${session.id}/pause`, { method: 'POST' })
    if (!res.ok) throw new Error('暂停失败')
    sessionStore.updateSessionState(session.id, 'paused')
    uiStore.showToast('info', '会话已暂停')
  } catch (e: any) {
    uiStore.showToast('error', e.message || '暂停失败')
  }
}

async function handleResumeSession() {
  const session = sessionStore.currentSession
  if (!session) return
  try {
    const res = await fetch(`${API_BASE}/sessions/${session.id}/resume`, { method: 'POST' })
    if (!res.ok) throw new Error('恢复失败')
    sessionStore.updateSessionState(session.id, 'idle')
    uiStore.showToast('success', '会话已恢复')
  } catch (e: any) {
    uiStore.showToast('error', e.message || '恢复失败')
  }
}

async function handleCancelSession() {
  const session = sessionStore.currentSession
  if (!session) return
  try {
    const res = await fetch(`${API_BASE}/sessions/${session.id}/cancel`, { method: 'POST' })
    if (!res.ok) throw new Error('取消失败')
    sessionStore.updateSessionState(session.id, 'cancelled')
    uiStore.showToast('info', '操作已取消')
  } catch (e: any) {
    uiStore.showToast('error', e.message || '取消失败')
  }
}

async function handleExport() {
  const session = sessionStore.currentSession
  if (!session) {
    uiStore.showToast('error', '没有当前会话')
    return
  }
  try {
    const sid = session.id
    await fetch(`${API_BASE}/sessions/${sid}/sync-archive`, { method: 'POST' })
    const url = `${API_BASE}/sessions/${sid}/archive`
    const filename = `${sid}.md`
    const a = document.createElement('a')
    a.href = url
    a.download = filename
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
  } catch {
    uiStore.showToast('error', '导出失败')
  }
}

function handleStatus() {
  const session = sessionStore.currentSession
  const contextTokens = formatTokenCount(session?.currentContextTokens ?? 0)
  const usage = session?.tokenUsage
  const inputTokens = formatTokenCount(usage?.input ?? 0)
  const outputTokens = formatTokenCount(usage?.output ?? 0)
  const totalTokens = formatTokenCount((usage?.input ?? 0) + (usage?.output ?? 0))
  const dir = session?.workingDirectory || '未设置'

  const lines = [
    `Context: ${contextTokens}`,
    `Tokens: ${totalTokens} (↑${inputTokens} ↓${outputTokens})`,
    `FPS: ${fps.value}`,
    `工作区: ${dir}`,
  ]
  showInfo('状态信息', lines.join('\n'))
}

function handleVersion() {
  showInfo('版本信息', `Devo v${appVersion}`)
}

async function handleAddWorkspace() {
  const path = prompt('请输入工作区路径：')
  if (!path) return
  try {
    const res = await fetch(`${API_BASE}/workspace`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ path }),
    })
    if (!res.ok) throw new Error('添加失败')
    await uiStore.fetchWorkspaceList()
    uiStore.showToast('success', '工作区已添加')
  } catch (e: any) {
    uiStore.showToast('error', e.message || '添加失败')
  }
}

function handleWorkspaceSwitched(_path: string) {
  uiStore.showToast('success', '已切换工作区')
}

function handleNewSessionFromPicker() {
  handleNewSession()
}
</script>

<template>
  <div class="mobile-layout" data-test="mobile-layout">
    <StatusBar />
    <div class="mobile-chat">
      <ChatPanel :hide-input="true" />
    </div>
    <MobileInputBar
      :is-disabled="false"
      :is-processing="isProcessing"
      @send="handleSend"
      @stop="handleStop"
      @open-command="openCommandSheet"
    />
    <GlobalModals />

    <MobileCommandSheet
      v-if="showCommandSheet"
      @select="handleCommandSelect"
      @close="closeCommandSheet"
    />

    <MobilePanelDrawer
      v-if="showPanelDrawer"
      @close="closePanelDrawer"
    />

    <MobileWorkspacePicker
      v-if="showWorkspacePicker"
      @close="closeWorkspacePicker"
      @switched="handleWorkspaceSwitched"
    />

    <MobileSessionPicker
      v-if="showSessionPicker"
      @close="closeSessionPicker"
      @new-session="handleNewSessionFromPicker"
    />

    <Teleport to="body">
      <div v-if="showNewSessionDialog" class="dialog-overlay" @click.self="cancelNewSession" data-test="new-session-dialog-overlay">
        <div class="dialog-sheet" @click.stop data-test="new-session-dialog">
          <div class="dialog-title">新建会话</div>
          <input
            v-model="newSessionTitle"
            class="new-session-mobile-input"
            type="text"
            placeholder="输入会话名称（可选）"
            @keydown.enter="confirmNewSession"
            @keydown.escape="cancelNewSession"
          />
          <div class="dialog-actions">
            <button class="dialog-btn-cancel" @click="cancelNewSession">取消</button>
            <button class="dialog-btn-confirm" @click="confirmNewSession">创建</button>
          </div>
        </div>
      </div>
    </Teleport>

    <Teleport to="body">
      <div v-if="showInfoDialog" class="dialog-overlay" @click.self="closeInfoDialog" data-test="info-dialog-overlay">
        <div class="dialog-sheet" @click.stop data-test="info-dialog">
          <div class="dialog-title" data-test="info-dialog-title">{{ infoDialogTitle }}</div>
          <pre class="info-dialog-content" data-test="info-dialog-content">{{ infoDialogContent }}</pre>
          <div class="dialog-actions">
            <button class="dialog-btn-confirm" @click="closeInfoDialog" data-test="info-dialog-confirm">确认</button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.mobile-layout {
  display: flex;
  flex-direction: column;
  height: 100dvh;
  width: 100vw;
  overflow: hidden;
  background: var(--color-bg-primary);
  padding-top: env(safe-area-inset-top, 0px);
  padding-left: env(safe-area-inset-left, 0px);
  padding-right: env(safe-area-inset-right, 0px);
}

.mobile-chat {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
}

.dialog-overlay {
  position: fixed;
  inset: 0;
  z-index: 5500;
  background: var(--color-overlay);
  animation: fadeIn var(--transition-fast) ease;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--space-xl);
}

.dialog-sheet {
  width: 100%;
  max-width: 360px;
  background: var(--color-bg-primary);
  border-radius: var(--radius-xl);
  padding: var(--space-xl);
  animation: modalIn var(--transition-base) ease;
}

.dialog-title {
  font-size: var(--font-size-lg);
  font-weight: 600;
  color: var(--color-text-primary);
  margin-bottom: var(--space-lg);
  text-align: center;
}

.new-session-mobile-input {
  width: 100%;
  padding: var(--space-md);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-bg-secondary);
  color: var(--color-text-primary);
  font-size: var(--font-size-base);
  outline: none;
  margin-bottom: var(--space-lg);
  min-height: 44px;
}

.new-session-mobile-input:focus {
  border-color: var(--color-accent);
}

.info-dialog-content {
  margin: 0 0 var(--space-md) 0;
  padding: var(--space-md);
  background: var(--color-bg-secondary);
  border-radius: var(--radius-sm);
  font-size: var(--font-size-base);
  font-family: var(--font-mono);
  color: var(--color-text-primary);
  line-height: 1.8;
  white-space: pre-wrap;
  word-break: break-all;
}

.dialog-actions {
  display: flex;
  gap: var(--space-md);
}

.dialog-btn-cancel,
.dialog-btn-confirm {
  flex: 1;
  padding: var(--space-md);
  border: none;
  border-radius: var(--radius-md);
  font-size: var(--font-size-base);
  cursor: pointer;
  min-height: 44px;
  transition: all var(--transition-fast);
}

.dialog-btn-cancel {
  background: var(--color-bg-secondary);
  color: var(--color-text-secondary);
}

.dialog-btn-cancel:active {
  background: var(--color-bg-tertiary);
}

.dialog-btn-confirm {
  background: var(--color-accent);
  color: white;
}

.dialog-btn-confirm:active {
  background: var(--color-accent-hover);
}

@keyframes modalIn {
  from {
    opacity: 0;
    transform: scale(0.95);
  }
  to {
    opacity: 1;
    transform: scale(1);
  }
}
</style>