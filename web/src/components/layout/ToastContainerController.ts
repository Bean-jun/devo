import { useUiStore } from '@/stores/ui'
import ToastItem from './ToastItem.vue'

export function useToastContainer() {
  const uiStore = useUiStore()
  return { uiStore, ToastItem }
}