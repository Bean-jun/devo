import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { useBackgroundStore, type BackgroundProcess } from '@/stores/background'
import { useSessionStore } from '@/stores/session'
import { formatRelativeTime } from '@/utils/formatters'
import AppIcon from '@/components/common/AppIcon.vue'

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

export function useBackgroundPanel() {
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

  return {
    expandedPids,
    stoppingPids,
    error,
    processes,
    runningCount,
    hasProcesses,
    toggleExpand,
    isExpanded,
    isStopping,
    statusLabel,
    statusColor,
    setStdoutRef,
    handleStop,
    handleDismiss,
    refresh,
    formatRelativeTime,
  }
}