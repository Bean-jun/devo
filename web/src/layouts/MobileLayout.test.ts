import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { useSessionStore } from '@/stores/session'
import { useUiStore } from '@/stores/ui'
import { useChatStore } from '@/stores/chat'
import MobileLayout from '@/layouts/MobileLayout.vue'

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: vi.fn(),
  }),
}))

describe('MobileLayout', () => {
  let container: HTMLDivElement

  beforeEach(() => {
    container = document.createElement('div')
    document.body.appendChild(container)
    setActivePinia(createPinia())
    global.fetch = vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve({}) })
  })

  afterEach(() => {
    document.body.removeChild(container)
  })

  function createSession() {
    const sessionStore = useSessionStore()
    sessionStore.sessions = [
      { id: 's1', title: 'Test Session', state: 'idle' },
    ] as any
    sessionStore.currentSession = { id: 's1', title: 'Test Session', state: 'idle' } as any
  }

  function mountLayout() {
    return mount(MobileLayout, {
      attachTo: container,
      global: {
        stubs: {
          Teleport: true,
          ChatPanel: true,
          MobileInputBar: {
            template: `<div data-test="mobile-input-bar">
              <button data-test="mobile-command-btn" @click="$emit('openCommand')">/</button>
              <textarea data-test="mobile-input-textarea" @keydown.enter.prevent="$emit('send', 'Hello')"></textarea>
            </div>`,
            emits: ['send', 'stop', 'openCommand'],
          },
          MobileCommandSheet: {
            template: `<div data-test="command-sheet">
              <button data-test="command-item" class="command-item" @click="$emit('select', { id: 'files' })">
                <span class="command-name">/files</span>
              </button>
            </div>`,
            emits: ['close', 'select'],
          },
          MobilePanelDrawer: {
            template: '<div data-test="panel-drawer" />',
            emits: ['close'],
          },
          MobileSessionPicker: {
            template: '<div data-test="session-picker" />',
            emits: ['close', 'newSession'],
          },
          MobileWorkspacePicker: {
            template: '<div data-test="workspace-picker" />',
            emits: ['close'],
          },
        },
      },
    })
  }

  it('should render mobile layout', () => {
    createSession()
    const wrapper = mountLayout()

    expect(wrapper.find('[data-test="mobile-input-bar"]').exists()).toBe(true)
  })

  it('should show command sheet when toggle button clicked', async () => {
    createSession()
    const wrapper = mountLayout()

    await wrapper.find('[data-test="mobile-command-btn"]').trigger('click')

    expect(wrapper.find('[data-test="command-sheet"]').exists()).toBe(true)
  })

  it('should show panel drawer when panel command selected', async () => {
    createSession()
    const wrapper = mountLayout()

    await wrapper.find('[data-test="mobile-command-btn"]').trigger('click')
    await wrapper.find('[data-test="command-item"]').trigger('click')

    expect(wrapper.find('[data-test="panel-drawer"]').exists()).toBe(true)
  })

  it('should send message on enter', async () => {
    createSession()
    const wrapper = mountLayout()

    const textarea = wrapper.find('[data-test="mobile-input-textarea"]')
    await textarea.trigger('keydown', { key: 'Enter' })

    expect(wrapper.emitted('send')).toBeFalsy()
  })
})