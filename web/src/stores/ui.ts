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

export type ThemeType = 'light' | 'dark'
export type ModalType = 'approval' | 'session-picker' | 'rollback-picker' | 'help' | null

function loadTheme(): ThemeType {
  try {
    const stored = localStorage.getItem('devo-theme')
    if (stored === 'dark' || stored === 'light') return stored
  } catch {}
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

function saveTheme(theme: ThemeType): void {
  try {
    localStorage.setItem('devo-theme', theme)
  } catch {}
}

export const useUiStore = defineStore('ui', () => {
  const toasts = ref<Toast[]>([])
  const activeModal = ref<ModalType>(null)
  const connectionStatus = ref<'connected' | 'disconnected' | 'connecting'>('disconnected')
  const focusInputCounter = ref(0)
  const pendingCommand = ref<string | null>(null)
  const theme = ref<ThemeType>(loadTheme())

  function requestFocusInput(): void {
    focusInputCounter.value++
  }

  function setPendingCommand(cmd: string): void {
    pendingCommand.value = cmd
  }

  function clearPendingCommand(): void {
    pendingCommand.value = null
  }

  function toggleTheme(): void {
    theme.value = theme.value === 'light' ? 'dark' : 'light'
    saveTheme(theme.value)
  }

  function setTheme(newTheme: ThemeType): void {
    theme.value = newTheme
    saveTheme(theme.value)
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
    pendingCommand,
    theme,
    showToast,
    removeToast,
    setConnectionStatus,
    setActiveModal,
    setPendingCommand,
    clearPendingCommand,
    requestFocusInput,
    toggleTheme,
    setTheme,
  }
})