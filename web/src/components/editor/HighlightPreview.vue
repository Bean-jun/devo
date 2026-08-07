<template>
  <div ref="wrapperRef" class="highlight-preview-wrapper" :style="wrapperStyle">
    <pre class="highlight-preview"><code v-html="highlightedHtml" /></pre>
  </div>
</template>

<script setup lang="ts">
import { useHighlightPreview } from './HighlightPreviewController'

const props = withDefaults(
  defineProps<{
    content: string
    language?: string
    autoHeight?: boolean
    minHeight?: number
    maxHeight?: number
  }>(),
  {
    language: 'plaintext',
    autoHeight: true,
    minHeight: 80,
    maxHeight: 600,
  }
)

const { wrapperRef, highlightedHtml, wrapperStyle } = useHighlightPreview(props)
</script>

<style scoped src="./HighlightPreview.css">
</style>