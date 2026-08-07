import { useUiStore } from '@/stores/ui'

export function useTerminalPanel() {
  const uiStore = useUiStore()
  return { uiStore }
}