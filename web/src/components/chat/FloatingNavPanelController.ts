import { computed, ref, watch } from 'vue'
import { useChatStore } from '@/stores/chat'

export interface FloatingNavPanelProps {
  scrollToMessage: (messageId: string) => void
}

export function useFloatingNavPanel(props: FloatingNavPanelProps) {
  const chatStore = useChatStore()

const navItems = computed(() => {
  return chatStore.messages
    .filter((msg) => msg.role === 'user')
    .map((msg) => ({
      id: msg.id,
      summary: msg.content.length > 20 ? msg.content.slice(0, 20) + '…' : msg.content,
    }))
})

const activeId = ref<string | null>(null)

watch(
  () => navItems.value.length,
  (len) => {
    if (len > 0) {
      activeId.value = navItems.value[len - 1].id
    }
  },
  { immediate: true }
)

function handleClick(itemId: string): void {
  activeId.value = itemId
  props.scrollToMessage(itemId)
}

  return {
    navItems,
    activeId,
    handleClick,
  }
}