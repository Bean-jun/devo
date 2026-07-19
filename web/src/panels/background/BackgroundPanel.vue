<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { useBackgroundStore, type BackgroundProcess } from '@/stores/background'
import { useSessionStore } from '@/stores/session'
import { formatRelativeTime } from '@/utils/formatters'
import AppIcon from '@/components/common/AppIcon.vue'

const backgroundStore = useBackgroundStore()
const sessionStore = useSessionStore()

const expandedPids = ref<Set<number>>(new Set())
const stoppingPids = ref<Set<number>>(new Set())
const error = ref<string | null>(null)
const stdoutRefs = ref<Map<number, HTMLElement>>(new Map())

let refreshTimer: ReturnType<typeof setInterval> | null = null

const processes = computed(() => backgroundStore.list)
const runningCount = computed(() => backgroundStore.runningCount)
const hasProcesses = computed(() => processes.value.length > 0)

function toggleExpand(pid: number): void {
  if (expandedPids.value.has(pid)) {
    expandedPids.value.delete(pid)
  } else {
    expandedPids.value.add(pid)
  }
}

function isExpanded(pid: number): boolean {
  return expandedPids.value.has(pid)
}

function isStopping(pid: number): boolean {
  return stoppingPids.value.has(pid)
}

function statusLabel(p: BackgroundProcess): string {
  switch (p.status) {
    case 'running':
      return '运行中'
    case 'stopped':
      return '已停止'
    case 'failed':
      return '失败'
  }
}

function statusColor(p: BackgroundProcess): string {
  switch (p.status) {
    case 'running':
      return 'var(--color-success)'
    case 'stopped':
      return 'var(--color-text-tertiary)'
    case 'failed':
      return 'var(--color-error)'
  }
}

function setStdoutRef(pid: number, el: HTMLElement | null): void {
  if (el) {
    stdoutRefs.value.set(pid, el)
    if (isExpanded(pid)) {
      nextTick(() => {
        el.scrollTop = el.scrollHeight
      })
    }
  } else {
    stdoutRefs.value.delete(pid)
  }
}

async function handleStop(p: BackgroundProcess): Promise<void> {
  const sessionID = sessionStore.currentSession?.id || p.sessionID
  if (!sessionID) {
    error.value = '找不到当前会话'
    return
  }
  stoppingPids.value.add(p.pid)
  error.value = null
  try {
    await backgroundStore.stopProcess(sessionID, p.pid)
    expandedPids.value.delete(p.pid)
  } catch (e: any) {
    error.value = e.message || '停止失败'
  } finally {
    stoppingPids.value.delete(p.pid)
  }
}

function handleDismiss(pid: number): void {
  backgroundStore.removeProcess(pid)
  expandedPids.value.delete(pid)
}

async function refresh(): Promise<void> {
  const sessionID = sessionStore.currentSession?.id
  if (!sessionID) return
  await backgroundStore.fetchProcesses(sessionID)
}

watch(
  () => processes.value.map((p) => p.stdout + p.stderr).join('|'),
  () => {
    for (const pid of expandedPids.value) {
      const el = stdoutRefs.value.get(pid)
      if (el) {
        el.scrollTop = el.scrollHeight
      }
    }
  },
)

watch(
  () => sessionStore.currentSession?.id,
  (newId, oldId) => {
    if (oldId) {
      backgroundStore.clearSession(oldId)
    }
    if (newId) {
      refresh()
    }
  },
)

onMounted(() => {
  refresh()
  refreshTimer = setInterval(refresh, 5000)
})

onUnmounted(() => {
  if (refreshTimer) {
    clearInterval(refreshTimer)
    refreshTimer = null
  }
})
</script>

<template>
  <div class="background-panel" data-test="background-panel">
    <div class="panel-toolbar">
      <div class="panel-title">
        后台进程
        <span v-if="runningCount > 0" class="running-badge" data-test="running-count">
          {{ runningCount }} 运行中
        </span>
      </div>
      <button class="icon-btn" title="刷新" @click="refresh">
        <AppIcon name="arrow-clockwise" :size="14" />
      </button>
    </div>

    <div v-if="error" class="panel-error" data-test="panel-error">{{ error }}</div>

    <div v-if="!hasProcesses" class="empty-state" data-test="empty-state">
      <AppIcon name="hourglass" :size="32" class="empty-icon" />
      <p>当前没有后台进程</p>
      <p class="hint">exec_python 的 background 模式启动的进程会显示在这里</p>
    </div>

    <div v-else class="process-list" data-test="process-list">
      <div
        v-for="p in processes"
        :key="p.pid"
        class="process-card"
        :class="`status-${p.status}`"
      >
        <div class="card-header" @click="toggleExpand(p.pid)">
          <div class="card-left">
            <AppIcon
              :name="p.status === 'running' ? 'arrow-clockwise' : 'stop'"
              :size="14"
              :color="statusColor(p)"
              class="status-icon"
            />
            <span class="pid">PID {{ p.pid }}</span>
            <span class="status-tag" :style="{ color: statusColor(p) }">
              {{ statusLabel(p) }}
            </span>
          </div>
          <div class="card-right">
            <span class="started-at">{{ formatRelativeTime(p.startedAt.toISOString()) }}</span>
            <AppIcon
              :name="isExpanded(p.pid) ? 'caret-down' : 'caret-right'"
              :size="12"
              class="chevron"
            />
          </div>
        </div>

        <div class="card-meta">
          <span class="cmd" :title="p.cmd">{{ p.cmd || '(无命令)' }}</span>
        </div>

        <div v-if="isExpanded(p.pid)" class="card-body" data-test="card-body">
          <div class="output-section">
            <div class="output-label">stdout</div>
            <pre
              :ref="(el) => setStdoutRef(p.pid, el as HTMLElement | null)"
              class="output-content"
              data-test="stdout-output"
            >{{ p.stdout || '(无输出)' }}</pre>
            <div v-if="p.stderr" class="output-label stderr-label">stderr</div>
            <pre
              v-if="p.stderr"
              class="output-content stderr"
              data-test="stderr-output"
            >{{ p.stderr }}</pre>
          </div>

          <div class="card-actions">
            <button
              v-if="p.status === 'running'"
              class="action-btn danger"
              :disabled="isStopping(p.pid)"
              @click="handleStop(p)"
              data-test="stop-btn"
            >
              <AppIcon name="stop" :size="12" />
              {{ isStopping(p.pid) ? '停止中...' : '停止' }}
            </button>
            <button
              v-else
              class="action-btn"
              @click="handleDismiss(p.pid)"
              data-test="dismiss-btn"
            >
              <AppIcon name="x" :size="12" />
              清除
            </button>
          </div>

          <div v-if="p.stopError" class="stop-error" data-test="stop-error">
            {{ p.stopError }}
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.background-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  padding: 8px;
  gap: 8px;
  overflow: hidden;
}

