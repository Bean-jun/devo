import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useUiStore } from '@/stores/ui'
import { useUpdateCheck } from '@/composables/useUpdateCheck'

export function useBrowserLayout() {
  const uiStore = useUiStore()
  const { updateInfo, checkUpdate } = useUpdateCheck()

  const sidebarWidth = ref(240)
  const sidebarCollapsed = computed(() => uiStore.sidebarCollapsed)
  const rightPanelVisible = computed(() => uiStore.rightPanelVisible)
  const collapsedWidth = 40
  const rightPanelWidth = ref(380)
  const resizing = ref<'left' | 'right' | null>(null)
  const leftWrapperRef = ref<HTMLElement | null>(null)
  const rightWrapperRef = ref<HTMLElement | null>(null)
  let rafId = 0

  function startResize(type: 'left' | 'right') {
    resizing.value = type
    document.body.style.cursor = 'col-resize'
    document.body.style.userSelect = 'none'
    if (type === 'left' && leftWrapperRef.value) {
      leftWrapperRef.value.style.willChange = 'width'
    } else if (type === 'right' && rightWrapperRef.value) {
      rightWrapperRef.value.style.willChange = 'width'
    }
  }

  function onMouseMove(e: MouseEvent) {
    if (!resizing.value) return
    cancelAnimationFrame(rafId)
    rafId = requestAnimationFrame(() => {
      if (resizing.value === 'left') {
        sidebarWidth.value = Math.max(180, Math.min(420, e.clientX))
      } else if (resizing.value === 'right') {
        rightPanelWidth.value = Math.max(220, Math.min(window.innerWidth * 0.75, window.innerWidth - e.clientX))
      }
    })
  }

  function onMouseUp() {
    if (resizing.value) {
      try { localStorage.setItem('devo-sidebar-width', String(sidebarWidth.value)) } catch {}
      try { localStorage.setItem('devo-rightpanel-width', String(rightPanelWidth.value)) } catch {}
    }
    if (leftWrapperRef.value) leftWrapperRef.value.style.willChange = ''
    if (rightWrapperRef.value) rightWrapperRef.value.style.willChange = ''
    resizing.value = null
    document.body.style.cursor = ''
    document.body.style.userSelect = ''
  }

  onMounted(() => {
    try {
      const sl = localStorage.getItem('devo-sidebar-width')
      if (sl) sidebarWidth.value = parseInt(sl, 10)
      const sr = localStorage.getItem('devo-rightpanel-width')
      if (sr) rightPanelWidth.value = parseInt(sr, 10)
    } catch {}
    document.addEventListener('mousemove', onMouseMove)
    document.addEventListener('mouseup', onMouseUp)
    checkUpdate()
  })

  onUnmounted(() => {
    document.removeEventListener('mousemove', onMouseMove)
    document.removeEventListener('mouseup', onMouseUp)
  })

  return {
    uiStore,
    sidebarWidth,
    sidebarCollapsed,
    rightPanelVisible,
    collapsedWidth,
    rightPanelWidth,
    leftWrapperRef,
    rightWrapperRef,
    startResize,
    updateInfo,
    checkUpdate,
  }
}