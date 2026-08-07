import { computed, ref } from 'vue'
import { useUiStore } from '@/stores/ui'
import { API_BASE } from '@/utils/constants'

export function useMobileWorkspacePicker(emit: (e: string, ...args: any[]) => void) {
  const uiStore = useUiStore()

const workspaces = computed(() => uiStore.workspaceList)
const activeWorkspace = computed(() => uiStore.activeWorkspace)

function selectWorkspace(ws: { id: string; path: string }) {
  uiStore.setActiveWorkspace(ws.id)
  fetch(`${API_BASE}/current-workspace`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ working_directory: ws.path }),
  }).catch(() => {})
  emit('switched', ws.path)
  emit('close')
}

function onBackdropClick() {
  emit('close')
}

function onSheetClick(e: Event) {
  e.stopPropagation()
}

let startY = 0

function onTouchStart(e: TouchEvent) {
  startY = e.touches[0].clientY
}

function onTouchMove(e: TouchEvent) {
  const delta = e.touches[0].clientY - startY
  if (delta > 80) {
    emit('close')
  }
}

  return {
    uiStore,
    workspaces,
    activeWorkspace,
    selectWorkspace,
    onBackdropClick,
    onSheetClick,
    onTouchStart,
    onTouchMove,
  }
}