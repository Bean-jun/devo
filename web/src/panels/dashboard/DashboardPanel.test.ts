import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import DashboardPanel from '@/panels/dashboard/DashboardPanel.vue'
import { useSessionStore } from '@/stores/session'

describe('DashboardPanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.stubGlobal('fetch', vi.fn())
  })

  function setupSession(id: string, workingDirectory: string) {
    const store = useSessionStore()
    store.currentSession = {
      id,
      title: 'Test Session',
      state: 'idle',
      workingDirectory,
      createdAt: '2026-01-01T00:00:00Z',
      lastActiveAt: '2026-01-01T00:00:00Z',
      messageCount: 0,
      tokenUsage: { input: 0, output: 0 },
      trustLevel: 'normal',
      approvalPolicy: {},
    }
  }

  function mockSessionUsageResponse(data: any) {
    vi.mocked(fetch).mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve(data),
    } as Response)
  }

  function mockProjectUsageResponse(data: any) {
    vi.mocked(fetch).mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve(data),
    } as Response)
  }

  it('should show empty state when no session', async () => {
    vi.mocked(fetch).mockResolvedValue({ ok: true, json: () => Promise.resolve({}) } as Response)
    vi.mocked(fetch).mockResolvedValue({ ok: true, json: () => Promise.resolve({}) } as Response)

    const wrapper = mount(DashboardPanel)

    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('暂无活跃会话')
  })

  it('should show session usage stats', async () => {
    setupSession('sess-1', '/tmp/test')

    mockSessionUsageResponse({
      total_input_tokens: 500,
      total_output_tokens: 200,
      total_tokens: 700,
      compression_count: 1,
    })
    mockProjectUsageResponse({
      summary: { input: 0, output: 0, total: 0 },
      groups: [],
    })

    const wrapper = mount(DashboardPanel)

    await vi.waitFor(() => {
      expect(wrapper.text()).toContain('500')
      expect(wrapper.text()).toContain('200')
      expect(wrapper.text()).toContain('700')
      expect(wrapper.text()).toContain('1')
    })
  })

  it('should show session steps as bar chart', async () => {
    setupSession('sess-2', '/tmp/test')

    mockSessionUsageResponse({
      total_input_tokens: 300,
      total_output_tokens: 150,
      total_tokens: 450,
      compression_count: 0,
      steps: [
        { step_seq: 1, input_tokens: 100, output_tokens: 50, created_at: '2026-01-01T00:00:00Z' },
        { step_seq: 2, input_tokens: 200, output_tokens: 100, created_at: '2026-01-01T00:00:00Z' },
      ],
    })
    mockProjectUsageResponse({
      summary: { input: 0, output: 0, total: 0 },
      groups: [],
    })

    const wrapper = mount(DashboardPanel)

    await vi.waitFor(() => {
      expect(wrapper.text()).toContain('步骤消耗')
      const barItems = wrapper.findAll('.bar-item')
      expect(barItems.length).toBe(2)
    })
  })

  it('should show project usage with date grouping', async () => {
    setupSession('sess-3', '/tmp/test')

    mockSessionUsageResponse({
      total_input_tokens: 0,
      total_output_tokens: 0,
      total_tokens: 0,
      compression_count: 0,
    })
    mockProjectUsageResponse({
      summary: { input: 1000, output: 500, total: 1500 },
      groups: [
        { key: '2026-01-01', input_tokens: 600, output_tokens: 300, total_tokens: 900 },
        { key: '2026-01-02', input_tokens: 400, output_tokens: 200, total_tokens: 600 },
      ],
    })

    const wrapper = mount(DashboardPanel)

    await vi.waitFor(() => {
      expect(wrapper.text()).toContain('1.5k')
      expect(wrapper.text()).toContain('2026-01-01')
      expect(wrapper.text()).toContain('2026-01-02')
    })
  })

  it('should show project usage with session grouping when toggled', async () => {
    setupSession('sess-4', '/tmp/test')

    mockSessionUsageResponse({
      total_input_tokens: 0,
      total_output_tokens: 0,
      total_tokens: 0,
      compression_count: 0,
    })
    mockProjectUsageResponse({
      summary: { input: 0, output: 0, total: 0 },
      groups: [],
    })

    const wrapper = mount(DashboardPanel)

    await vi.waitFor(() => {
      const sessionBtn = wrapper.find('.group-by-switch button:last-child')
      expect(sessionBtn.exists()).toBe(true)
    })
  })

  it('should show empty state when no workspace', async () => {
    vi.mocked(fetch).mockResolvedValue({ ok: true, json: () => Promise.resolve({}) } as Response)
    vi.mocked(fetch).mockResolvedValue({ ok: true, json: () => Promise.resolve({}) } as Response)

    const store = useSessionStore()
    store.currentSession = null
    store.workingDirectory = ''

    const wrapper = mount(DashboardPanel)

    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('暂无活跃会话')
  })

  it('should format large numbers with k/M suffixes', async () => {
    setupSession('sess-5', '/tmp/test')

    mockSessionUsageResponse({
      total_input_tokens: 1_500_000,
      total_output_tokens: 500_000,
      total_tokens: 2_000_000,
      compression_count: 0,
    })
    mockProjectUsageResponse({
      summary: {
        input: 1_500_000,
        output: 500_000,
        total: 2_000_000,
      },
      groups: [],
    })

    const wrapper = mount(DashboardPanel)

    await vi.waitFor(() => {
      expect(wrapper.text()).toContain('1.5M')
      expect(wrapper.text()).toContain('500.0k')
      expect(wrapper.text()).toContain('2.0M')
    })
  })
})