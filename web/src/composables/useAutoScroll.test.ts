import { describe, it, expect, vi } from 'vitest'
import { useAutoScroll } from '@/composables/useAutoScroll'

describe('useAutoScroll', () => {
  it('should return containerRef', () => {
    const { containerRef } = useAutoScroll()
    expect(containerRef.value).toBeNull()
  })

  it('should detect near bottom', () => {
    const { containerRef, isNearBottom } = useAutoScroll()

    const el = document.createElement('div')
    containerRef.value = el

    const result = isNearBottom()
    expect(typeof result).toBe('boolean')
  })

  it('should show scroll to bottom initially as false', () => {
    const { showScrollToBottom } = useAutoScroll()
    expect(showScrollToBottom.value).toBe(false)
  })

  it('should set showScrollToBottom on scroll up', () => {
    const { containerRef, onScroll, showScrollToBottom } = useAutoScroll()

    const el = document.createElement('div')
    Object.defineProperty(el, 'scrollHeight', { value: 1000, configurable: true })
    Object.defineProperty(el, 'scrollTop', { value: 500, configurable: true })
    Object.defineProperty(el, 'clientHeight', { value: 400, configurable: true })
    containerRef.value = el

    onScroll()

    expect(showScrollToBottom.value).toBe(true)
  })
})