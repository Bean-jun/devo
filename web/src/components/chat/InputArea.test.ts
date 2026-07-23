import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, type DOMWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'
import InputArea from '@/components/chat/InputArea.vue'

async function setEditorText(editor: DOMWrapper<HTMLElement>, text: string): Promise<void> {
  const el = editor.element as HTMLElement
  el.textContent = text
  await nextTick()
  await editor.trigger('input')
  await nextTick()
}

function getEditor(wrapper: ReturnType<typeof mount>): DOMWrapper<HTMLElement> {
  return wrapper.find('[data-test="message-input"]')
}

describe('InputArea', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    document.execCommand = vi.fn(() => true)
  })

  it('should render editor and send button', () => {
    const wrapper = mount(InputArea, {
      props: { isDisabled: false, isProcessing: false },
    })

    expect(getEditor(wrapper).exists()).toBe(true)
    expect(wrapper.find('.btn-send').exists()).toBe(true)
  })

  it('should emit send event with text content on Enter', async () => {
    const wrapper = mount(InputArea, {
      props: { isDisabled: false, isProcessing: false },
    })
    const editor = getEditor(wrapper)

    await setEditorText(editor, 'Hello, world!')
    await editor.trigger('keydown', { key: 'Enter', shiftKey: false })

    expect(wrapper.emitted('send')).toBeTruthy()
    expect(wrapper.emitted('send')![0]).toEqual(['Hello, world!'])
  })

  it('should not emit send event on Shift+Enter', async () => {
    const wrapper = mount(InputArea, {
      props: { isDisabled: false, isProcessing: false },
    })
    const editor = getEditor(wrapper)

    await setEditorText(editor, 'Hello')
    await editor.trigger('keydown', { key: 'Enter', shiftKey: true })

    expect(wrapper.emitted('send')).toBeFalsy()
  })

  it('should clear editor after sending', async () => {
    const wrapper = mount(InputArea, {
      props: { isDisabled: false, isProcessing: false },
    })
    const editor = getEditor(wrapper)

    await setEditorText(editor, 'Test message')
    await editor.trigger('keydown', { key: 'Enter', shiftKey: false })

    expect((editor.element as HTMLElement).textContent).toBe('')
  })

  it('should show stop button when isProcessing is true', () => {
    const wrapper = mount(InputArea, {
      props: { isDisabled: false, isProcessing: true },
    })

    expect(wrapper.find('.btn-stop').exists()).toBe(true)
    expect(wrapper.find('.btn-send').exists()).toBe(false)
  })

  it('should reflect disabled state via is-disabled class when isDisabled is true', () => {
    const wrapper = mount(InputArea, {
      props: { isDisabled: true, isProcessing: false },
    })

    expect(getEditor(wrapper).classes()).toContain('is-disabled')
  })

  it('should show character count', async () => {
    const wrapper = mount(InputArea, {
      props: { isDisabled: false, isProcessing: false },
    })
    const editor = getEditor(wrapper)

    await setEditorText(editor, 'Hi')

    const counter = wrapper.find('[data-test="char-count"]')
    expect(counter.text()).toContain('2')
  })

  it('should emit openCommand on / when input is empty', async () => {
    const wrapper = mount(InputArea, {
      props: { isDisabled: false, isProcessing: false },
    })
    const editor = getEditor(wrapper)

    await editor.trigger('keydown', { key: '/' })

    expect(wrapper.emitted('openCommand')).toBeTruthy()
  })

  it('should not emit openCommand on / when input is not empty', async () => {
    const wrapper = mount(InputArea, {
      props: { isDisabled: false, isProcessing: false },
    })
    const editor = getEditor(wrapper)

    await setEditorText(editor, 'existing text')
    await editor.trigger('keydown', { key: '/' })

    expect(wrapper.emitted('openCommand')).toBeFalsy()
  })

  it('should not send empty message', async () => {
    const wrapper = mount(InputArea, {
      props: { isDisabled: false, isProcessing: false },
    })
    const editor = getEditor(wrapper)

    await setEditorText(editor, '   ')
    await editor.trigger('keydown', { key: 'Enter', shiftKey: false })

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
    const editor = getEditor(wrapper)

    await setEditorText(editor, '/new my session')
    await editor.trigger('keydown', { key: 'Enter', shiftKey: false })

    expect(wrapper.emitted('executeCommand')).toBeTruthy()
    expect(wrapper.emitted('executeCommand')![0]).toEqual(['/new my session'])
    expect(wrapper.emitted('send')).toBeFalsy()
  })

  it('should emit executeCommand for parameterless commands', async () => {
    const wrapper = mount(InputArea, {
      props: { isDisabled: false, isProcessing: false },
    })
    const editor = getEditor(wrapper)

    await setEditorText(editor, '/pause')
    await editor.trigger('keydown', { key: 'Enter', shiftKey: false })

    expect(wrapper.emitted('executeCommand')).toBeTruthy()
    expect(wrapper.emitted('executeCommand')![0]).toEqual(['/pause'])
  })

  it('should preserve multi-line content without folding', async () => {
    const wrapper = mount(InputArea, {
      props: { isDisabled: false, isProcessing: false },
    })
    const editor = getEditor(wrapper)

    const multiLineText = 'line1\nline2\nline3\nline4'
    await setEditorText(editor, multiLineText)

    expect((editor.element as HTMLElement).textContent).toBe(multiLineText)
    expect((editor.element as HTMLElement).textContent).not.toContain('已粘贴')
  })

  it('should send multi-line content verbatim on Enter', async () => {
    const wrapper = mount(InputArea, {
      props: { isDisabled: false, isProcessing: false },
    })
    const editor = getEditor(wrapper)

    const multiLineText = 'line1\nline2\nline3\nline4'
    await setEditorText(editor, multiLineText)
    await editor.trigger('keydown', { key: 'Enter', shiftKey: false })

    expect(wrapper.emitted('send')).toBeTruthy()
    expect(wrapper.emitted('send')![0]).toEqual([multiLineText])
  })

  it('should send large text without folding', async () => {
    const wrapper = mount(InputArea, {
      props: { isDisabled: false, isProcessing: false },
    })
    const editor = getEditor(wrapper)

    const largeText = Array.from({ length: 50 }, (_, i) => `line ${i + 1}`).join('\n')
    await setEditorText(editor, largeText)
    await editor.trigger('keydown', { key: 'Enter', shiftKey: false })

    expect(wrapper.emitted('send')![0]).toEqual([largeText])
  })

  it('should show placeholder class when empty', () => {
    const wrapper = mount(InputArea, {
      props: { isDisabled: false, isProcessing: false },
    })

    expect(getEditor(wrapper).classes()).toContain('is-empty')
  })

  it('should hide placeholder class when text is present', async () => {
    const wrapper = mount(InputArea, {
      props: { isDisabled: false, isProcessing: false },
    })
    const editor = getEditor(wrapper)

    await setEditorText(editor, 'some text')

    expect(editor.classes()).not.toContain('is-empty')
  })
})
