<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useSessionStore } from '@/stores/session'
import { useUiStore } from '@/stores/ui'
import { API_BASE } from '@/utils/constants'
import AppIcon from '@/components/common/AppIcon.vue'

const emit = defineEmits<{
  close: []
  newSession: []
}>()

const router = useRouter()
const sessionStore = useSessionStore()
const uiStore = useUiStore()

const sessions = computed(() => sessionStore.sessions)
const currentSessionId = computed(() => sessionStore.currentSession?.id)

async function selectSession(sessionId: string) {
  const ok = await sessionStore.switchSessionById(sessionId)
  if (ok && sessionStore.currentSession?.workingDirectory) {
    fetch(`${API_BASE}/current-workspace`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ working_directory: sessionStore.currentSession.workingDirectory }),
    }).catch(() => {})
  }
  router.push('/chat')
  emit('close')
}

function newSession() {
  emit('newSession')
  emit('close')
}

function onBackdropClick() {
  emit('close')
}

function onSheetClick(e: Event) {
  e.stopPropagation()
}

let startY = 0

function onTouchStart(e: TouchEvent) {
  startY = e.touches[0].clientY
}

function onTouchMove(e: TouchEvent) {
  const delta = e.touches[0].clientY - startY
  if (delta > 80) {
    emit('close')
  }
}
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
            <span class="picker-item-name">{{ sess.title }}</span>
            <span v-if="currentSessionId === sess.id" class="picker-check">●</span>
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

<style scoped>
.picker-overlay {
  position: fixed;
  inset: 0;
  z-index: 5000;
  background: var(--color-overlay);
  animation: fadeIn var(--transition-fast) ease;
  display: flex;
  align-items: flex-end;
}

.picker-sheet {
  width: 100%;
  max-height: 70vh;
  max-height: 70dvh;
  background: var(--color-bg-primary);
  border-radius: var(--radius-xl) var(--radius-xl) 0 0;
  display: flex;
  flex-direction: column;
  animation: slideUp 0.3s ease;
  padding-bottom: env(safe-area-inset-bottom, 0px);
  overflow: hidden;
}

.sheet-drag-handle {
  display: flex;
  justify-content: center;
  padding: var(--space-sm) 0;
}

.drag-bar {
  width: 36px;
  height: 4px;
  border-radius: 2px;
  background: var(--color-border);
}

.picker-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-sm) var(--space-lg) var(--space-md);
}

.picker-title {
  font-size: var(--font-size-lg);
  font-weight: 600;
  color: var(--color-text-primary);
}

.new-session-btn {
  display: flex;
  align-items: center;
  gap: var(--space-xs);
  padding: var(--space-xs) var(--space-md);
  border: 1px solid var(--color-accent);
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--color-accent);
  font-size: var(--font-size-sm);
  cursor: pointer;
  min-height: 36px;
  -webkit-tap-highlight-color: transparent;
}

.picker-list {
  flex: 1;
  overflow-y: auto;
  padding: 0 var(--space-lg);
  -webkit-overflow-scrolling: touch;
}

.picker-item {
  display: flex;
  align-items: center;
  gap: var(--space-md);
  width: 100%;
  padding: var(--space-md);
  border: none;
  border-radius: var(--radius-md);
  background: transparent;
  cursor: pointer;
  text-align: left;
  transition: background var(--transition-fast);
  min-height: 56px;
  -webkit-tap-highlight-color: transparent;
}

.picker-item:active {
  background: var(--color-bg-hover);
}

.picker-item.active {
  background: var(--color-accent-light);
}

.picker-item-name {
  flex: 1;
  font-size: var(--font-size-base);
  color: var(--color-text-primary);
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.picker-check {
  color: var(--color-accent);
  font-size: 12px;
  flex-shrink: 0;
}

.picker-empty {
  text-align: center;
  padding: var(--space-2xl);
  color: var(--color-text-tertiary);
  font-size: var(--font-size-base);
}

.picker-cancel {
  width: 100%;
  padding: var(--space-lg);
  border: none;
  border-top: 1px solid var(--color-border-light);
  background: transparent;
  color: var(--color-text-secondary);
  font-size: var(--font-size-base);
  cursor: pointer;
  min-height: 44px;
  -webkit-tap-highlight-color: transparent;
  flex-shrink: 0;
}

.picker-cancel:active {
  background: var(--color-bg-hover);
}

@keyframes slideUp {
  from {
    transform: translateY(100%);
  }
  to {
    transform: translateY(0);
  }
}

@keyframes fadeIn {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}
</style>