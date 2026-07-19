import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import { API_BASE } from '@/utils/constants'

export type BackgroundProcessStatus = 'running' | 'stopped' | 'failed'

export interface BackgroundProcess {
  pid: number
  cmd: string
  sessionID: string
  startedAt: Date
  status: BackgroundProcessStatus
  stdout: string
  stderr: string
  stoppedAt?: Date
  stopError?: string
}

const MAX_OUTPUT_BYTES = 256 * 1024

function trimOutput(s: string): string {
  if (s.length <= MAX_OUTPUT_BYTES) return s
  return s.slice(s.length - MAX_OUTPUT_BYTES)
}

export const useBackgroundStore = defineStore('background', () => {
  const processes = ref<Map<number, BackgroundProcess>>(new Map())

  const list = computed(() =>
    Array.from(processes.value.values()).sort((a, b) => b.startedAt.getTime() - a.startedAt.getTime()),
  )

  const runningCount = computed(() =>
    Array.from(processes.value.values()).filter((p) => p.status === 'running').length,
  )

  function register(pid: number, cmd: string, sessionID: string): void {
    const existing = processes.value.get(pid)
    if (existing) {
      if (existing.status === 'running') return
      existing.status = 'running'
      existing.stoppedAt = undefined
      existing.stopError = undefined
      processes.value.set(pid, existing)
      return
    }
    processes.value.set(pid, {
      pid,
      cmd,
      sessionID,
      startedAt: new Date(),
      status: 'running',
      stdout: '',
      stderr: '',
    })
  }

  function appendOutput(pid: number, stream: 'stdout' | 'stderr', data: string): void {
    const p = processes.value.get(pid)
    if (!p) return
    if (stream === 'stderr') {
      p.stderr = trimOutput(p.stderr + data)
    } else {
      p.stdout = trimOutput(p.stdout + data)
    }
    processes.value.set(pid, p)
  }

  function markStopped(pid: number, err?: string): void {
    const p = processes.value.get(pid)
    if (!p) return
    p.status = err ? 'failed' : 'stopped'
    p.stoppedAt = new Date()
    p.stopError = err
    processes.value.set(pid, p)
  }

  function removeProcess(pid: number): void {
    processes.value.delete(pid)
  }

  function clear(): void {
    processes.value.clear()
  }

  function clearSession(sessionID: string): void {
    for (const [pid, p] of processes.value) {
      if (p.sessionID === sessionID) processes.value.delete(pid)
    }
  }

  async function fetchProcesses(sessionID: string): Promise<void> {
    try {
      const res = await fetch(`${API_BASE}/sessions/${sessionID}/background`)
      if (!res.ok) return
      const data = await res.json()
      const procs = Array.isArray(data.processes) ? data.processes : []
      const next = new Map<number, BackgroundProcess>()
      for (const p of procs) {
        const pid = Number(p.pid)
        if (!Number.isFinite(pid)) continue
        const existing = processes.value.get(pid)
        next.set(pid, {
          pid,
          cmd: p.cmd || '',
          sessionID: p.session_id || sessionID,
          startedAt: p.started_at ? new Date(p.started_at) : new Date(),
          status: existing?.status ?? 'running',
          stdout: existing?.stdout ?? '',
          stderr: existing?.stderr ?? '',
          stoppedAt: existing?.stoppedAt,
          stopError: existing?.stopError,
        })
      }
      // Preserve processes the server no longer reports. A running process
      // disappearing means it has exited - mark it stopped. Already-stopped
      // or failed processes are kept verbatim so the user can still see the
      // output they captured before the process went away.
      for (const [pid, p] of processes.value) {
        if (p.sessionID === sessionID && !next.has(pid)) {
          next.set(pid, p.status === 'running' ? { ...p, status: 'stopped', stoppedAt: new Date() } : p)
        }
      }
      processes.value = next
    } catch {
      // Silently ignore - the panel will retry on next refresh.
    }
  }

  async function stopProcess(sessionID: string, pid: number): Promise<void> {
    const res = await fetch(`${API_BASE}/sessions/${sessionID}/background/${pid}/stop`, {
      method: 'POST',
    })
    if (!res.ok) {
      const data = await res.json().catch(() => ({}))
      throw new Error(data.message || `停止失败 (${res.status})`)
    }
    markStopped(pid)
  }

  return {
    processes,
    list,
    runningCount,
    register,
    appendOutput,
    markStopped,
    removeProcess,
    clear,
    clearSession,
    fetchProcesses,
    stopProcess,
  }
})
