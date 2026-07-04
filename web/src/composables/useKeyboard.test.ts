import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import { useKeyboard } from '@/composables/useKeyboard'

describe('useKeyboard', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  function mountWithKeyboard(shortcuts: Parameters<typeof useKeyboard>[0]) {
    const wrapper = mount(
      defineComponent({
        setup() {
          useKeyboard(shortcuts)
          return () => h('div')
        },
      })
    )
    return wrapper
  }

  it('should trigger handler on key combination', () => {
    const handler = vi.fn()
    const shortcuts = [{ key: 'k', ctrl: true, handler }]

    mountWithKeyboard(shortcuts)

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'k', ctrlKey: true }))

    expect(handler).toHaveBeenCalled()
  })

  it('should not trigger when ctrl not pressed', () => {
    const handler = vi.fn()
    const shortcuts = [{ key: 'k', ctrl: true, handler }]

    mountWithKeyboard(shortcuts)

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'k', ctrlKey: false }))

    expect(handler).not.toHaveBeenCalled()
  })

  it('should trigger without ctrl requirement', () => {
    const handler = vi.fn()
    const shortcuts = [{ key: 'Escape', handler }]

    mountWithKeyboard(shortcuts)

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))

    expect(handler).toHaveBeenCalled()
  })

  it('should handle shift key requirement', () => {
    const handler = vi.fn()
    const shortcuts = [{ key: 'Enter', shift: true, handler }]

    mountWithKeyboard(shortcuts)

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', shiftKey: true }))

    expect(handler).toHaveBeenCalled()
  })

  it('should not trigger when shift not pressed', () => {
    const handler = vi.fn()
    const shortcuts = [{ key: 'Enter', shift: true, handler }]

    mountWithKeyboard(shortcuts)

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', shiftKey: false }))

    expect(handler).not.toHaveBeenCalled()
  })

  it('should handle alt key requirement', () => {
    const handler = vi.fn()
    const shortcuts = [{ key: 'y', alt: true, handler }]

    mountWithKeyboard(shortcuts)

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'y', altKey: true }))

    expect(handler).toHaveBeenCalled()
  })

  it('should not trigger when alt not pressed', () => {
    const handler = vi.fn()
    const shortcuts = [{ key: 'y', alt: true, handler }]

    mountWithKeyboard(shortcuts)

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'y', altKey: false }))

    expect(handler).not.toHaveBeenCalled()
  })

  it('should not trigger when ctrl pressed but not required', () => {
    const handler = vi.fn()
    const shortcuts = [{ key: 'Escape', handler }]

    mountWithKeyboard(shortcuts)

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', ctrlKey: true }))

    expect(handler).not.toHaveBeenCalled()
  })
})