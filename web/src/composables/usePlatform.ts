import { computed, ref, onMounted, onUnmounted } from 'vue'
import { useUiStore } from '@/stores/ui'

export type LayoutMode = 'browser' | 'mobile'
export type DensityLevel = 'compact' | 'tablet' | 'full'

const densityLevel = ref<DensityLevel>('full')
let resizeTimer: ReturnType<typeof setTimeout> | null = null
let listenerCount = 0

function updateDensity(): void {
  const w = window.innerWidth
  if (w < 500) {
    densityLevel.value = 'compact'
  } else if (w < 900) {
    densityLevel.value = 'tablet'
  } else {
    densityLevel.value = 'full'
  }
}

function onResize(): void {
  if (resizeTimer) clearTimeout(resizeTimer)
  resizeTimer = setTimeout(updateDensity, 150)
}

function addResizeListener(): void {
  if (listenerCount === 0) {
    window.addEventListener('resize', onResize)
  }
  listenerCount++
}

function removeResizeListener(): void {
  listenerCount--
  if (listenerCount <= 0) {
    listenerCount = 0
    window.removeEventListener('resize', onResize)
    if (resizeTimer) {
      clearTimeout(resizeTimer)
      resizeTimer = null
    }
  }
}

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
      updateDensity()
      return
    }

    // 'vscode' is treated as a mobile alias (VSCode extension uses ?mode=vscode)
    if (mode === 'vscode' || mode === 'mobile' || window.innerWidth < 768) {
      uiStore.setLayoutMode('mobile')
      updateDensity()
      return
    }

    uiStore.setLayoutMode('browser')
    updateDensity()
  }

  onMounted(() => {
    addResizeListener()
  })

  onUnmounted(() => {
    removeResizeListener()
  })

  return {
    isBrowserMode,
    isMobileMode,
    layoutMode,
    densityLevel,
    detectMode,
  }
}