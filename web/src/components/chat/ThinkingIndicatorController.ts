import { computed, ref } from 'vue'
import { useChatStore } from '@/stores/chat'

export function useThinkingIndicator() {
  const chatStore = useChatStore()

  const hasContent = computed(() => chatStore.streamingContent.length > 0)
  const hasReasoning = computed(() => chatStore.streamingReasoning.length > 0)
  const isReasoningActive = computed(() => chatStore.isReasoningActive)

  const reasoningExpanded = ref(false)

  function toggleReasoning() {
    reasoningExpanded.value = !reasoningExpanded.value
  }

  return { chatStore, hasContent, hasReasoning, isReasoningActive, reasoningExpanded, toggleReasoning }
}