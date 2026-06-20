import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { useUiStore } from '@/stores/ui'
import { useSessionStore } from '@/stores/session'
import SessionPicker from '@/components/modal/SessionPicker.vue'

describe('SessionPicker', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve([]),
    } as Response)
  })

  it('should not render when modal is not active', () => {
    const wrapper = mount(SessionPicker)

    expect(wrapper.find('.session-picker').exists()).toBe(false)
  })

  it('should render when active', () => {
    const uiStore = useUiStore()
    uiStore.setActiveModal('session-picker')

    const wrapper = mount(SessionPicker)

    expect(wrapper.find('.session-picker').exists()).toBe(true)
  })

  it('should show sessions in list', async () => {
    const uiStore = useUiStore()
    const sessionStore = useSessionStore()

    sessionStore.sessions = [
      { id: 's1', title: 'Session 1', state: 'idle', messageCount: 5, lastActiveAt: '2026-01-01T00:00:00Z' } as any,
      { id: 's2', title: 'Session 2', state: 'archived', messageCount: 10, lastActiveAt: '2026-01-02T00:00:00Z' } as any,
    ]

    uiStore.setActiveModal('session-picker')

    const wrapper = mount(SessionPicker)

    expect(wrapper.findAll('.picker-item')).toHaveLength(2)
  })

  it('should show empty state', () => {
    const uiStore = useUiStore()
    uiStore.setActiveModal('session-picker')

    const wrapper = mount(SessionPicker)

    expect(wrapper.find('.picker-empty').text()).toContain('暂无')
  })
})