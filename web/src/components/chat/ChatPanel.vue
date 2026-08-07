<script setup lang="ts">
import AppIcon from '@/components/common/AppIcon.vue'
import MessageList from './MessageList.vue'
import InputArea from './InputArea.vue'
import FloatingNavPanel from './FloatingNavPanel.vue'
import { useChatPanel } from './ChatPanelController'

const props = withDefaults(defineProps<{
  hideInput?: boolean
}>(), {
  hideInput: false,
})

const {
  sessionStore,
  chatStore,
  messageListRef,
  isDisabled,
  isProcessing,
  handleSend,
  handleStop,
  handleClear,
  handleOpenCommand,
  handleExecuteCommand,
  handleScrollToMessage,
} = useChatPanel()
</script>

<template>
  <div v-if="sessionStore.currentSession" class="chat-panel" data-test="chat-panel">
    <div class="chat-body">
      <MessageList ref="messageListRef" />
      <FloatingNavPanel
        v-if="chatStore.messages.length > 0"
        :scroll-to-message="handleScrollToMessage"
      />
    </div>

    <InputArea
      v-if="!props.hideInput"
      :is-disabled="isDisabled"
      :is-processing="isProcessing"
      @send="handleSend"
      @stop="handleStop"
      @clear="handleClear"
      @open-command="handleOpenCommand"
      @execute-command="handleExecuteCommand"
    />
  </div>
  <div v-else class="chat-empty">
    <AppIcon name="chat-dots" :size="48" class="chat-empty-icon" />
    <div class="chat-empty-title">请选择或新建一个会话</div>
    <div class="chat-empty-desc">在左侧选择一个已有会话，或点击 + 新建会话开始对话</div>
  </div>
</template>

<style scoped src="./ChatPanel.css">
</style>