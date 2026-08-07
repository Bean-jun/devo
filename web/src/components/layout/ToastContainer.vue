<script setup lang="ts">
import { useToastContainer } from './ToastContainerController'

const { uiStore, ToastItem } = useToastContainer()
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

<style scoped src="./ToastContainer.css">
</style>