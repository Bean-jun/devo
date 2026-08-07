<script setup lang="ts">
import AppIcon from '@/components/common/AppIcon.vue'
import type { Message } from '@/types/message'
import { useMessageList } from './MessageListController'

const {
  chatStore,
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
  scrollToBottom,
  showScrollToBottom,
  isNearBottom,
  scrollToMessage,
  getItemKey,
  getItem,
  updateItemHeight,
  onScroll,
} = useMessageList()

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
      <AppIcon name="arrow-down" :size="14" /> 回到底部
    </button>
  </div>
</template>

<style scoped src="./MessageList.css">
</style>