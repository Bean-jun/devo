import { computed } from 'vue'
import { useSessionStore } from '@/stores/session'

export function useAppHeader() {
  const sessionStore = useSessionStore()

  const sessionTitle = computed(() => {
    return sessionStore.currentSession?.title || 'Devo'
  })

  return { sessionTitle }
}