.panel-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 4px 4px 8px;
  border-bottom: 1px solid var(--color-border);
}

.panel-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-primary);
  display: flex;
  align-items: center;
  gap: 8px;
}

.running-badge {
  font-size: 11px;
  font-weight: 500;
  padding: 1px 8px;
  border-radius: var(--radius-full);
  background: var(--color-success);
  color: white;
}

.icon-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.icon-btn:hover {
  background: var(--color-bg-hover);
  color: var(--color-text-primary);
}

.panel-error {
  font-size: 12px;
  color: var(--color-error);
  padding: 6px 8px;
  background: rgba(255, 0, 0, 0.05);
  border-radius: var(--radius-sm);
}

.empty-state {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--color-text-tertiary);
  text-align: center;
  padding: 32px 16px;
}

.empty-icon {
  opacity: 0.4;
}

.empty-state p {
  margin: 0;
  font-size: 13px;
}

.empty-state .hint {
  font-size: 11px;
  color: var(--color-text-tertiary);
}

.process-list {
  flex: 1;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.process-card {
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-bg-primary);
  overflow: hidden;
}

.process-card.status-running {
  border-left: 3px solid var(--color-success);
}

.process-card.status-stopped {
  border-left: 3px solid var(--color-text-tertiary);
  opacity: 0.85;
}

.process-card.status-failed {
  border-left: 3px solid var(--color-error);
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 8px;
  cursor: pointer;
  user-select: none;
}

.card-header:hover {
  background: var(--color-bg-hover);
}

.card-left {
  display: flex;
  align-items: center;
  gap: 6px;
}

.status-icon {
  flex-shrink: 0;
}

.pid {
  font-family: var(--font-mono);
  font-size: 12px;
  font-weight: 600;
  color: var(--color-text-primary);
}

.status-tag {
  font-size: 10px;
  font-weight: 500;
  padding: 1px 6px;
  border-radius: var(--radius-full);
  background: var(--color-bg-tertiary);
}

.card-right {
  display: flex;
  align-items: center;
  gap: 6px;
}

.started-at {
  font-size: 11px;
  color: var(--color-text-tertiary);
}

.chevron {
  color: var(--color-text-tertiary);
}

.card-meta {
  padding: 0 8px 6px;
}

.cmd {
  display: block;
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--color-text-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.card-body {
  border-top: 1px solid var(--color-border-light);
  padding: 8px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.output-section {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.output-label {
  font-size: 10px;
  font-weight: 600;
  color: var(--color-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.stderr-label {
  color: var(--color-error);
  margin-top: 4px;
}

.output-content {
  font-family: var(--font-mono);
  font-size: 11px;
  background: var(--color-bg-secondary);
  padding: 6px 8px;
  border-radius: var(--radius-sm);
  max-height: 200px;
  overflow-y: auto;
  white-space: pre-wrap;
  word-break: break-all;
  color: var(--color-text-primary);
  line-height: 1.4;
  margin: 0;
}

.output-content.stderr {
  color: var(--color-error);
}

.card-actions {
  display: flex;
  gap: 6px;
  margin-top: 4px;
}

.action-btn {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text-primary);
  cursor: pointer;
  font-size: 11px;
  transition: all var(--transition-fast);
}

.action-btn:hover:not(:disabled) {
  background: var(--color-bg-hover);
}

.action-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.action-btn.danger {
  color: var(--color-error);
  border-color: var(--color-error);
}

.action-btn.danger:hover:not(:disabled) {
  background: var(--color-error);
  color: white;
}

.stop-error {
  font-size: 11px;
  color: var(--color-error);
  padding: 4px 6px;
  background: rgba(255, 0, 0, 0.05);
  border-radius: var(--radius-sm);
}
</style>
