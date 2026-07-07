<script setup lang="ts">
import { computed, ref } from 'vue'
import { useChatStore } from '@/stores/chat'

const props = defineProps<{
  scrollToMessage: (messageId: string) => void
}>()

const chatStore = useChatStore()

const navItems = computed(() => {
  return chatStore.messages
    .filter((msg) => msg.role === 'user')
    .map((msg) => ({
      id: msg.id,
      summary: msg.content.length > 20 ? msg.content.slice(0, 20) + '…' : msg.content,
    }))
})

const activeId = ref<string | null>(null)

function handleClick(itemId: string): void {
  activeId.value = itemId
  props.scrollToMessage(itemId)
}
</script>

<template>
  <div v-if="navItems.length > 0" class="nav-panel">
    <div class="nav-list">
      <button
        v-for="item in navItems"
        :key="item.id"
        class="nav-pill"
        :class="{ active: item.id === activeId }"
        @click="handleClick(item.id)"
      >
        <span class="nav-pill-label">{{ item.summary }}</span>
      </button>
    </div>
  </div>
</template>

<style scoped>
.nav-panel {
  position: absolute;
  right: 6px;
  top: 0;
  bottom: 0;
  width: 24px;
  pointer-events: none;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  z-index: 10;
}

.nav-list {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 3px;
  padding: 0;
  pointer-events: auto;
}

.nav-pill {
  position: relative;
  flex-shrink: 0;
  width: 8px;
  height: 8px;
  margin: 0;
  padding: 0;
  border: none;
  border-radius: 4px;
  background: var(--color-border);
  cursor: pointer;
  transition: all var(--transition-fast) ease;
  transform-origin: right center;
}

.nav-pill:hover {
  transform: scale(1.5);
  background: var(--color-accent);
}

.nav-pill.active {
  width: 14px;
  height: 8px;
  border-radius: 4px;
  background: var(--color-accent);
}

.nav-pill.active:hover {
  transform: scale(1.5);
}

.nav-pill-label {
  position: absolute;
  right: 16px;
  top: 50%;
  transform: translateY(-50%);
  white-space: nowrap;
  font-size: 12px;
  line-height: 1;
  color: var(--color-text-primary);
  background: var(--color-bg-primary);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  padding: 5px 10px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.12);
  opacity: 0;
  pointer-events: none;
  transition: opacity 0.15s ease, transform 0.15s ease;
  z-index: 200;
}

.nav-pill:hover .nav-pill-label {
  opacity: 1;
  transform: translateY(-50%) translateX(-2px);
}

.nav-pill.active .nav-pill-label {
  background: var(--color-accent);
  color: var(--color-text-inverse);
  border-color: var(--color-accent);
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.2);
}
</style>