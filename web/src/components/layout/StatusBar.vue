<script setup lang="ts">
import { computed, ref, nextTick } from 'vue'
import { useSessionStore } from '@/stores/session'
import { useUiStore } from '@/stores/ui'
import { STATUS_LABELS, STATUS_COLORS } from '@/utils/constants'

const sessionStore = useSessionStore()
const uiStore = useUiStore()

const isRenaming = ref(false)
const renameValue = ref('')
const renameInputRef = ref<HTMLInputElement>()
const yoloLoading = ref(false)

const sessionName = computed(() => sessionStore.currentSession?.title ?? '未连接')
const statusLabel = computed(() => STATUS_LABELS[sessionStore.sessionStatus] ?? '空闲')
const statusColor = computed(() => STATUS_COLORS[sessionStore.sessionStatus] ?? '#34c759')
const isProcessing = computed(() => sessionStore.isProcessing)
const workingDir = computed(() => sessionStore.currentSession?.workingDirectory ?? '')

const themeIcon = computed(() => uiStore.theme === 'dark' ? '☀️' : '🌙')
const themeLabel = computed(() => uiStore.theme === 'dark' ? '浅色' : '深色')

async function startRename() {
  if (!sessionStore.currentSession) return
  renameValue.value = sessionName.value
  isRenaming.value = true
  await nextTick()
  renameInputRef.value?.focus()
  renameInputRef.value?.select()
}

async function confirmRename() {
  if (!isRenaming.value) return
  const newName = renameValue.value.trim()
  isRenaming.value = false
  if (newName && newName !== sessionName.value && sessionStore.currentSession) {
    try {
      await sessionStore.renameSession(sessionStore.currentSession.id, newName)
      uiStore.showToast('success', '会话已重命名')
    } catch {
      uiStore.showToast('error', '重命名失败')
    }
  }
}

function cancelRename() {
  isRenaming.value = false
}

defineExpose({ startRename })

function toggleTheme(event: MouseEvent) {
  const rect = (event.currentTarget as HTMLElement).getBoundingClientRect()
  const x = rect.left + rect.width / 2
  const y = rect.top + rect.height / 2
  uiStore.toggleThemeWithTransition(x, y)
}

async function toggleYolo() {
  yoloLoading.value = true
  try {
    await sessionStore.toggleYolo()
    const label = sessionStore.yoloEnabled ? 'YOLO 模式已开启' : 'YOLO 模式已关闭'
    uiStore.showToast('success', label)
  } catch {
    uiStore.showToast('error', 'YOLO 切换失败')
  } finally {
    yoloLoading.value = false
  }
}

const connectionStatusText = computed(() => {
  switch (uiStore.connectionStatus) {
    case 'connected': return '已连接'
    case 'connecting': return '连接中'
    case 'disconnected': return '未连接'
  }
})

const connectionDot = computed(() => {
  switch (uiStore.connectionStatus) {
    case 'connected': return '🟢'
    case 'connecting': return '🟡'
    case 'disconnected': return '🔴'
  }
})

const serverPort = computed(() => window.location.port)
</script>

<template>
  <header v-if="sessionStore.currentSession" class="statusbar" :class="{ yolo: sessionStore.yoloEnabled }">
    <div class="statusbar-left">
      <input
        v-if="isRenaming"
        ref="renameInputRef"
        v-model="renameValue"
        class="session-name-input"
        @keydown.enter="confirmRename"
        @keydown.escape="cancelRename"
        @blur="confirmRename"
      />
      <span v-else class="session-name" :title="sessionName" @dblclick="startRename">{{ sessionName }}</span>
      <span
        class="status-indicator"
        :class="{ processing: isProcessing }"
        :style="{ color: statusColor }"
      >
        <span class="status-dot" :style="{ background: statusColor }"></span>
        {{ statusLabel }}
      </span>
      <button
        class="yolo-toggle"
        :class="{ active: sessionStore.yoloEnabled }"
        :title="sessionStore.yoloEnabled ? 'YOLO 模式已开启 - 点击关闭' : 'YOLO 模式 - 点击开启自动批准'"
        :disabled="yoloLoading"
        @click="toggleYolo"
      >
        <span class="yolo-icon">{{ yoloLoading ? '⏳' : '🔥' }}</span>
        <span class="yolo-label" :class="{ on: sessionStore.yoloEnabled }">YOLO</span>
      </button>
      <span v-if="workingDir" class="working-dir" :title="workingDir">
        {{ workingDir }}
      </span>
    </div>
    <div class="statusbar-right">
      <button class="theme-toggle" :title="themeLabel" @click="toggleTheme">
        {{ themeIcon }}
      </button>
      <span class="connection-status" :title="connectionStatusText">
        {{ connectionDot }} {{ connectionStatusText }}
      </span>
      <span class="port-info" title="后端端口">:{{ serverPort }}</span>
    </div>
  </header>
