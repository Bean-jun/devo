<script setup lang="ts">
import { useUiStore } from '@/stores/ui'
import ToastItem from './ToastItem.vue'

const uiStore = useUiStore()
</script>

<template>
  <Teleport to="body">
    <div class="toast-container" v-if="uiStore.toasts.length > 0">
      <ToastItem
        v-for="toast in uiStore.toasts"
        :key="toast.id"
        :toast="toast"
        @remove="uiStore.removeToast(toast.id)"
      />
    </div>
  </Teleport>
</template>

<style scoped>
.toast-container {
  position: fixed;
  top: calc(var(--statusbar-height) + var(--space-sm));
  right: var(--space-lg);
  z-index: 10000;
  display: flex;
  flex-direction: column;
  gap: var(--space-sm);
  pointer-events: none;
}
</style>