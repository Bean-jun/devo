import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useUiStore } from '@/stores/ui'

describe('UiStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  describe('showToast', () => {
    it('should add toast message', () => {
      const store = useUiStore()

      store.showToast('success', '操作成功')

      expect(store.toasts).toHaveLength(1)
      expect(store.toasts[0].type).toBe('success')
      expect(store.toasts[0].message).toBe('操作成功')
    })

    it('should auto-remove toast after duration', () => {
      const store = useUiStore()

      store.showToast('info', '测试消息', 1000)

      expect(store.toasts).toHaveLength(1)

      vi.advanceTimersByTime(1000)

      expect(store.toasts).toHaveLength(0)
    })

    it('should limit toasts to 5', () => {
      const store = useUiStore()

      for (let i = 0; i < 7; i++) {
        store.showToast('info', `消息 ${i}`)
      }

      expect(store.toasts).toHaveLength(5)
    })
  })

  describe('removeToast', () => {
    it('should remove toast by ID', () => {
      const store = useUiStore()

      store.showToast('success', '消息')
      const id = store.toasts[0].id
      store.removeToast(id)

      expect(store.toasts).toHaveLength(0)
    })

    it('should do nothing for non-existent ID', () => {
      const store = useUiStore()

      store.showToast('success', '消息')
      store.removeToast('non-existent')

      expect(store.toasts).toHaveLength(1)
    })
  })

  describe('setConnectionStatus', () => {
    it('should set connection status', () => {
      const store = useUiStore()

      store.setConnectionStatus('connected')
      expect(store.connectionStatus).toBe('connected')

      store.setConnectionStatus('disconnected')
      expect(store.connectionStatus).toBe('disconnected')
    })
  })

  describe('setActiveModal', () => {
    it('should set active modal', () => {
      const store = useUiStore()

      store.setActiveModal('approval')
      expect(store.activeModal).toBe('approval')

      store.setActiveModal(null)
      expect(store.activeModal).toBeNull()
    })
  })
})