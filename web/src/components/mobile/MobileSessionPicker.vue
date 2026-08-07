<script setup lang="ts">
import AppIcon from '@/components/common/AppIcon.vue'
import { useMobileSessionPicker } from './MobileSessionPickerController'

const emit = defineEmits<{
  close: []
  newSession: []
}>()

const {
  sessions,
  currentSessionId,
  truncateText,
  formatLastMessageTime,
  selectSession,
  newSession,
  onBackdropClick,
  onSheetClick,
  onTouchStart,
  onTouchMove,
} = useMobileSessionPicker(emit as any)
</script>

<template>
  <Teleport to="body">
    <div class="picker-overlay" @click.self="onBackdropClick">
      <div
        class="picker-sheet"
        data-test="session-picker"
        @click="onSheetClick"
        @touchstart.passive="onTouchStart"
        @touchmove.passive="onTouchMove"
      >
        <div class="sheet-drag-handle">
          <div class="drag-bar"></div>
        </div>
        <div class="picker-header">
          <span class="picker-title">切换会话</span>
          <button class="new-session-btn" @click="newSession">
            <AppIcon name="plus" :size="16" />
            <span>新建</span>
          </button>
        </div>
        <div class="picker-list">
          <button
            v-for="sess in sessions"
            :key="sess.id"
            class="picker-item"
            :class="{ active: currentSessionId === sess.id }"
            @click="selectSession(sess.id)"
          >
            <AppIcon name="chat-dots" :size="18" />
            <div class="picker-item-content">
              <span class="picker-item-name">{{ sess.title }}</span>
              <span v-if="sess.lastMessageContent" class="picker-item-last-msg">
                {{ truncateText(sess.lastMessageContent, 50) }}
              </span>
              <span class="picker-item-meta">
                {{ sess.messageCount }} 条消息
                <span v-if="sess.lastMessageTime"> · {{ formatLastMessageTime(sess.lastMessageTime) }}</span>
              </span>
            </div>
            <AppIcon v-if="currentSessionId === sess.id" name="circle" :size="10" weight="fill" class="picker-check" />
          </button>
          <div v-if="sessions.length === 0" class="picker-empty">
            暂无会话
          </div>
        </div>
        <button class="picker-cancel" @click="emit('close')">取消</button>
      </div>
    </div>
  </Teleport>
</template>

<style scoped src="./MobileSessionPicker.css">
</style>