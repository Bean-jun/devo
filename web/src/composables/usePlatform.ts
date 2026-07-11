import { computed } from 'vue'
import { useUiStore } from '@/stores/ui'

export type LayoutMode = 'browser' | 'vscode' | 'mobile'

export function usePlatform() {
  const uiStore = useUiStore()

  const isVscodeMode = computed(() => uiStore.layoutMode === 'vscode')

  const isBrowserMode = computed(() => uiStore.layoutMode === 'browser')

  const isMobileMode = computed(() => uiStore.layoutMode === 'mobile')

  const layoutMode = computed(() => uiStore.layoutMode)

  function detectMode(): void {
    const params = new URLSearchParams(window.location.search)
    const mode = params.get('mode')

    if (mode === 'vscode') {
      uiStore.setLayoutMode('vscode')
      return
    }
    if (mode === 'mobile') {
      uiStore.setLayoutMode('mobile')
      return
    }
    if (mode === 'browser') {
      uiStore.setLayoutMode('browser')
      return
    }

    if (window.innerWidth < 768) {
      uiStore.setLayoutMode('mobile')
      return
    }

    uiStore.setLayoutMode('browser')
  }

  return {
    isVscodeMode,
    isBrowserMode,
    isMobileMode,
    layoutMode,
    detectMode,
  }
}