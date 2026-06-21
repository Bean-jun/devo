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
      <div class="empty-icon">🤖</div>
      <h2>欢迎使用 Devo</h2>
      <p>AI 编码助手，帮你写代码、调 Bug、管理项目</p>
      <div class="empty-hints">
        <p>输入 <kbd>/help</kbd> 查看可用命令</p>
        <p>直接输入需求开始对话</p>
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

.empty-icon {
  font-size: 48px;
  margin-bottom: var(--space-lg);
}

.empty-state h2 {
  font-size: var(--font-size-xl);
  font-weight: 600;
  color: var(--color-text-primary);
  margin-bottom: var(--space-sm);
}

.empty-state p {
  color: var(--color-text-secondary);
  font-size: var(--font-size-base);
}

.empty-hints {
  margin-top: var(--space-xl);
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