<script setup lang="ts">
import AppIcon from '@/components/common/AppIcon.vue'
import type { Toast } from '@/stores/ui'
import { useToastItem } from './ToastItemController'

const props = defineProps<{
  toast: Toast
}>()

const emit = defineEmits<{
  remove: []
}>()

const { iconMap } = useToastItem(props, emit as any)
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

<style scoped src="./ToastItem.css">
</style>