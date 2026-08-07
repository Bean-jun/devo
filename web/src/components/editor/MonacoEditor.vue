<template>
  <div ref="containerRef" class="monaco-editor-wrapper" :style="{ height: computedHeight }"></div>
</template>

<script setup lang="ts">
import { useMonacoEditor } from './MonacoEditorController'

const props = withDefaults(
  defineProps<{
    content: string
    language?: string
    readonly?: boolean
    theme?: 'vs' | 'vs-dark'
    autoHeight?: boolean
    minHeight?: number
    maxHeight?: number
  }>(),
  {
    language: 'plaintext',
    readonly: true,
    theme: 'vs',
    autoHeight: true,
    minHeight: 100,
    maxHeight: 600,
  }
)

const { containerRef, computedHeight } = useMonacoEditor(props)
</script>

<style scoped src="./MonacoEditor.css">
</style>