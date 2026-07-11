import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { useSessionStore } from '@/stores/session'
import MobileSessionPicker from '@/components/mobile/MobileSessionPicker.vue'

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: vi.fn(),
  }),
}))

describe('MobileSessionPicker', () => {
  let container: HTMLDivElement

  beforeEach(() => {
    container = document.createElement('div')
    document.body.appendChild(container)
    setActivePinia(createPinia())
    global.fetch = vi.fn().mockResolvedValue({ ok: true })
  })

  afterEach(() => {
    document.body.removeChild(container)
  })

  function mountPicker() {
    return mount(MobileSessionPicker, {
      attachTo: container,
      global: {
        stubs: {
          Teleport: true,
        },
      },
    })
  }

  it('should render title and new session button', () => {
    const wrapper = mountPicker()

    expect(wrapper.find('.picker-title').text()).toBe('切换会话')
    expect(wrapper.find('.new-session-btn').exists()).toBe(true)
  })

  it('should render cancel button', () => {
    const wrapper = mountPicker()

    expect(wrapper.find('.picker-cancel').exists()).toBe(true)
  })

  it('should render session list items', () => {
    const sessionStore = useSessionStore()
    sessionStore.sessions = [
      { id: 's1', title: 'Session 1', state: 'idle' },
      { id: 's2', title: 'Session 2', state: 'idle' },
    ] as any

    const wrapper = mountPicker()

    const items = wrapper.findAll('.picker-item')
    expect(items.length).toBe(2)
  })

  it('should highlight current session', () => {
    const sessionStore = useSessionStore()
    sessionStore.sessions = [
      { id: 's1', title: 'Session 1', state: 'idle' },
      { id: 's2', title: 'Session 2', state: 'idle' },
    ] as any
    sessionStore.currentSession = { id: 's2', title: 'Session 2', state: 'idle' } as any

    const wrapper = mountPicker()

    const activeItem = wrapper.find('.picker-item.active')
    expect(activeItem.exists()).toBe(true)
    expect(activeItem.find('.picker-item-name').text()).toBe('Session 2')
  })

  it('should show check mark on current session', () => {
    const sessionStore = useSessionStore()
    sessionStore.sessions = [
      { id: 's1', title: 'Session 1', state: 'idle' },
    ] as any
    sessionStore.currentSession = { id: 's1', title: 'Session 1', state: 'idle' } as any

    const wrapper = mountPicker()

    expect(wrapper.find('.picker-check').exists()).toBe(true)
  })

  it('should emit close when cancel button clicked', async () => {
    const wrapper = mountPicker()

    await wrapper.find('.picker-cancel').trigger('click')

    expect(wrapper.emitted('close')).toBeTruthy()
  })

  it('should emit newSession when new button clicked', async () => {
    const wrapper = mountPicker()

    await wrapper.find('.new-session-btn').trigger('click')

    expect(wrapper.emitted('newSession')).toBeTruthy()
  })

  it('should show empty state when no sessions', () => {
    const sessionStore = useSessionStore()
    sessionStore.sessions = []

    const wrapper = mountPicker()

    expect(wrapper.find('.picker-empty').exists()).toBe(true)
  })

  it('should display session items', () => {
    const sessionStore = useSessionStore()
    sessionStore.sessions = [
      { id: 's1', title: 'Session 1', state: 'idle' },
    ] as any

    const wrapper = mountPicker()

    expect(wrapper.find('.picker-item').exists()).toBe(true)
  })
})