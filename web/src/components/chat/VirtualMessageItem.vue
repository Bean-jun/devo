<script setup lang="ts">
import { useVirtualMessageItem } from './VirtualMessageItemController'

const props = defineProps<{
  index: number
  onHeightChange: (index: number, height: number) => void
}>()

const { rootRef } = useVirtualMessageItem(props)
</script>

<template>
  <div ref="rootRef" class="virtual-item">
    <slot />
  </div>
</template>

<style scoped src="./VirtualMessageItem.css">
</style>