import { computed } from 'vue'
import { useUiStore } from '@/stores/ui'

export type LayoutMode = 'browser' | 'mobile'

export function usePlatform() {
  const uiStore = useUiStore()

  const isBrowserMode = computed(() => uiStore.layoutMode === 'browser')

  const isMobileMode = computed(() => uiStore.layoutMode === 'mobile')

  const layoutMode = computed(() => uiStore.layoutMode)

  function detectMode(): void {
    const params = new URLSearchParams(window.location.search)
    const mode = params.get('mode')

    if (mode === 'browser') {
      uiStore.setLayoutMode('browser')
      return
    }

    // 'vscode' is treated as a mobile alias (VSCode extension uses ?mode=vscode)
    if (mode === 'vscode' || mode === 'mobile' || window.innerWidth < 768) {
      uiStore.setLayoutMode('mobile')
      return
    }

    uiStore.setLayoutMode('browser')
  }

  return {
    isBrowserMode,
    isMobileMode,
    layoutMode,
    detectMode,
  }
}