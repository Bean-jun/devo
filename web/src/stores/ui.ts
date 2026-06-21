import { defineStore } from 'pinia'
import { ref } from 'vue'
import { generateId } from '@/utils/formatters'
import { TOAST_DURATION } from '@/utils/constants'

export type ToastType = 'success' | 'error' | 'info' | 'warning'

export interface Toast {
  id: string
  type: ToastType
  message: string
  duration: number
}

export type ModalType = 'approval' | 'session-picker' | 'rollback-picker' | 'help' | 'input-prompt' | null

export interface InputPromptConfig {
  title: string
  placeholder?: string
  defaultValue?: string
  confirmLabel?: string
  onConfirm: (value: string) => void
  onCancel?: () => void
}

export const useUiStore = defineStore('ui', () => {
  const toasts = ref<Toast[]>([])
  const activeModal = ref<ModalType>(null)
  const connectionStatus = ref<'connected' | 'disconnected' | 'connecting'>('disconnected')
  const focusInputCounter = ref(0)
  const inputPrompt = ref<InputPromptConfig>({
    title: '',
    placeholder: '',
    defaultValue: '',
    confirmLabel: '确定',
    onConfirm: () => {},
    onCancel: () => {},
  })

  function requestFocusInput(): void {
    focusInputCounter.value++
  }

  function openInputPrompt(config: InputPromptConfig): void {
    inputPrompt.value = { ...config }
    activeModal.value = 'input-prompt'
  }

  function showToast(type: ToastType, message: string, duration: number = TOAST_DURATION): void {
    const toast: Toast = {
      id: generateId(),
      type,
      message,
      duration,
    }
    toasts.value.push(toast)

    if (toasts.value.length > 5) {
      toasts.value.shift()
    }

    setTimeout(() => {
      removeToast(toast.id)
    }, duration)
  }

  function removeToast(id: string): void {
    const idx = toasts.value.findIndex(t => t.id === id)
    if (idx !== -1) {
      toasts.value.splice(idx, 1)
    }
  }

  function setConnectionStatus(status: 'connected' | 'disconnected' | 'connecting'): void {
    connectionStatus.value = status
  }

  function setActiveModal(modal: ModalType): void {
    activeModal.value = modal
    if (modal === null) {
      requestFocusInput()
    }
  }

  return {
    toasts,
    activeModal,
    connectionStatus,
    focusInputCounter,
    inputPrompt,
    showToast,
    removeToast,
    setConnectionStatus,
    setActiveModal,
    openInputPrompt,
    requestFocusInput,
  }
})