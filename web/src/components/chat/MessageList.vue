<script setup lang="ts">
import { computed, watch, nextTick } from 'vue'
import { useChatStore } from '@/stores/chat'
import type { Message } from '@/types/message'
import MessageBubble from './MessageBubble.vue'
import ToolCallCard from './ToolCallCard.vue'
import ToolCallGroup from './ToolCallGroup.vue'
import ThinkingIndicator from './ThinkingIndicator.vue'

const props = defineProps<{
  scrollToBottom: (smooth?: boolean) => void
}>()

const chatStore = useChatStore()

const allMessages = computed(() => chatStore.messages)

const visibleMessages = computed(() =>
  allMessages.value.filter((msg) => {
    if (msg.role === 'assistant' && !msg.content) return false
    return true
  })
)

/**
 * 将连续的 tool 消息分组，非 tool 消息保持独立
 * 返回 (Message | Message[])[] 数组
 */
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

  // 处理末尾的 tool 组
  if (toolGroup.length > 0) {
    if (toolGroup.length === 1) {
      result.push(toolGroup[0])
    } else {
      result.push([...toolGroup])
    }
  }

  return result
})

watch(
  () => [chatStore.messages.length, chatStore.streamingContent],
  () => {
    nextTick(() => props.scrollToBottom(false))
  },
  { deep: false }
)
</script>

<template>
  <div class="message-list">
    <div v-if="allMessages.length === 0 && !chatStore.isStreaming" class="empty-state">
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

    <template v-for="item in groupedMessages" :key="Array.isArray(item) ? item[0].id : item.id">
      <ToolCallGroup v-if="Array.isArray(item)" :messages="item" />
      <MessageBubble v-else-if="item.role !== 'tool'" :message="item" />
      <ToolCallCard v-else-if="item.toolCall" :tool-call="item.toolCall" />
    </template>

    <ThinkingIndicator v-if="chatStore.isStreaming" />
  </div>
</template>

<style scoped>
.message-list {
  max-width: 800px;
  margin: 0 auto;
  padding: 0 var(--space-lg);
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
</style>