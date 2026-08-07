import { computed } from 'vue'
import { useUiStore } from '@/stores/ui'

export function useConfigWarningDialog() {
  const uiStore = useUiStore()

  const isOpen = computed(() => uiStore.activeModal === 'config-warning')

  function close() {
    uiStore.setActiveModal(null)
  }

  return { uiStore, isOpen, close }
}