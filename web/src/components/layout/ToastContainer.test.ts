import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { useUiStore } from '@/stores/ui'
import ToastContainer from '@/components/layout/ToastContainer.vue'

describe('ToastContainer', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('should render no toasts initially', () => {
    const wrapper = mount(ToastContainer)

    expect(document.querySelector('.toast-container')).toBeNull()
  })

  it('should render toasts', () => {
    const uiStore = useUiStore()
    uiStore.showToast('success', '操作成功')
    uiStore.showToast('error', '操作失败')

    const wrapper = mount(ToastContainer)

    expect(document.querySelector('.toast-container')).toBeTruthy()
  })

  it('should remove toast after duration', () => {
    vi.useFakeTimers()
    const uiStore = useUiStore()
    uiStore.showToast('info', '自动消失', 1000)

    const wrapper = mount(ToastContainer)
    expect(document.querySelector('.toast-container')).toBeTruthy()

    vi.advanceTimersByTime(1000)
    expect(uiStore.toasts).toHaveLength(0)

    vi.useRealTimers()
  })
})