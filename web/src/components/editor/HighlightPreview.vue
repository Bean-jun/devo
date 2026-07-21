<template>
  <div ref="wrapperRef" class="highlight-preview-wrapper" :style="wrapperStyle">
    <pre class="highlight-preview"><code v-html="highlightedHtml" /></pre>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted, watch, nextTick } from 'vue'
import hljs from 'highlight.js'

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

const wrapperRef = ref<HTMLElement | null>(null)
const contentHeight = ref(200)

function normalizeLang(lang: string): string {
  const map: Record<string, string> = {
    shell: 'bash',
    bat: 'dos',
    powershell: 'powershell',
  }
  return map[lang] || lang
}

const highlightedHtml = computed(() => {
  if (!props.content) return ''
  const lang = normalizeLang(props.language!)
  if (lang && lang !== 'plaintext' && hljs.getLanguage(lang)) {
    return hljs.highlight(props.content, { language: lang }).value
  }
  return hljs.highlightAuto(props.content).value
})

const wrapperStyle = computed(() => {
  if (!props.autoHeight) return {}
  const h = Math.max(props.minHeight, Math.min(contentHeight.value, props.maxHeight))
  return { height: `${h}px` }
})

function updateHeight() {
  if (!props.autoHeight || !wrapperRef.value) return
  const lineCount = Math.max(1, props.content.split('\n').length)
  const lineHeight = 18
  const padding = 20
  contentHeight.value = lineCount * lineHeight + padding
}

watch(() => props.content, () => {
  nextTick(() => updateHeight())
})

onMounted(() => {
  nextTick(() => updateHeight())
})
</script>

<style scoped>
.highlight-preview-wrapper {
  width: 100%;
  overflow: auto;
  border-radius: 4px;
  background: var(--color-bg-secondary);
}

.highlight-preview {
  margin: 0;
  padding: 10px;
  font-family: var(--font-mono);
  font-size: 12px;
  line-height: 1.5;
  tab-size: 2;
  background: transparent !important;
  color: var(--color-text-primary);
}

.highlight-preview :deep(code) {
  font-family: var(--font-mono);
  font-size: 12px;
  line-height: 1.5;
  background: transparent;
  color: inherit;
}

.highlight-preview :deep(.hljs-keyword) { color: var(--color-syntax-keyword, #d73a49); }
.highlight-preview :deep(.hljs-string)  { color: var(--color-syntax-string, #032f62); }
.highlight-preview :deep(.hljs-number)  { color: var(--color-syntax-number, #005cc5); }
.highlight-preview :deep(.hljs-comment) { color: var(--color-syntax-comment, #6a737d); font-style: italic; }
.highlight-preview :deep(.hljs-function) { color: var(--color-syntax-function, #6f42c1); }
.highlight-preview :deep(.hljs-title)   { color: var(--color-syntax-title, #6f42c1); }
.highlight-preview :deep(.hljs-type)    { color: var(--color-syntax-type, #22863a); }
.highlight-preview :deep(.hljs-attr)    { color: var(--color-syntax-attr, #005cc5); }
.highlight-preview :deep(.hljs-built_in) { color: var(--color-syntax-builtin, #005cc5); }
.highlight-preview :deep(.hljs-literal) { color: var(--color-syntax-literal, #005cc5); }
.highlight-preview :deep(.hljs-meta)    { color: var(--color-syntax-meta, #6a737d); }
.highlight-preview :deep(.hljs-symbol)  { color: var(--color-syntax-symbol, #005cc5); }
.highlight-preview :deep(.hljs-regexp)  { color: var(--color-syntax-regexp, #032f62); }
.highlight-preview :deep(.hljs-selector-class) { color: var(--color-syntax-selector, #22863a); }
.highlight-preview :deep(.hljs-selector-tag)   { color: var(--color-syntax-selector-tag, #22863a); }
.highlight-preview :deep(.hljs-template-variable) { color: var(--color-syntax-template, #e36209); }
.highlight-preview :deep(.hljs-variable) { color: var(--color-syntax-variable, #e36209); }
.highlight-preview :deep(.hljs-addition) { color: var(--color-syntax-addition, #22863a); }
.highlight-preview :deep(.hljs-deletion) { color: var(--color-syntax-deletion, #d73a49); }
.highlight-preview :deep(.hljs-section) { color: var(--color-syntax-section, #005cc5); font-weight: bold; }
.highlight-preview :deep(.hljs-emphasis) { font-style: italic; }
.highlight-preview :deep(.hljs-strong) { font-weight: bold; }
</style>