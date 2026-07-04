<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useSessionStore } from '@/stores/session'
import { useUiStore } from '@/stores/ui'
import { useSession } from '@/composables/useSession'
import { formatDateTime } from '@/utils/formatters'
import { STATUS_LABELS } from '@/utils/constants'
import type { TokenUsage } from '@/types/session'

const sessionStore = useSessionStore()
const uiStore = useUiStore()
const { createAndSwitch, switchTo } = useSession()

const searchQuery = ref('')
const searchInput = ref<HTMLInputElement | null>(null)
const selectedIndex = ref(0)

function formatTokenUsage(usage?: TokenUsage): string {
  if (!usage || (usage.input === 0 && usage.output === 0)) return '0 token'
  const total = usage.input + usage.output
  if (total >= 1000) return `${(total / 1000).toFixed(1)}k token`
  return `${total} token`
}

const isOpen = computed(() => uiStore.activeModal === 'session-picker')
const filteredSessions = computed(() => {
  if (!searchQuery.value) return sessionStore.sessions
  const q = searchQuery.value.toLowerCase()
  return sessionStore.sessions.filter(
    s => s.title.toLowerCase().includes(q) || s.id.toLowerCase().includes(q)
  )
})

onMounted(async () => {
  await sessionStore.fetchSessions(sessionStore.workingDirectory)
})

watch(searchQuery, () => {
  selectedIndex.value = 0
})

watch(() => uiStore.activeModal, (val) => {
  if (val === 'session-picker') {
    selectedIndex.value = 0
  }
})

async function handleSelect(sessionId: string) {
  await switchTo(sessionId)
  uiStore.setActiveModal(null)
}

async function handleCreate() {
  const name = searchQuery.value.trim() || undefined
  await createAndSwitch(name)
  uiStore.setActiveModal(null)
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    e.stopPropagation()
    uiStore.setActiveModal(null)
    return
  }
  if (e.key === 'ArrowDown') {
    e.preventDefault()
    selectedIndex.value = Math.min(selectedIndex.value + 1, filteredSessions.value.length - 1)
    return
  }
  if (e.key === 'ArrowUp') {
    e.preventDefault()
    selectedIndex.value = Math.max(selectedIndex.value - 1, 0)
    return
  }
  if (e.key === 'Enter') {
    e.preventDefault()
    const session = filteredSessions.value[selectedIndex.value]
    if (session) {
      handleSelect(session.id)
    }
    return
  }
}
</script>

<template>
  <div v-if="isOpen" class="modal-overlay" @click="uiStore.setActiveModal(null)" @keydown="handleKeydown">
    <div class="session-picker" @click.stop>
      <div class="picker-header">
        <h3>会话列表</h3>
        <button class="btn-close" @click="uiStore.setActiveModal(null)">✕</button>
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
            <span class="item-meta">
              {{ session.messageCount }} 条消息 · {{ formatTokenUsage(session.tokenUsage) }} · {{ formatDateTime(session.createdAt) }}
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

.session-picker {
  width: 480px;
  max-height: 60vh;
  background: var(--color-bg-primary);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-modal);
  animation: modalIn var(--transition-base) ease;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.picker-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-lg);
  border-bottom: 1px solid var(--color-border-light);
}

.picker-header h3 {
  font-size: var(--font-size-lg);
  font-weight: 600;
}

.btn-close {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-md);
  color: var(--color-text-secondary);
  font-size: var(--font-size-base);
}

.btn-close:hover {
  background: var(--color-bg-hover);
  color: var(--color-text-primary);
}

.picker-search {
  padding: var(--space-md) var(--space-lg);
}

.search-input {
  width: 100%;
  padding: var(--space-sm) var(--space-md);
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-md);
  font-size: var(--font-size-base);
  color: var(--color-text-primary);
}

.search-input:focus {
  border-color: var(--color-accent);
  box-shadow: 0 0 0 3px var(--color-accent-light);
}

.picker-list {
  flex: 1;
  overflow-y: auto;
  padding: var(--space-xs) 0;
}

.picker-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-sm) var(--space-lg);
  cursor: pointer;
  transition: background var(--transition-fast);
}

.picker-item:hover {
  background: var(--color-bg-hover);
}

.picker-item.active {
  background: var(--color-accent-light);
  border-left: 3px solid var(--color-accent);
}

.picker-item.selected:not(.active) {
  background: var(--color-bg-secondary);
  border-left: 3px solid var(--color-border);
}

.item-left {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.item-name {
  font-size: var(--font-size-base);
  font-weight: 500;
  color: var(--color-text-primary);
}

.item-meta {
  font-size: var(--font-size-xs);
  color: var(--color-text-tertiary);
}

.item-status {
  font-size: var(--font-size-xs);
  padding: 2px 8px;
  border-radius: var(--radius-full);
  background: var(--color-bg-tertiary);
  color: var(--color-text-secondary);
}

.item-status.status-idle,
.item-status.status-completed {
  color: var(--color-success);
}

.item-status.status-processing {
  color: var(--color-accent);
}

.item-status.status-awaiting_approval {
  color: var(--color-warning);
}

.picker-empty {
  padding: var(--space-2xl);
  text-align: center;
  font-size: var(--font-size-sm);
  color: var(--color-text-tertiary);
}

.picker-footer {
  padding: var(--space-md) var(--space-lg);
  border-top: 1px solid var(--color-border-light);
}

.btn-new {
  width: 100%;
  padding: var(--space-sm);
  border: 1px dashed var(--color-border);
  border-radius: var(--radius-md);
  font-size: var(--font-size-sm);
  color: var(--color-accent);
  transition: all var(--transition-fast);
}

.btn-new:hover {
  background: var(--color-accent-light);
  border-color: var(--color-accent);
}
</style>