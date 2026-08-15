import { computed, ref } from 'vue'
import { useUiStore } from '@/stores/ui'

export function useConfigWarningDialog() {
  const uiStore = useUiStore()

  const isOpen = computed(() => uiStore.activeModal === 'config-warning')
  const showAddForm = ref(false)

  function close() {
    uiStore.setActiveModal(null)
    showAddForm.value = false
  }

  return { isOpen, close, showAddForm }
}