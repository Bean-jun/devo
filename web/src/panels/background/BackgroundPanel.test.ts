import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import BackgroundPanel from '@/panels/background/BackgroundPanel.vue'
import { useBackgroundStore, type BackgroundProcess } from '@/stores/background'
import { useSessionStore } from '@/stores/session'

function procResponse(pid: number, cmd: string, sessionID = 'sess-1') {
  return {
    ok: true,
    json: () =>
      Promise.resolve({
        processes: [
          { pid, cmd, session_id: sessionID, started_at: new Date().toISOString() },
        ],
      }),
  } as Response
}

function emptyResponse() {
  return { ok: true, json: () => Promise.resolve({ processes: [] }) } as Response
}

describe('BackgroundPanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.stubGlobal('fetch', vi.fn())
    vi.mocked(fetch).mockResolvedValue(emptyResponse())
  })

  function mountPanel() {
    return mount(BackgroundPanel, {
      global: {
        plugins: [],
      },
    })
  }

  it('shows empty state when no processes are registered', async () => {
    const wrapper = mountPanel()
    await flushPromises()

    expect(wrapper.find('[data-test="empty-state"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="process-list"]').exists()).toBe(false)
  })

  it('shows process card after registering a process', async () => {
    const store = useBackgroundStore()
    store.register(1234, 'python dev server', 'sess-1')

    // Make refresh return the registered process so it stays running.
    vi.mocked(fetch).mockResolvedValue(procResponse(1234, 'python dev server'))

    const wrapper = mountPanel()
    await flushPromises()

    expect(wrapper.find('[data-test="empty-state"]').exists()).toBe(false)
    expect(wrapper.find('[data-test="process-list"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('PID 1234')
    expect(wrapper.text()).toContain('python dev server')
    expect(wrapper.text()).toContain('运行中')
    expect(wrapper.find('[data-test="running-count"]').text()).toContain('1 运行中')
  })

  it('expands to show stdout/stderr when header clicked', async () => {
    const store = useBackgroundStore()
    store.register(1234, 'cmd', 'sess-1')
    store.appendOutput(1234, 'stdout', 'server started on :3000')
    store.appendOutput(1234, 'stderr', 'warning: low memory')

    vi.mocked(fetch).mockResolvedValue(procResponse(1234, 'cmd'))

    const wrapper = mountPanel()
    await flushPromises()

    expect(wrapper.find('[data-test="card-body"]').exists()).toBe(false)

    await wrapper.find('.card-header').trigger('click')

    expect(wrapper.find('[data-test="card-body"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="stdout-output"]').text()).toContain('server started on :3000')
    expect(wrapper.find('[data-test="stderr-output"]').text()).toContain('warning: low memory')
  })

  it('shows stop button for running processes', async () => {
    const store = useBackgroundStore()
    store.register(1234, 'cmd', 'sess-1')

    vi.mocked(fetch).mockResolvedValue(procResponse(1234, 'cmd'))

    const wrapper = mountPanel()
    await flushPromises()
    await wrapper.find('.card-header').trigger('click')

    const stopBtn = wrapper.find('[data-test="stop-btn"]')
    expect(stopBtn.exists()).toBe(true)
    expect(stopBtn.text()).toContain('停止')
  })

  it('shows dismiss button instead of stop for stopped processes', async () => {
    const store = useBackgroundStore()
    store.register(1234, 'cmd', 'sess-1')
    store.markStopped(1234)

    // Even when refresh returns the process, it's locally marked stopped.
    vi.mocked(fetch).mockResolvedValue(procResponse(1234, 'cmd'))

    const wrapper = mountPanel()
    await flushPromises()
    await wrapper.find('.card-header').trigger('click')

    expect(wrapper.find('[data-test="stop-btn"]').exists()).toBe(false)
    const dismissBtn = wrapper.find('[data-test="dismiss-btn"]')
    expect(dismissBtn.exists()).toBe(true)
    expect(dismissBtn.text()).toContain('清除')
  })

  it('calls stopProcess and disables button while in flight', async () => {
    const store = useBackgroundStore()
    const sessionStore = useSessionStore()
    sessionStore.currentSession = { id: 'sess-1', title: 't', state: 'idle' } as any
    store.register(1234, 'cmd', 'sess-1')

    let resolveStop!: () => void
    const stopPromise = new Promise<void>((r) => { resolveStop = r })
    vi.mocked(fetch).mockImplementation((url) => {
      if (typeof url === 'string' && url.includes('/stop')) {
        return stopPromise.then(() => ({ ok: true, json: () => Promise.resolve({}) } as Response))
      }
      return Promise.resolve(procResponse(1234, 'cmd'))
    })

    const wrapper = mountPanel()
    await flushPromises()
    await wrapper.find('.card-header').trigger('click')

    const stopBtn = wrapper.find('[data-test="stop-btn"]')
    await stopBtn.trigger('click')
    await flushPromises()

    expect(stopBtn.text()).toContain('停止中...')
    expect(stopBtn.attributes('disabled')).toBeDefined()

    resolveStop()
    await flushPromises()

    const p = store.processes.get(1234)
    expect(p).toBeTruthy()
    expect(p!.status).toBe('stopped')
  })

  it('shows error message when stop fails', async () => {
    const store = useBackgroundStore()
    const sessionStore = useSessionStore()
    sessionStore.currentSession = { id: 'sess-1', title: 't', state: 'idle' } as any
    store.register(1234, 'cmd', 'sess-1')

    vi.mocked(fetch).mockImplementation((url) => {
      if (typeof url === 'string' && url.includes('/stop')) {
        return Promise.resolve({
          ok: false,
          status: 500,
          json: () => Promise.resolve({ message: 'kill failed: permission denied' }),
        } as Response)
      }
      return Promise.resolve(procResponse(1234, 'cmd'))
    })

    const wrapper = mountPanel()
    await flushPromises()
    await wrapper.find('.card-header').trigger('click')
    await wrapper.find('[data-test="stop-btn"]').trigger('click')
    await flushPromises()

    expect(wrapper.find('[data-test="panel-error"]').text()).toContain('kill failed: permission denied')
  })

  it('removes process from list when dismiss clicked', async () => {
    const store = useBackgroundStore()
    store.register(1234, 'cmd', 'sess-1')
    store.markStopped(1234)

    vi.mocked(fetch).mockResolvedValue(procResponse(1234, 'cmd'))

    const wrapper = mountPanel()
    await flushPromises()
    await wrapper.find('.card-header').trigger('click')

    await wrapper.find('[data-test="dismiss-btn"]').trigger('click')

    expect(store.processes.has(1234)).toBe(false)
    expect(wrapper.find('[data-test="empty-state"]').exists()).toBe(true)
  })

  it('refreshes process list on mount via fetchProcesses', async () => {
    const sessionStore = useSessionStore()
    sessionStore.currentSession = { id: 'sess-1', title: 't', state: 'idle' } as any

    vi.mocked(fetch).mockResolvedValue({
      ok: true,
      json: () =>
        Promise.resolve({
          processes: [
            { pid: 555, cmd: 'fetched-from-server', session_id: 'sess-1', started_at: '2026-01-01T00:00:00Z' },
          ],
        }),
    } as Response)

    const wrapper = mountPanel()
    await flushPromises()

    expect(wrapper.text()).toContain('PID 555')
    expect(wrapper.text()).toContain('fetched-from-server')
  })
})

// Ensure unused type import doesn't fail strict lint.
type _Unused = BackgroundProcess
