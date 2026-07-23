<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useChatStore } from '@/stores/chat'
import { useUiStore } from '@/stores/ui'
import { useSessionStore } from '@/stores/session'
import { formatTime } from '@/utils/formatters'
import { API_BASE } from '@/utils/constants'
import AppIcon from '@/components/common/AppIcon.vue'

const chatStore = useChatStore()
const uiStore = useUiStore()
const sessionStore = useSessionStore()

const selectedIndex = ref(0)
const isLoading = ref(false)

const isOpen = computed(() => uiStore.activeModal === 'rollback-picker')

const userMessages = computed(() => {
  const msgs = chatStore.messages
    .map((msg, originalIndex) => ({ msg, originalIndex }))
    .filter(({ msg }) => msg.role === 'user')
    .reverse()
  return msgs
})

watch(isOpen, async (val) => {
  if (val && sessionStore.currentSession) {
    isLoading.value = true
    try {
      await chatStore.fetchMessages(sessionStore.currentSession.id)
    } catch {
      uiStore.showToast('error', '加载消息失败')
    } finally {
      isLoading.value = false
    }
    if (userMessages.value.length > 0) {
      selectedIndex.value = userMessages.value[0].originalIndex
    }
  }
})

function selectMessage(entry: { msg: any; originalIndex: number }) {
  selectedIndex.value = entry.originalIndex
}

async function confirmRollback() {
  if (selectedIndex.value < 0 || !sessionStore.currentSession) return
  const targetMsg = userMessages.value.find(
    (e) => e.originalIndex === selectedIndex.value
  )
  if (!targetMsg) return

  try {
    const res = await fetch(
      `${API_BASE}/sessions/${sessionStore.currentSession.id}/rollback`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ target_message_id: targetMsg.msg.id }),
      }
    )
    if (!res.ok) throw new Error('回滚失败')
    chatStore.rollbackTo(selectedIndex.value)
    uiStore.setPendingCommand(targetMsg.msg.content)
    uiStore.showToast('info', '消息已回滚')
    uiStore.setActiveModal(null)
  } catch (err) {
    uiStore.showToast('error', `回滚失败: ${(err as Error).message}`)
  }
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    e.stopPropagation()
    uiStore.setActiveModal(null)
    return
  }
  if (e.key === 'ArrowDown') {
    e.preventDefault()
    const next = userMessages.value.findIndex(
      (entry) => entry.originalIndex === selectedIndex.value
    )
    if (next >= 0 && next < userMessages.value.length - 1) {
      selectedIndex.value = userMessages.value[next + 1].originalIndex
    }
    return
  }
  if (e.key === 'ArrowUp') {
    e.preventDefault()
    const curr = userMessages.value.findIndex(
      (entry) => entry.originalIndex === selectedIndex.value
    )
    if (curr > 0) {
      selectedIndex.value = userMessages.value[curr - 1].originalIndex
    }
    return
  }
  if (e.key === 'Enter') {
    e.preventDefault()
    confirmRollback()
    return
  }
}
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

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  z-index: 6000;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-overlay);
  animation: fadeIn var(--transition-fast) ease;
}

.rollback-picker {
  width: 500px;
  max-height: 70vh;
  background: var(--color-bg-primary);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-modal);
  animation: modalIn var(--transition-base) ease;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.picker-header {
  padding: var(--space-lg);
  border-bottom: 1px solid var(--color-border-light);
}

.picker-header h3 {
  font-size: var(--font-size-lg);
  font-weight: 600;
}

.picker-hint {
  font-size: var(--font-size-xs);
  color: var(--color-text-tertiary);
}

.timeline {
  flex: 1;
  overflow-y: auto;
  padding: var(--space-lg);
}

.timeline-empty {
  text-align: center;
  padding: var(--space-2xl);
  color: var(--color-text-tertiary);
  font-size: var(--font-size-sm);
}

.timeline-item {
  display: flex;
  gap: var(--space-md);
  padding: var(--space-sm) 0;
  cursor: pointer;
  transition: opacity var(--transition-fast);
}

.timeline-item:hover {
  opacity: 0.8;
}

.timeline-item.selected .marker-dot {
  background: var(--color-accent);
  transform: scale(1.5);
  box-shadow: 0 0 0 4px var(--color-accent-light);
}

.timeline-item.after-selected {
  opacity: 0.3;
  text-decoration: line-through;
}

.timeline-marker {
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 20px;
  flex-shrink: 0;
}

.marker-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: var(--color-border);
  border: 2px solid var(--color-bg-primary);
  transition: all var(--transition-fast);
}

.marker-line {
  width: 2px;
  flex: 1;
  background: var(--color-border-light);
  min-height: 24px;
}

.timeline-content {
  flex: 1;
  min-width: 0;
}

.timeline-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 2px;
}

.timeline-role {
  font-size: var(--font-size-xs);
  font-weight: 600;
  color: var(--color-text-secondary);
}

.timeline-time {
  font-size: 10px;
  color: var(--color-text-tertiary);
  font-family: var(--font-mono);
}

.timeline-preview {
  font-size: var(--font-size-sm);
  color: var(--color-text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.picker-footer {
  padding: var(--space-lg);
  border-top: 1px solid var(--color-border-light);
}

.rollback-warning {
  font-size: var(--font-size-xs);
  color: var(--color-error);
  margin-bottom: var(--space-md);
  padding: var(--space-sm);
  background: var(--color-error-light);
  border-radius: var(--radius-sm);
}

.picker-actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-md);
}

.btn-cancel,
.btn-confirm {
  padding: var(--space-sm) var(--space-xl);
  border-radius: var(--radius-md);
  font-size: var(--font-size-base);
  font-weight: 500;
  transition: all var(--transition-fast);
}

.btn-cancel {
  background: var(--color-bg-secondary);
  color: var(--color-text-primary);
}

.btn-cancel:hover {
  background: var(--color-bg-tertiary);
}

.btn-confirm {
  background: var(--color-error);
  color: white;
}

.btn-confirm:hover:not(:disabled) {
  background: #e0352b;
}

.btn-confirm:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
</style>