<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'

const props = defineProps<{
  index: number
  onHeightChange: (index: number, height: number) => void
}>()

const rootRef = ref<HTMLElement | null>(null)
let observer: ResizeObserver | null = null

onMounted(() => {
  if (!rootRef.value) return
  observer = new ResizeObserver((entries) => {
    for (const entry of entries) {
      const h = Math.round(entry.contentRect.height)
      if (h > 0) {
        props.onHeightChange(props.index, h)
      }
    }
  })
  observer.observe(rootRef.value)
})

onUnmounted(() => {
  if (observer) {
    observer.disconnect()
    observer = null
  }
})
</script>

<template>
  <div ref="rootRef" class="virtual-item">
    <slot />
  </div>
</template>

<style scoped>
.virtual-item {
  width: 100%;
}
</style>