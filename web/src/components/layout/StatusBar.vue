<script setup lang="ts">
import { computed } from 'vue'
import { useSessionStore } from '@/stores/session'
import { useUiStore } from '@/stores/ui'
import { STATUS_LABELS, STATUS_COLORS } from '@/utils/constants'
import { formatTokenCount } from '@/utils/formatters'

const sessionStore = useSessionStore()
const uiStore = useUiStore()

const sessionName = computed(() => sessionStore.currentSession?.title ?? '未连接')
const statusLabel = computed(() => STATUS_LABELS[sessionStore.sessionStatus] ?? '空闲')
const statusColor = computed(() => STATUS_COLORS[sessionStore.sessionStatus] ?? '#34c759')
const isProcessing = computed(() => sessionStore.isProcessing)
const workingDir = computed(() => sessionStore.currentSession?.workingDirectory ?? '')

const tokenUsage = computed(() => {
  const usage = sessionStore.currentSession?.tokenUsage
  const input = usage?.input ?? 0
  const output = usage?.output ?? 0
  const total = input + output
  return `${formatTokenCount(total)} (↑${formatTokenCount(input)} ↓${formatTokenCount(output)})`
})

const contextUsage = computed(() => {
  const usage = sessionStore.currentSession?.tokenUsage
  const input = usage?.input ?? 0
  return formatTokenCount(input)
})

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
</script>

<template>
  <header class="statusbar">
    <div class="statusbar-left">
      <span class="session-name" :title="sessionName">{{ sessionName }}</span>
      <span
        class="status-indicator"
        :class="{ processing: isProcessing }"
        :style="{ color: statusColor }"
      >
        <span class="status-dot" :style="{ background: statusColor }"></span>
        {{ statusLabel }}
      </span>
      <span v-if="workingDir" class="working-dir" :title="workingDir">
        {{ workingDir }}
      </span>
    </div>
    <div class="statusbar-right">
      <span class="context-usage" title="context">context: {{ contextUsage }}</span>
      <span class="token-usage" title="输入/输出 Token">Tokens: {{ tokenUsage }}</span>
      <span class="connection-status" :title="connectionStatusText">
        {{ connectionDot }} {{ connectionStatusText }}
      </span>
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
}
</style>