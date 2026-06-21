<script setup lang="ts">
import { computed } from 'vue'
import type { Message } from '@/types/message'
import { formatTime } from '@/utils/formatters'
import { marked } from 'marked'
import { markedHighlight } from 'marked-highlight'
import hljs from 'highlight.js'
import 'highlight.js/styles/github-dark.css'

const props = defineProps<{
  message: Message
}>()

marked.use(markedHighlight({
  langPrefix: 'hljs language-',
  highlight(code: string, lang: string) {
    if (lang && hljs.getLanguage(lang)) {
      return hljs.highlight(code, { language: lang }).value
    }
    return hljs.highlightAuto(code).value
  },
}))

const renderedContent = computed(() => {
  if (props.message.role === 'assistant') {
    return marked.parse(props.message.content, { breaks: true }) as string
  }
  return props.message.content
})

const displayTime = computed(() => formatTime(props.message.timestamp))
</script>

<template>
  <div
    class="message-bubble"
    :class="[`role-${message.role}`]"
    :data-test="message.role === 'assistant' ? 'message-bubble assistant' : 'message-bubble'"
  >
    <div class="bubble-inner">
      <div class="bubble-header">
        <span class="bubble-role">
          {{ message.role === 'user' ? '你' : message.role === 'system' ? '系统' : 'Devo' }}
        </span>
        <span class="bubble-time">{{ displayTime }}</span>
      </div>

      <div
        v-if="message.role === 'assistant'"
        class="bubble-content markdown-body"
        v-html="renderedContent"
      />
      <div v-else class="bubble-content">
        {{ message.content }}
      </div>
    </div>
  </div>
</template>

<style scoped>
.message-bubble {
  margin-bottom: var(--space-lg);
  animation: fadeIn var(--transition-fast) ease;
}

.message-bubble.role-user {
  display: flex;
  justify-content: flex-end;
}

.message-bubble.role-user .bubble-inner {
  background: var(--color-user-bubble);
  color: var(--color-user-bubble-text);
  border-radius: var(--radius-lg) var(--radius-lg) var(--radius-sm) var(--radius-lg);
  max-width: 75%;
}

.message-bubble.role-assistant .bubble-inner {
  background: var(--color-assistant-bubble);
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-lg) var(--radius-lg) var(--radius-lg) var(--radius-sm);
  max-width: 85%;
}

.message-bubble.role-system {
  display: flex;
  justify-content: center;
}

.message-bubble.role-system .bubble-inner {
  background: none;
  border: none;
  max-width: 100%;
  padding: var(--space-xs) var(--space-md);
}

.bubble-inner {
  padding: var(--space-md) var(--space-lg);
  position: relative;
}

.bubble-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--space-xs);
}

.bubble-role {
  font-size: var(--font-size-xs);
  font-weight: 600;
  opacity: 0.7;
}

.role-system .bubble-role {
  color: var(--color-text-tertiary);
}

.bubble-time {
  font-size: 10px;
  opacity: 0.5;
  font-family: var(--font-mono);
}

.bubble-content {
  font-size: var(--font-size-base);
  line-height: 1.6;
  word-break: break-word;
}

.role-system .bubble-content {
  font-size: var(--font-size-xs);
  color: var(--color-text-tertiary);
  text-align: center;
}

/* Markdown 内容样式 */
.markdown-body :deep(p) {
  margin-bottom: var(--space-sm);
}

.markdown-body :deep(p:last-child) {
  margin-bottom: 0;
}

.markdown-body :deep(pre) {
  position: relative;
  margin: var(--space-sm) 0;
  background: var(--color-code-bg) !important;
  border-radius: var(--radius-md);
  overflow: hidden;
}

.markdown-body :deep(code) {
  font-family: var(--font-mono);
  font-size: 0.85em;
}

.markdown-body :deep(pre code) {
  display: block;
  padding: var(--space-md);
  overflow-x: auto;
  background: none !important;
}

.markdown-body :deep(blockquote) {
  margin: var(--space-sm) 0;
  padding: var(--space-xs) var(--space-md);
  border-left: 3px solid var(--color-accent);
  background: var(--color-bg-hover);
  border-radius: 0 var(--radius-sm) var(--radius-sm) 0;
}

.markdown-body :deep(ul),
.markdown-body :deep(ol) {
  padding-left: var(--space-xl);
  margin: var(--space-sm) 0;
}

.markdown-body :deep(li) {
  margin: var(--space-xs) 0;
}

.markdown-body :deep(table) {
  width: 100%;
  margin: var(--space-sm) 0;
  border-collapse: collapse;
  font-size: var(--font-size-sm);
}

.markdown-body :deep(th),
.markdown-body :deep(td) {
  padding: var(--space-xs) var(--space-sm);
  border: 1px solid var(--color-border-light);
  text-align: left;
}

.markdown-body :deep(th) {
  background: var(--color-bg-tertiary);
  font-weight: 600;
}

.markdown-body :deep(a) {
  color: var(--color-accent);
}

.markdown-body :deep(hr) {
  margin: var(--space-md) 0;
  border: none;
  border-top: 1px solid var(--color-border-light);
}
</style>