<script setup lang="ts">
import type { Toast } from '@/stores/ui'
import AppIcon from '@/components/common/AppIcon.vue'

const props = defineProps<{
  toast: Toast
}>()

const emit = defineEmits<{
  remove: []
}>()

const iconMap = {
  success: { name: 'check-circle', color: 'var(--color-success)' },
  error: { name: 'x-circle', color: 'var(--color-error)' },
  info: { name: 'info', color: 'var(--color-accent)' },
  warning: { name: 'warning', color: 'var(--color-warning)' },
} as const
</script>

<template>
  <div class="toast-item" :class="toast.type" @click="emit('remove')">
    <AppIcon
      :name="iconMap[toast.type]?.name ?? 'info'"
      :size="16"
      :color="iconMap[toast.type]?.color"
      class="toast-icon"
    />
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