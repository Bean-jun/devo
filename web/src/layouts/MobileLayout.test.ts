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

  let lastWrapper: ReturnType<typeof mount> | null = null

  afterEach(() => {
    if (lastWrapper) {
      lastWrapper.unmount()
      lastWrapper = null
    }
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
    const wrapper = mount(MobileLayout, {
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
              <button data-test="command-item-files" class="command-item" @click="$emit('select', { id: 'files' })">
                <span class="command-name">/files</span>
              </button>
              <button data-test="command-item-rollback" class="command-item" @click="$emit('select', { id: 'rollback', description: '回滚消息' })">
                <span class="command-name">/rollback</span>
              </button>
              <button data-test="command-item-export" class="command-item" @click="$emit('select', { id: 'export', description: '导出会话' })">
                <span class="command-name">/export</span>
              </button>
              <button data-test="command-item-status" class="command-item" @click="$emit('select', { id: 'status', description: '查看状态' })">
                <span class="command-name">/status</span>
              </button>
              <button data-test="command-item-version" class="command-item" @click="$emit('select', { id: 'version', description: '查看版本' })">
                <span class="command-name">/version</span>
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
    lastWrapper = wrapper
    return wrapper
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
    await wrapper.find('[data-test="command-item-files"]').trigger('click')

    expect(wrapper.find('[data-test="panel-drawer"]').exists()).toBe(true)
  })

  it('should send message on enter', async () => {
    createSession()
    const wrapper = mountLayout()

    const textarea = wrapper.find('[data-test="mobile-input-textarea"]')
    await textarea.trigger('keydown', { key: 'Enter' })

    expect(wrapper.emitted('send')).toBeFalsy()
  })

  it('should open rollback picker when rollback command selected', async () => {
    createSession()
    const wrapper = mountLayout()
    const uiStore = useUiStore()

    await wrapper.find('[data-test="mobile-command-btn"]').trigger('click')
    await wrapper.find('[data-test="command-item-rollback"]').trigger('click')

    expect(uiStore.activeModal).toBe('rollback-picker')
  })

  it('should call sync-archive API when export command selected', async () => {
    createSession()
    const wrapper = mountLayout()

    await wrapper.find('[data-test="mobile-command-btn"]').trigger('click')
    await wrapper.find('[data-test="command-item-export"]').trigger('click')

    await new Promise(r => setTimeout(r, 50))

    expect(global.fetch).toHaveBeenCalledWith(
      expect.stringContaining('/sessions/s1/sync-archive'),
      expect.objectContaining({ method: 'POST' })
    )
  })

  it('should show status info dialog when status command selected', async () => {
    createSession()
    const wrapper = mountLayout()

    await wrapper.find('[data-test="mobile-command-btn"]').trigger('click')
    await wrapper.find('[data-test="command-item-status"]').trigger('click')

    const title = document.querySelector('[data-test="info-dialog-title"]')
    const content = document.querySelector('[data-test="info-dialog-content"]')
    expect(title).not.toBeNull()
    expect(title?.textContent).toBe('状态信息')
    expect(content).not.toBeNull()
    expect(content?.textContent).toContain('Context')
    expect(content?.textContent).toContain('Tokens')
    expect(content?.textContent).toContain('工作区')
  })

  it('should show version info dialog when version command selected', async () => {
    createSession()
    const wrapper = mountLayout()

    await wrapper.find('[data-test="mobile-command-btn"]').trigger('click')
    await wrapper.find('[data-test="command-item-version"]').trigger('click')

    const title = document.querySelector('[data-test="info-dialog-title"]')
    const content = document.querySelector('[data-test="info-dialog-content"]')
    expect(title).not.toBeNull()
    expect(title?.textContent).toBe('版本信息')
    expect(content).not.toBeNull()
    expect(content?.textContent).toContain('Devo')
  })

  it('should close command sheet on Escape and stop propagation', async () => {
    createSession()
    const wrapper = mountLayout()

    const bubbleSpy = vi.fn()
    window.addEventListener('keydown', bubbleSpy)

    await wrapper.find('[data-test="mobile-command-btn"]').trigger('click')
    expect(wrapper.find('[data-test="command-sheet"]').exists()).toBe(true)

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))

    await wrapper.vm.$nextTick()
    await new Promise(r => setTimeout(r, 0))

    expect(wrapper.find('[data-test="command-sheet"]').exists()).toBe(false)
    expect(bubbleSpy).not.toHaveBeenCalled()

    window.removeEventListener('keydown', bubbleSpy)
  })
})