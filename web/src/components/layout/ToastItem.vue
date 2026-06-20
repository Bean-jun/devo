<script setup lang="ts">
import type { Toast } from '@/stores/ui'

const props = defineProps<{
  toast: Toast
}>()

const emit = defineEmits<{
  remove: []
}>()

const iconMap: Record<string, string> = {
  success: '✅',
  error: '❌',
  info: 'ℹ️',
  warning: '⚠️',
}
</script>

<template>
  <div class="toast-item" :class="toast.type" @click="emit('remove')">
    <span class="toast-icon">{{ iconMap[toast.type] ?? 'ℹ️' }}</span>
    <span class="toast-message">{{ toast.message }}</span>
  </div>
</template>

<style scoped>
.toast-item {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
  padding: var(--space-sm) var(--space-md);
  background: var(--color-bg-primary);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-md);
  font-size: var(--font-size-sm);
  cursor: pointer;
  pointer-events: auto;
  animation: slideInRight var(--transition-base) ease;
  min-width: 240px;
  max-width: 360px;
}

.toast-item.success {
  border-left: 3px solid var(--color-success);
}

.toast-item.error {
  border-left: 3px solid var(--color-error);
}

.toast-item.info {
  border-left: 3px solid var(--color-accent);
}

.toast-item.warning {
  border-left: 3px solid var(--color-warning);
}

.toast-icon {
  font-size: var(--font-size-base);
  flex-shrink: 0;
}

.toast-message {
  color: var(--color-text-primary);
  line-height: 1.4;
}
</style>