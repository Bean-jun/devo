import { defineStore } from 'pinia'
import { ref } from 'vue'
import { generateId } from '@/utils/formatters'
import { TOAST_DURATION, API_BASE } from '@/utils/constants'

export type ToastType = 'success' | 'error' | 'info' | 'warning'

export interface Toast {
  id: string
  type: ToastType
  message: string
  duration: number
}

export type ThemeType = 'light' | 'dark'
export type ModalType = 'approval' | 'session-picker' | 'rollback-picker' | 'help' | null

export type RightTabType =
  | 'files'
  | 'skills'
  | 'memory'
  | 'dashboard'
  | 'settings'
  | 'terminal'

export interface WorkspaceEntry {
  id: string
  name: string
  path: string
}

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

function loadActiveWorkspace(): string | null {
  try {
    return localStorage.getItem('devo-active-workspace')
  } catch {
    return null
  }
}

function saveActiveWorkspace(id: string | null): void {
  try {
    if (id) {
      localStorage.setItem('devo-active-workspace', id)
    } else {
      localStorage.removeItem('devo-active-workspace')
    }
  } catch {}
}

function loadSidebarCollapsed(): boolean {
  try {
    return localStorage.getItem('devo-sidebar-collapsed') === 'true'
  } catch {}
  return false
}

function saveSidebarCollapsed(collapsed: boolean): void {
  try {
    localStorage.setItem('devo-sidebar-collapsed', String(collapsed))
  } catch {}
}

export const useUiStore = defineStore('ui', () => {
  const toasts = ref<Toast[]>([])
  const activeModal = ref<ModalType>(null)
  const connectionStatus = ref<'connected' | 'disconnected' | 'connecting'>('disconnected')
  const focusInputCounter = ref(0)
  const pendingCommand = ref<string | null>(null)
  const theme = ref<ThemeType>(loadTheme())
  const isVscodeMode = ref(false)

  const activeWorkspace = ref<string | null>(loadActiveWorkspace())
  const activeRightTab = ref<RightTabType>('files')
  const rightPanelVisible = ref(false)
  const sidebarCollapsed = ref(loadSidebarCollapsed())

  const workspaceList = ref<WorkspaceEntry[]>([])

  async function fetchWorkspaceList(): Promise<WorkspaceEntry[]> {
    try {
      const res = await fetch(`${API_BASE}/workspace`)
      if (!res.ok) return []
      const data = await res.json()
      const list = (data.workspaces || []) as WorkspaceEntry[]
      workspaceList.value = list
      if (!activeWorkspace.value && list.length > 0) {
        activeWorkspace.value = list[0].id
      }
      return list
    } catch {
      return []
    }
  }

  function addWorkspace(path: string): void {
    if (!path) return
    const existing = workspaceList.value.find(w => w.path === path)
    if (existing) {
      if (!activeWorkspace.value) {
        activeWorkspace.value = existing.id
      }
      return
    }
    const name = path.split('/').pop() || path.split('\\').pop() || path
    const entry: WorkspaceEntry = { id: path, name, path }
    workspaceList.value = [entry, ...workspaceList.value]
    if (!activeWorkspace.value) {
      activeWorkspace.value = entry.id
    }
  }

  function removeWorkspace(id: string): void {
    workspaceList.value = workspaceList.value.filter(w => w.id !== id)
    if (activeWorkspace.value === id) {
      activeWorkspace.value = workspaceList.value[0]?.id ?? null
    }
    fetch(`${API_BASE}/workspace?path=${encodeURIComponent(id)}`, { method: 'DELETE' }).catch(() => {})
  }

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

  function setVscodeMode(mode: boolean): void {
    isVscodeMode.value = mode
  }

  let _themeTransitionHandler: ((x: number, y: number, cb: () => void) => void) | null = null

  function registerThemeTransition(handler: (x: number, y: number, cb: () => void) => void): void {
    _themeTransitionHandler = handler
  }

  function toggleThemeWithTransition(x: number, y: number): void {
    const newTheme = theme.value === 'light' ? 'dark' : 'light'
    if (_themeTransitionHandler) {
      _themeTransitionHandler(x, y, () => {
        theme.value = newTheme
        saveTheme(theme.value)
      })
    } else {
      theme.value = newTheme
      saveTheme(theme.value)
    }
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

  function setActiveWorkspace(id: string | null): void {
    activeWorkspace.value = id
    saveActiveWorkspace(id)
  }

  function setActiveRightTab(tab: RightTabType): void {
    activeRightTab.value = tab
    if (!rightPanelVisible.value) {
      rightPanelVisible.value = true
    }
  }

  function toggleRightPanel(): void {
    rightPanelVisible.value = !rightPanelVisible.value
  }

  function closeRightPanel(): void {
    rightPanelVisible.value = false
  }

  function toggleSidebar(): void {
    sidebarCollapsed.value = !sidebarCollapsed.value
    saveSidebarCollapsed(sidebarCollapsed.value)
  }

  return {
    toasts,
    activeModal,
    connectionStatus,
    focusInputCounter,
    pendingCommand,
    theme,
    isVscodeMode,
    activeWorkspace,
    activeRightTab,
    rightPanelVisible,
    sidebarCollapsed,
    workspaceList,
    fetchWorkspaceList,
    addWorkspace,
    removeWorkspace,
    showToast,
    removeToast,
    setConnectionStatus,
    setActiveModal,
    setActiveWorkspace,
    setActiveRightTab,
    toggleRightPanel,
    closeRightPanel,
    toggleSidebar,
    setPendingCommand,
    clearPendingCommand,
    requestFocusInput,
    toggleTheme,
    setTheme,
    setVscodeMode,
    registerThemeTransition,
    toggleThemeWithTransition,
  }
})