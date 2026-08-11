import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { useSessionStore } from '@/stores/session'
import MobileInputBar from '@/components/mobile/MobileInputBar.vue'

describe('MobileInputBar', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('should render textarea and send button', () => {
    const wrapper = mount(MobileInputBar, {
      props: { isDisabled: false, isProcessing: false },
    })

    expect(wrapper.find('textarea').exists()).toBe(true)
    expect(wrapper.find('.mobile-btn-send').exists()).toBe(true)
  })

  it('should render command button with / text', () => {
    const wrapper = mount(MobileInputBar, {
      props: { isDisabled: false, isProcessing: false },
    })

    const cmdBtn = wrapper.find('.command-btn')
    expect(cmdBtn.exists()).toBe(true)
    expect(cmdBtn.text()).toBe('/')
  })

  it('should emit send event with text content on Enter', async () => {
    const wrapper = mount(MobileInputBar, {
      props: { isDisabled: false, isProcessing: false },
    })
    const textarea = wrapper.find('textarea')

    await textarea.setValue('Hello, world!')
    await textarea.trigger('keydown', { key: 'Enter', shiftKey: false })

    expect(wrapper.emitted('send')).toBeTruthy()
    expect(wrapper.emitted('send')![0]).toEqual(['Hello, world!', undefined])
  })

  it('should not emit send event on Shift+Enter', async () => {
    const wrapper = mount(MobileInputBar, {
      props: { isDisabled: false, isProcessing: false },
    })
    const textarea = wrapper.find('textarea')

    await textarea.setValue('Hello')
    await textarea.trigger('keydown', { key: 'Enter', shiftKey: true })

    expect(wrapper.emitted('send')).toBeFalsy()
  })

  it('should clear textarea after sending', async () => {
    const wrapper = mount(MobileInputBar, {
      props: { isDisabled: false, isProcessing: false },
    })
    const textarea = wrapper.find('textarea')

    await textarea.setValue('Test message')
    await textarea.trigger('keydown', { key: 'Enter', shiftKey: false })

    expect((textarea.element as HTMLTextAreaElement).value).toBe('')
  })

  it('should show stop button when isProcessing is true', () => {
    const wrapper = mount(MobileInputBar, {
      props: { isDisabled: false, isProcessing: true },
    })

    expect(wrapper.find('.mobile-btn-stop').exists()).toBe(true)
    expect(wrapper.find('.mobile-btn-send').exists()).toBe(false)
  })

  it('should disable send button when input is empty', () => {
    const wrapper = mount(MobileInputBar, {
      props: { isDisabled: false, isProcessing: false },
    })

    const sendBtn = wrapper.find('.mobile-btn-send')
    expect(sendBtn.attributes('disabled')).toBeDefined()
  })

  it('should enable send button when input has text', async () => {
    const wrapper = mount(MobileInputBar, {
      props: { isDisabled: false, isProcessing: false },
    })
    const textarea = wrapper.find('textarea')

    await textarea.setValue('Hi')

    const sendBtn = wrapper.find('.mobile-btn-send')
    expect(sendBtn.attributes('disabled')).toBeUndefined()
  })

  it('should disable input when isDisabled is true', () => {
    const wrapper = mount(MobileInputBar, {
      props: { isDisabled: true, isProcessing: false },
    })

    const textarea = wrapper.find('textarea')
    expect(textarea.attributes('disabled')).toBeDefined()
  })

  it('should not send empty message', async () => {
    const wrapper = mount(MobileInputBar, {
      props: { isDisabled: false, isProcessing: false },
    })
    const textarea = wrapper.find('textarea')

    await textarea.setValue('   ')
    await textarea.trigger('keydown', { key: 'Enter', shiftKey: false })

    expect(wrapper.emitted('send')).toBeFalsy()
  })

  it('should emit stop event when stop button clicked', async () => {
    const wrapper = mount(MobileInputBar, {
      props: { isDisabled: false, isProcessing: true },
    })

    await wrapper.find('.mobile-btn-stop').trigger('click')

    expect(wrapper.emitted('stop')).toBeTruthy()
  })

  it('should emit openCommand when command button clicked', async () => {
    const wrapper = mount(MobileInputBar, {
      props: { isDisabled: false, isProcessing: false },
    })

    await wrapper.find('.command-btn').trigger('click')

    expect(wrapper.emitted('openCommand')).toBeTruthy()
  })

  it('should render footer with context and token usage', async () => {
    const sessionStore = useSessionStore()
    sessionStore.currentSession = {
      id: 's1',
      title: 'Test',
      state: 'idle',
      workingDirectory: '/home/user',
      currentContextTokens: 12500,
      tokenUsage: { input: 10200, output: 5100 },
    } as any

    const wrapper = mount(MobileInputBar, {
      props: { isDisabled: false, isProcessing: false },
    })

    expect(wrapper.find('.mobile-input-footer').exists()).toBe(true)
    expect(wrapper.text()).toContain('Context')
    expect(wrapper.text()).toContain('Tokens')
  })

  it('should show context value with warn color', async () => {
    const sessionStore = useSessionStore()
    sessionStore.currentSession = {
      id: 's1',
      title: 'Test',
      state: 'idle',
      workingDirectory: '/home/user',
      currentContextTokens: 12500,
      tokenUsage: { input: 10200, output: 5100 },
    } as any

    const wrapper = mount(MobileInputBar, {
      props: { isDisabled: false, isProcessing: false },
    })

    expect(wrapper.find('.context-warn').exists()).toBe(true)
  })
})