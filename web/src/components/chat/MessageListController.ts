import { computed, watch, nextTick, ref } from 'vue'
import { useChatStore } from '@/stores/chat'
import { useSessionStore } from '@/stores/session'
import { useUiStore } from '@/stores/ui'
import { useVirtualScroll } from '@/composables/useVirtualScroll'
import type { Message } from '@/types/message'
import MessageBubble from './MessageBubble.vue'
import ToolCallCard from './ToolCallCard.vue'
import ToolCallGroup from './ToolCallGroup.vue'
import ThinkingIndicator from './ThinkingIndicator.vue'
import VirtualMessageItem from './VirtualMessageItem.vue'

export function useMessageList() {
  const chatStore = useChatStore()
  const sessionStore = useSessionStore()
  const uiStore = useUiStore()

const yoloMode = computed(() => sessionStore.yoloEnabled)

const allMessages = computed(() => chatStore.messages)

const visibleMessages = computed(() =>
  allMessages.value.filter((msg) => {
    if (msg.role === 'assistant' && !msg.content.trim()) return false
    return true
  })
)

const groupedMessages = computed(() => {
  const result: (Message | Message[])[] = []
  let toolGroup: Message[] = []

  for (const msg of visibleMessages.value) {
    if (msg.role === 'tool' && msg.toolCall) {
      toolGroup.push(msg)
    } else {
      if (toolGroup.length > 0) {
        if (toolGroup.length === 1) {
          result.push(toolGroup[0])
        } else {
          result.push([...toolGroup])
        }
        toolGroup = []
      }
      result.push(msg)
    }
  }

  if (toolGroup.length > 0) {
    if (toolGroup.length === 1) {
      result.push(toolGroup[0])
    } else {
      result.push([...toolGroup])
    }
  }

  return result
})

const STREAMING_SENTINEL = Symbol('streaming')

const isStreaming = computed(() => chatStore.isStreaming)

const displayItems = computed(() => {
  const items: (Message | Message[] | typeof STREAMING_SENTINEL)[] = [...groupedMessages.value]
  if (isStreaming.value) {
    items.push(STREAMING_SENTINEL)
  }
  return items
})

const itemCount = computed(() => displayItems.value.length)

const {
  containerRef,
  visibleRange,
  totalHeight,
  offsetY,
  scrollToIndex,
  scrollToBottom: scrollContainerToBottom,
  updateItemHeight,
} = useVirtualScroll(itemCount, { estimateHeight: 200, bufferSize: 5 })

const showScrollToBottom = ref(false)
const isNearBottom = ref(true)

function checkNearBottom(): void {
  const el = containerRef.value
  if (!el) return
  isNearBottom.value = el.scrollHeight - el.scrollTop - el.clientHeight < 100
  showScrollToBottom.value = !isNearBottom.value
}

function scrollToBottom(smooth = true): void {
  scrollContainerToBottom(smooth)
  showScrollToBottom.value = false
  isNearBottom.value = true
}

function scrollToMessage(messageId: string): void {
  const idx = displayItems.value.findIndex((item) => {
    if (item === STREAMING_SENTINEL) return false
    if (Array.isArray(item)) return item[0].id === messageId
    return item.id === messageId
  })
  if (idx >= 0) {
    scrollToIndex(idx)
  }
}

function getItemKey(index: number): string {
  const item = displayItems.value[index]
  if (!item) return `empty-${index}`
  if (item === STREAMING_SENTINEL) return 'streaming-indicator'
  if (Array.isArray(item)) return item[0].id
  return item.id
}

function getItem(index: number): Message | Message[] | typeof STREAMING_SENTINEL | undefined {
  return displayItems.value[index]
}

const lastScrollLength = ref(0)

watch(
  () => chatStore.messages.length,
  (newLen) => {
    if (newLen !== lastScrollLength.value) {
      lastScrollLength.value = newLen
      nextTick(() => {
        requestAnimationFrame(() => {
          scrollToBottom(false)
        })
      })
    }
  },
  { immediate: false }
)

let scrollThrottleTimer: ReturnType<typeof setTimeout> | null = null

watch(
  () => chatStore.streamingContent,
  () => {
    if (scrollThrottleTimer) return
    scrollThrottleTimer = setTimeout(() => {
      scrollThrottleTimer = null
      nextTick(() => {
        requestAnimationFrame(() => {
          if (isNearBottom.value) {
            scrollToBottom(false)
          }
        })
      })
    }, 100)
  },
  { immediate: false }
)

let didInitialScroll = false

watch(
  itemCount,
  (newCount) => {
    if (newCount > 0 && !didInitialScroll) {
      didInitialScroll = true
      nextTick(() => {
        scrollToBottom(false)
      })
    }
  }
)

function onScroll(): void {
  checkNearBottom()
}

  return {
    chatStore,
    sessionStore,
    uiStore,
    MessageBubble,
    ToolCallCard,
    ToolCallGroup,
    ThinkingIndicator,
    VirtualMessageItem,
    yoloMode,
    allMessages,
    groupedMessages,
    isStreaming,
    displayItems,
    STREAMING_SENTINEL,
    itemCount,
    containerRef,
    visibleRange,
    totalHeight,
    offsetY,
    scrollToIndex,
    scrollToBottom,
    showScrollToBottom,
    isNearBottom,
    scrollToMessage,
    getItemKey,
    getItem,
    updateItemHeight,
    onScroll,
  }
}