</template>

<style scoped>
.statusbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: var(--statusbar-height);
  padding: 0 var(--space-lg);
  background: var(--color-bg-secondary);
  border-bottom: 1px solid var(--color-border-light);
  flex-shrink: 0;
  user-select: none;
  -webkit-app-region: drag;
}

.statusbar-left,
.statusbar-right {
  display: flex;
  align-items: center;
  gap: var(--space-md);
  -webkit-app-region: no-drag;
}

.session-name {
  font-size: var(--font-size-sm);
  font-weight: 600;
  color: var(--color-text-primary);
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  cursor: pointer;
}

.session-name-input {
  font-size: var(--font-size-sm);
  font-weight: 600;
  color: var(--color-text-primary);
  background: var(--color-bg-primary);
  border: 1px solid var(--color-accent);
  border-radius: var(--radius-sm);
  padding: 2px 6px;
  width: 180px;
  outline: none;
  font-family: inherit;
}

.status-indicator {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: var(--font-size-xs);
  font-weight: 500;
}

.status-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  display: inline-block;
}

.status-indicator.processing .status-dot {
  animation: pulse 1.5s ease-in-out infinite;
}

.token-usage {
  font-size: var(--font-size-xs);
  color: var(--color-text-secondary);
  font-family: var(--font-mono);
}

.context-usage {
  font-size: var(--font-size-xs);
  color: var(--color-text-secondary);
  font-family: var(--font-mono);
}

.working-dir {
  font-size: var(--font-size-xs);
  color: var(--color-text-tertiary);
  max-width: 250px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.connection-status {
  font-size: var(--font-size-xs);
  color: var(--color-text-secondary);
  cursor: pointer;
  white-space: nowrap;
  display: flex;
  align-items: center;
  gap: 4px;
  line-height: 1;
}

.port-info {
  font-size: var(--font-size-xs);
  color: var(--color-text-tertiary);
  font-family: var(--font-mono);
  white-space: nowrap;
  line-height: 1;
}

.theme-toggle {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border-radius: var(--radius-full);
  border: 1px solid var(--color-border-light);
  background: transparent;
  cursor: pointer;
  font-size: 14px;
  line-height: 1;
  padding: 0;
  transition: all var(--transition-fast);
}

.theme-toggle:hover {
  background: var(--color-bg-hover);
  border-color: var(--color-border);
}

.statusbar.yolo {
  background: linear-gradient(90deg, rgba(255, 149, 0, 0.12), rgba(255, 149, 0, 0.04) 40%, var(--color-bg-secondary) 80%);
}

.yolo-toggle {
  display: flex;
  align-items: center;
  gap: 4px;
  height: 30px;
  padding: 0 10px;
  border-radius: 6px;
  border: 1px solid var(--color-border);
  background: var(--color-bg-tertiary);
  cursor: pointer;
  font-size: 13px;
  line-height: 1;
  transition: all var(--transition-fast);
  opacity: 0.7;
}

.yolo-toggle:hover {
  opacity: 1;
  border-color: var(--color-text-tertiary);
}

.yolo-toggle.active {
  opacity: 1;
  border-color: #ff9500;
  background: rgba(255, 149, 0, 0.15);
  animation: yolo-pulse 2s ease-in-out infinite;
}

.yolo-toggle.active:hover {
  border-color: #ff9500;
  background: rgba(255, 149, 0, 0.22);
}

.yolo-toggle:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}

.yolo-icon {
  font-size: 15px;
  line-height: 1;
}

.yolo-label {
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.5px;
  color: var(--color-text-tertiary);
  transition: color var(--transition-fast);
}

.yolo-label.on {
  color: #ff9500;
}

@keyframes yolo-pulse {
  0%, 100% { box-shadow: 0 0 0 0 rgba(255, 149, 0, 0.4); }
  50% { box-shadow: 0 0 0 4px rgba(255, 149, 0, 0); }
}
</style>