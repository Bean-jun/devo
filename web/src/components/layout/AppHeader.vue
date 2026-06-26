<script setup lang="ts">
import { computed } from 'vue'
import { useSessionStore } from '@/stores/session'

const sessionStore = useSessionStore()

const sessionTitle = computed(() => {
  return sessionStore.currentSession?.title || 'Devo'
})
</script>

<template>
  <header class="app-header">
    <div class="header-left">
      <h1 class="header-title">{{ sessionTitle }}</h1>
    </div>
    <div class="header-right">
      <slot name="actions" />
    </div>
  </header>
</template>

<style scoped>
.app-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 48px;
  padding: 0 20px;
  background: var(--color-bg-primary);
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}

.header-left {
  display: flex;
  align-items: center;
}

.header-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin: 0;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 8px;
}
</style>