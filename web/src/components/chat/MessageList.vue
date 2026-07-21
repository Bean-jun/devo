<script setup lang="ts">
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

const chatStore = useChatStore()
const sessionStore = useSessionStore()
const uiStore = useUiStore()

const yoloMode = computed(() => sessionStore.yoloEnabled)

const allMessages = computed(() => chatStore.messages)

const visibleMessages = computed(() =>
  allMessages.value.filter((msg) => {
    if (msg.role === 'assistant' && !msg.content) return false
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

defineExpose({
  scrollToBottom,
  scrollToMessage,
})
</script>

<template>
  <div
    ref="containerRef"
    class="message-list-container"
    @scroll="onScroll"
    @click.self="uiStore.requestFocusInput()"
  >
    <div v-if="allMessages.length === 0 && !isStreaming && chatStore.initialFetchDone" class="empty-state">
      <pre class="ascii-banner">
██████╗ ███████╗██╗   ██╗ ██████╗ 
██╔══██╗██╔════╝██║   ██║██╔═══██╗
██║  ██║█████╗  ██║   ██║██║   ██║
██║  ██║██╔══╝  ╚██╗ ██╔╝██║   ██║
██████╔╝███████╗ ╚████╔╝ ╚██████╔╝
╚═════╝ ╚══════╝  ╚═══╝   ╚═════╝ 
      </pre>
      <div class="empty-subtitle">AI 编码助手 — 写代码 · 调 Bug · 管理项目</div>
      <div class="empty-hints">
        <p><kbd>/help</kbd> 查看命令 • 直接输入需求开始对话</p>
      </div>
    </div>

    <div
      v-else
      class="message-list-viewport"
      :style="{ height: totalHeight + 'px' }"
    >
      <div class="message-list-content" :style="{ paddingTop: offsetY + 'px' }">
        <template
          v-for="idx in visibleRange.end - visibleRange.start + 1"
          :key="getItemKey(visibleRange.start + idx - 1)"
        >
          <VirtualMessageItem
            v-if="getItem(visibleRange.start + idx - 1) !== undefined"
            :index="visibleRange.start + idx - 1"
            :on-height-change="updateItemHeight"
          >
            <template v-if="getItem(visibleRange.start + idx - 1) === STREAMING_SENTINEL">
              <ThinkingIndicator />
            </template>
            <template
              v-else-if="Array.isArray(getItem(visibleRange.start + idx - 1))"
            >
              <ToolCallGroup
                :messages="(getItem(visibleRange.start + idx - 1) as Message[])"
                :yolo-mode="yoloMode"
                :data-message-id="(getItem(visibleRange.start + idx - 1) as Message[])[0].id"
                data-role="tool-group"
              />
            </template>
            <template
              v-else-if="(getItem(visibleRange.start + idx - 1) as Message).role === 'tool' && (getItem(visibleRange.start + idx - 1) as Message).toolCall"
            >
              <ToolCallCard
                :tool-call="(getItem(visibleRange.start + idx - 1) as Message).toolCall!"
                :yolo-mode="yoloMode"
                :data-message-id="(getItem(visibleRange.start + idx - 1) as Message).id"
                data-role="tool"
              />
            </template>
            <template v-else>
              <MessageBubble
                :message="(getItem(visibleRange.start + idx - 1) as Message)"
                v-memo="[(getItem(visibleRange.start + idx - 1) as Message).content, (getItem(visibleRange.start + idx - 1) as Message).role]"
                :data-message-id="(getItem(visibleRange.start + idx - 1) as Message).id"
                :data-role="(getItem(visibleRange.start + idx - 1) as Message).role"
              />
            </template>
          </VirtualMessageItem>
        </template>
      </div>
    </div>

    <button
      v-if="showScrollToBottom"
      class="scroll-to-bottom"
      @click="scrollToBottom(true)"
    >
      ↓ 回到底部
    </button>
  </div>
</template>

<style scoped>
.message-list-container {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  position: relative;
  padding: var(--space-lg) 0;
}

.message-list-viewport {
  position: relative;
  width: 100%;
}

.message-list-content {
  max-width: 800px;
  margin: 0 auto;
  padding-left: var(--space-lg);
  padding-right: var(--space-lg);
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: var(--space-2xl) var(--space-lg);
  text-align: center;
  animation: fadeIn var(--transition-slow) ease;
}

.ascii-banner {
  font-family: var(--font-mono);
  font-size: 10px;
  line-height: 1.15;
  color: var(--color-accent);
  margin: 0 0 var(--space-lg) 0;
  white-space: pre;
  user-select: none;
  opacity: 0.85;
}

.empty-subtitle {
  font-size: var(--font-size-base);
  font-weight: 500;
  color: var(--color-text-secondary);
  margin-bottom: var(--space-lg);
}

.empty-hints {
  display: flex;
  flex-direction: column;
  gap: var(--space-sm);
}

.empty-hints p {
  font-size: var(--font-size-sm);
  color: var(--color-text-tertiary);
}

kbd {
  display: inline-block;
  padding: 1px 6px;
  font-family: var(--font-mono);
  font-size: var(--font-size-xs);
  color: var(--color-text-secondary);
  background: var(--color-bg-tertiary);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
}

.scroll-to-bottom {
  position: sticky;
  bottom: var(--space-md);
  left: 50%;
  transform: translateX(-50%);
  padding: var(--space-xs) var(--space-md);
  background: var(--color-accent);
  color: var(--color-text-inverse);
  border-radius: var(--radius-full);
  font-size: var(--font-size-xs);
  box-shadow: var(--shadow-md);
  z-index: 10;
  animation: slideInUp var(--transition-fast) ease;
}

.scroll-to-bottom:hover {
  background: var(--color-accent-hover);
}
</style>