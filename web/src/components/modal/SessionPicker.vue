<script setup lang="ts">
import AppIcon from '@/components/common/AppIcon.vue'
import { useSessionPicker } from './SessionPickerController'

const {
  sessionStore,
  uiStore,
  searchQuery,
  searchInput,
  selectedIndex,
  formatTokenUsage,
  truncateText,
  formatLastMessageTime,
  isOpen,
  filteredSessions,
  handleSelect,
  handleCreate,
  handleKeydown,
  STATUS_LABELS,
  formatDateTime,
} = useSessionPicker()
</script>

<template>
  <div v-if="isOpen" class="modal-overlay" @click="uiStore.setActiveModal(null)" @keydown="handleKeydown">
    <div class="session-picker" @click.stop>
      <div class="picker-header">
        <h3>会话列表</h3>
        <button class="btn-close" @click="uiStore.setActiveModal(null)"><AppIcon name="x" :size="16" /></button>
      </div>

      <div class="picker-search">
        <input
          ref="searchInput"
          v-model="searchQuery"
          type="text"
          class="search-input"
          placeholder="搜索会话..."
        />
      </div>

      <div class="picker-list">
        <div
          v-for="(session, index) in filteredSessions"
          :key="session.id"
          class="picker-item"
          :class="{ active: session.id === sessionStore.currentSession?.id, selected: index === selectedIndex }"
          @click="handleSelect(session.id)"
        >
          <div class="item-left">
            <span class="item-name">{{ session.title }}</span>
            <span v-if="session.lastMessageContent" class="item-last-msg">
              {{ truncateText(session.lastMessageContent, 60) }}
            </span>
            <span class="item-meta">
              {{ session.messageCount }} 条消息 · {{ formatTokenUsage(session.tokenUsage) }}
              <span v-if="session.lastMessageTime"> · {{ formatLastMessageTime(session.lastMessageTime) }}</span>
              <span v-else> · {{ formatDateTime(session.createdAt) }}</span>
            </span>
          </div>
          <div class="item-right">
            <span class="item-status" :class="`status-${session.state}`">
              {{ STATUS_LABELS[session.state] ?? session.state }}
            </span>
          </div>
        </div>

        <div v-if="filteredSessions.length === 0" class="picker-empty">
          {{ searchQuery ? '无匹配会话' : '暂无会话' }}
        </div>
      </div>

      <div class="picker-footer">
        <button class="btn-new" @click="handleCreate">+ 新建会话</button>
      </div>
    </div>
  </div>
</template>

<style scoped src="./SessionPicker.css">
</style>