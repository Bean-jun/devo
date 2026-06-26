import { computed } from 'vue'
import { useUiStore } from '@/stores/ui'

export function usePlatform() {
  const uiStore = useUiStore()

  const isVscodeMode = computed(() => uiStore.isVscodeMode)

  const isBrowserMode = computed(() => !uiStore.isVscodeMode)

  function detectMode(): void {
    const params = new URLSearchParams(window.location.search)
    const mode = params.get('mode')
    uiStore.setVscodeMode(mode === 'vscode')
  }

  return {
    isVscodeMode,
    isBrowserMode,
    detectMode,
  }
}