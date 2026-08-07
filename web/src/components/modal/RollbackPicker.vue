<script setup lang="ts">
import AppIcon from '@/components/common/AppIcon.vue'
import { useRollbackPicker } from './RollbackPickerController'

const {
  chatStore,
  uiStore,
  selectedIndex,
  isLoading,
  isOpen,
  userMessages,
  selectMessage,
  confirmRollback,
  handleKeydown,
  formatTime,
} = useRollbackPicker()
</script>

<template>
  <div v-if="isOpen" class="modal-overlay" @click="uiStore.setActiveModal(null)" @keydown="handleKeydown">
    <div class="rollback-picker" @click.stop>
      <div class="picker-header">
        <h3>回滚消息</h3>
        <span class="picker-hint">选择要回滚到的消息位置</span>
      </div>

      <div class="timeline">
        <div v-if="isLoading" class="timeline-empty">
          加载中...
        </div>
        <div v-else-if="userMessages.length === 0" class="timeline-empty">
          暂无消息可回滚
        </div>
        <div
          v-for="entry in userMessages"
          :key="entry.msg.id"
          class="timeline-item"
          :class="{
            selected: entry.originalIndex === selectedIndex,
            'after-selected': selectedIndex >= 0 && entry.originalIndex > selectedIndex,
          }"
          @click="selectMessage(entry)"
        >
          <div class="timeline-marker">
            <div class="marker-dot"></div>
            <div v-if="entry.originalIndex < chatStore.messages.length - 1" class="marker-line"></div>
          </div>
          <div class="timeline-content">
            <div class="timeline-header">
              <span class="timeline-role">你</span>
              <span class="timeline-time">{{ formatTime(entry.msg.timestamp) }}</span>
            </div>
            <div class="timeline-preview">
              {{ entry.msg.content.slice(0, 80) }}{{ entry.msg.content.length > 80 ? '...' : '' }}
            </div>
          </div>
        </div>
      </div>

      <div class="picker-footer">
        <div v-if="selectedIndex >= 0" class="rollback-warning">
          <AppIcon name="warning" :size="16" class="warning-icon" /> 将删除 #{{ selectedIndex + 1 }} 之后的所有消息（共 {{ chatStore.messages.length - selectedIndex - 1 }} 条），此操作不可撤销
        </div>
        <div class="picker-actions">
          <button class="btn-cancel" @click="uiStore.setActiveModal(null)">取消</button>
          <button
            class="btn-confirm"
            :disabled="selectedIndex < 0"
            @click="confirmRollback"
          >
            确认回滚
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped src="./RollbackPicker.css">
</style>