import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import InputArea from '@/components/chat/InputArea.vue'

describe('InputArea', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('should render textarea and send button', () => {
    const wrapper = mount(InputArea, {
      props: { isDisabled: false, isProcessing: false },
    })

    expect(wrapper.find('textarea').exists()).toBe(true)
    expect(wrapper.find('.btn-send').exists()).toBe(true)
  })

  it('should emit send event with text content on Enter', async () => {
    const wrapper = mount(InputArea, {
      props: { isDisabled: false, isProcessing: false },
    })
    const textarea = wrapper.find('textarea')

    await textarea.setValue('Hello, world!')
    await textarea.trigger('keydown', { key: 'Enter', shiftKey: false })

    expect(wrapper.emitted('send')).toBeTruthy()
    expect(wrapper.emitted('send')![0]).toEqual(['Hello, world!'])
  })

  it('should not emit send event on Shift+Enter', async () => {
    const wrapper = mount(InputArea, {
      props: { isDisabled: false, isProcessing: false },
    })
    const textarea = wrapper.find('textarea')

    await textarea.setValue('Hello')
    await textarea.trigger('keydown', { key: 'Enter', shiftKey: true })

    expect(wrapper.emitted('send')).toBeFalsy()
  })

  it('should clear textarea after sending', async () => {
    const wrapper = mount(InputArea, {
      props: { isDisabled: false, isProcessing: false },
    })
    const textarea = wrapper.find('textarea')

    await textarea.setValue('Test message')
    await textarea.trigger('keydown', { key: 'Enter', shiftKey: false })

    expect((textarea.element as HTMLTextAreaElement).value).toBe('')
  })

  it('should show stop button when isProcessing is true', () => {
    const wrapper = mount(InputArea, {
      props: { isDisabled: false, isProcessing: true },
    })

    expect(wrapper.find('.btn-stop').exists()).toBe(true)
    expect(wrapper.find('.btn-send').exists()).toBe(false)
  })

  it('should disable input when isDisabled is true', () => {
    const wrapper = mount(InputArea, {
      props: { isDisabled: true, isProcessing: false },
    })

    const textarea = wrapper.find('textarea')
    expect(textarea.attributes('disabled')).toBeDefined()
  })

  it('should show character count', async () => {
    const wrapper = mount(InputArea, {
      props: { isDisabled: false, isProcessing: false },
    })
    const textarea = wrapper.find('textarea')

    await textarea.setValue('Hi')

    const counter = wrapper.find('[data-test="char-count"]')
    expect(counter.text()).toContain('2')
  })

  it('should emit openCommand on / when input is empty', async () => {
    const wrapper = mount(InputArea, {
      props: { isDisabled: false, isProcessing: false },
    })
    const textarea = wrapper.find('textarea')

    await textarea.setValue('')
    await textarea.trigger('keydown', { key: '/' })

    expect(wrapper.emitted('openCommand')).toBeTruthy()
  })

  it('should not send empty message', async () => {
    const wrapper = mount(InputArea, {
      props: { isDisabled: false, isProcessing: false },
    })
    const textarea = wrapper.find('textarea')

    await textarea.setValue('   ')
    await textarea.trigger('keydown', { key: 'Enter', shiftKey: false })

    expect(wrapper.emitted('send')).toBeFalsy()
  })

  it('should emit stop event', async () => {
    const wrapper = mount(InputArea, {
      props: { isDisabled: false, isProcessing: true },
    })

    await wrapper.find('.btn-stop').trigger('click')

    expect(wrapper.emitted('stop')).toBeTruthy()
  })

  it('should emit executeCommand when input starts with /', async () => {
    const wrapper = mount(InputArea, {
      props: { isDisabled: false, isProcessing: false },
    })
    const textarea = wrapper.find('textarea')

    await textarea.setValue('/new my session')
    await textarea.trigger('keydown', { key: 'Enter', shiftKey: false })

    expect(wrapper.emitted('executeCommand')).toBeTruthy()
    expect(wrapper.emitted('executeCommand')![0]).toEqual(['/new my session'])
    expect(wrapper.emitted('send')).toBeFalsy()
  })

  it('should emit executeCommand for parameterless commands', async () => {
    const wrapper = mount(InputArea, {
      props: { isDisabled: false, isProcessing: false },
    })
    const textarea = wrapper.find('textarea')

    await textarea.setValue('/pause')
    await textarea.trigger('keydown', { key: 'Enter', shiftKey: false })

    expect(wrapper.emitted('executeCommand')).toBeTruthy()
    expect(wrapper.emitted('executeCommand')![0]).toEqual(['/pause'])
  })
})