import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useBackgroundStore } from '@/stores/background'

describe('BackgroundStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.stubGlobal('fetch', vi.fn())
  })

  describe('register', () => {
    it('should add a new process with running status', () => {
      const store = useBackgroundStore()

      store.register(123, 'import time; time.sleep(60)', 'sess-1')

      expect(store.processes.has(123)).toBe(true)
      const p = store.processes.get(123)!
      expect(p.pid).toBe(123)
      expect(p.cmd).toBe('import time; time.sleep(60)')
      expect(p.sessionID).toBe('sess-1')
      expect(p.status).toBe('running')
      expect(p.stdout).toBe('')
      expect(p.stderr).toBe('')
    })

    it('should not overwrite an already-running process on re-register', () => {
      const store = useBackgroundStore()

      store.register(123, 'cmd-a', 'sess-1')
      store.appendOutput(123, 'stdout', 'hello\n')

      store.register(123, 'cmd-b', 'sess-1')

      const p = store.processes.get(123)!
      expect(p.cmd).toBe('cmd-a')
      expect(p.stdout).toBe('hello\n')
      expect(p.status).toBe('running')
    })

    it('should revive a stopped process back to running', () => {
      const store = useBackgroundStore()

      store.register(123, 'cmd-a', 'sess-1')
      store.markStopped(123)
      expect(store.processes.get(123)!.status).toBe('stopped')

      store.register(123, 'cmd-b', 'sess-1')
      const p = store.processes.get(123)!
      expect(p.status).toBe('running')
      expect(p.stoppedAt).toBeUndefined()
    })
  })

  describe('appendOutput', () => {
    it('should append to stdout buffer', () => {
      const store = useBackgroundStore()
      store.register(123, 'cmd', 'sess-1')

      store.appendOutput(123, 'stdout', 'line 1\n')
      store.appendOutput(123, 'stdout', 'line 2\n')

      expect(store.processes.get(123)!.stdout).toBe('line 1\nline 2\n')
      expect(store.processes.get(123)!.stderr).toBe('')
    })

    it('should append to stderr buffer when stream is stderr', () => {
      const store = useBackgroundStore()
      store.register(123, 'cmd', 'sess-1')

      store.appendOutput(123, 'stderr', 'error 1\n')

      expect(store.processes.get(123)!.stderr).toBe('error 1\n')
      expect(store.processes.get(123)!.stdout).toBe('')
    })

    it('should be a no-op for an unregistered PID', () => {
      const store = useBackgroundStore()

      store.appendOutput(999, 'stdout', 'data')

      expect(store.processes.has(999)).toBe(false)
    })

    it('should trim the buffer when it exceeds the cap', () => {
      const store = useBackgroundStore()
      store.register(123, 'cmd', 'sess-1')

      const chunk = 'x'.repeat(100 * 1024)
      for (let i = 0; i < 5; i++) {
        store.appendOutput(123, 'stdout', chunk)
      }

      const stdout = store.processes.get(123)!.stdout
      expect(stdout.length).toBeLessThanOrEqual(256 * 1024)
      // Most recent data must be retained.
      expect(stdout.endsWith('x')).toBe(true)
    })
  })

  describe('markStopped', () => {
    it('should set status to stopped without an error', () => {
      const store = useBackgroundStore()
      store.register(123, 'cmd', 'sess-1')

      store.markStopped(123)

      const p = store.processes.get(123)!
      expect(p.status).toBe('stopped')
      expect(p.stoppedAt).toBeInstanceOf(Date)
      expect(p.stopError).toBeUndefined()
    })

    it('should set status to failed with an error', () => {
      const store = useBackgroundStore()
      store.register(123, 'cmd', 'sess-1')

      store.markStopped(123, 'killed by user')

      const p = store.processes.get(123)!
      expect(p.status).toBe('failed')
      expect(p.stopError).toBe('killed by user')
    })

    it('should be a no-op for an unregistered PID', () => {
      const store = useBackgroundStore()
      expect(() => store.markStopped(999)).not.toThrow()
    })
  })

  describe('list computed', () => {
    it('should sort processes by startedAt descending', () => {
      const store = useBackgroundStore()
      const now = Date.now()
      vi.useFakeTimers()
      vi.setSystemTime(now)
      store.register(100, 'cmd-100', 'sess-1')
      vi.setSystemTime(now + 5000)
      store.register(200, 'cmd-200', 'sess-1')
      vi.setSystemTime(now + 10000)
      store.register(300, 'cmd-300', 'sess-1')
      vi.useRealTimers()

      const list = store.list
      expect(list.map((p) => p.pid)).toEqual([300, 200, 100])
    })

    it('should reflect running count', () => {
      const store = useBackgroundStore()
      store.register(100, 'cmd', 'sess-1')
      store.register(200, 'cmd', 'sess-1')
      store.markStopped(200)

      expect(store.runningCount).toBe(1)
    })
  })

  describe('clear / clearSession', () => {
    it('clear should remove all processes', () => {
      const store = useBackgroundStore()
      store.register(1, 'cmd', 'sess-1')
      store.register(2, 'cmd', 'sess-2')

      store.clear()

      expect(store.processes.size).toBe(0)
    })

    it('clearSession should remove only processes for that session', () => {
      const store = useBackgroundStore()
      store.register(1, 'cmd', 'sess-1')
      store.register(2, 'cmd', 'sess-2')
      store.register(3, 'cmd', 'sess-1')

      store.clearSession('sess-1')

      expect(store.processes.size).toBe(1)
      expect(store.processes.has(2)).toBe(true)
    })
  })

  describe('fetchProcesses', () => {
    it('should sync processes from the server response', async () => {
      const store = useBackgroundStore()
      store.register(100, 'old-cmd', 'sess-1')
      store.appendOutput(100, 'stdout', 'existing output')

      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        json: () =>
          Promise.resolve({
            processes: [
              { pid: 100, cmd: 'new-cmd', session_id: 'sess-1', started_at: '2026-01-01T00:00:00Z' },
              { pid: 200, cmd: 'fresh', session_id: 'sess-1', started_at: '2026-01-01T00:01:00Z' },
            ],
          }),
      } as Response)

      await store.fetchProcesses('sess-1')

      expect(store.processes.size).toBe(2)
      // Existing process keeps its accumulated output but updates cmd from server.
      const p100 = store.processes.get(100)!
      expect(p100.cmd).toBe('new-cmd')
      expect(p100.stdout).toBe('existing output')
      // New process is added.
      const p200 = store.processes.get(200)!
      expect(p200.cmd).toBe('fresh')
      expect(p200.status).toBe('running')
    })

    it('should mark running processes missing from server response as stopped', async () => {
      const store = useBackgroundStore()
      store.register(100, 'cmd', 'sess-1')

      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ processes: [] }),
      } as Response)

      await store.fetchProcesses('sess-1')

      const p = store.processes.get(100)!
      expect(p.status).toBe('stopped')
      expect(p.stoppedAt).toBeInstanceOf(Date)
    })

    it('should preserve stopped state for processes missing from server response', async () => {
      const store = useBackgroundStore()
      store.register(100, 'cmd', 'sess-1')
      store.markStopped(100)
      const originalStoppedAt = store.processes.get(100)!.stoppedAt

      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ processes: [] }),
      } as Response)

      await store.fetchProcesses('sess-1')

      const p = store.processes.get(100)!
      expect(p.status).toBe('stopped')
      expect(p.stoppedAt).toBe(originalStoppedAt)
    })

    it('should not throw on fetch failure', async () => {
      const store = useBackgroundStore()
      vi.mocked(fetch).mockRejectedValueOnce(new Error('network'))

      await expect(store.fetchProcesses('sess-1')).resolves.toBeUndefined()
    })

    it('should not throw on non-ok response', async () => {
      const store = useBackgroundStore()
      vi.mocked(fetch).mockResolvedValueOnce({ ok: false, status: 404 } as Response)

      await expect(store.fetchProcesses('sess-1')).resolves.toBeUndefined()
    })
  })

  describe('stopProcess', () => {
    it('should POST to the stop endpoint and mark the process stopped', async () => {
      const store = useBackgroundStore()
      store.register(123, 'cmd', 'sess-1')

      vi.mocked(fetch).mockResolvedValueOnce({ ok: true } as Response)

      await store.stopProcess('sess-1', 123)

      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('/sessions/sess-1/background/123/stop'),
        expect.objectContaining({ method: 'POST' }),
      )
      expect(store.processes.get(123)!.status).toBe('stopped')
    })

    it('should throw with server error message on failure', async () => {
      const store = useBackgroundStore()
      store.register(123, 'cmd', 'sess-1')

      vi.mocked(fetch).mockResolvedValueOnce({
        ok: false,
        status: 500,
        json: () => Promise.resolve({ message: 'kill failed' }),
      } as Response)

      await expect(store.stopProcess('sess-1', 123)).rejects.toThrow('kill failed')
    })

    it('should throw a generic message when response body is missing', async () => {
      const store = useBackgroundStore()
      store.register(123, 'cmd', 'sess-1')

      vi.mocked(fetch).mockResolvedValueOnce({
        ok: false,
        status: 500,
        json: () => Promise.reject(new Error('bad json')),
      } as Response)

      await expect(store.stopProcess('sess-1', 123)).rejects.toThrow(/停止失败/)
    })
  })
})
