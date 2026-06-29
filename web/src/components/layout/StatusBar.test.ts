import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import StatusBar from '@/components/layout/StatusBar.vue'
import { useSessionStore } from '@/stores/session'
import { useUiStore } from '@/stores/ui'

describe('StatusBar', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('should render default status', () => {
    const sessionStore = useSessionStore()
    sessionStore.currentSession = {
      id: 'sess-1',
      title: 'Test',
      state: 'idle',
      workingDirectory: '',
      createdAt: '',
      lastActiveAt: '',
      messageCount: 0,
      tokenUsage: { input: 0, output: 0 },
      trustLevel: 'normal',
      approvalPolicy: {},
    }

    const wrapper = mount(StatusBar)

    const statusIndicator = wrapper.find('.status-indicator')
    expect(statusIndicator.exists()).toBe(true)
  })

  it('should render session name', async () => {
    const sessionStore = useSessionStore()
    sessionStore.currentSession = {
      id: 'sess-1',
      title: 'My Project',
      state: 'idle',
      workingDirectory: '/tmp',
      createdAt: '',
      lastActiveAt: '',
      messageCount: 0,
      tokenUsage: { input: 0, output: 0 },
      trustLevel: 'normal',
      approvalPolicy: {},
    }

    const wrapper = mount(StatusBar)

    const name = wrapper.find('.session-name')
    expect(name.exists()).toBe(true)
    expect(name.text()).toBe('My Project')
  })

  it('should render connection status', () => {
    const sessionStore = useSessionStore()
    sessionStore.currentSession = {
      id: 'sess-1',
      title: 'Test',
      state: 'idle',
      workingDirectory: '',
      createdAt: '',
      lastActiveAt: '',
      messageCount: 0,
      tokenUsage: { input: 0, output: 0 },
      trustLevel: 'normal',
      approvalPolicy: {},
    }

    const uiStore = useUiStore()
    uiStore.setConnectionStatus('connected')

    const wrapper = mount(StatusBar)

    expect(wrapper.find('.connection-status').exists()).toBe(true)
  })
})