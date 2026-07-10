<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Message } from '@/types/message'
import { formatTime } from '@/utils/formatters'
import { renderMarkdown } from '@/utils/markdown'
import AppIcon from '@/components/common/AppIcon.vue'

const props = defineProps<{
  message: Message
}>()

const copied = ref(false)

const renderedContent = computed(() => {
  if (props.message.role === 'assistant') {
    return renderMarkdown(props.message.content)
  }
  return props.message.content
})

const displayTime = computed(() => formatTime(props.message.timestamp))

async function copyContent() {
  try {
    await navigator.clipboard.writeText(props.message.content)
    copied.value = true
    setTimeout(() => {
      copied.value = false
    }, 2000)
  } catch {
    // Fallback for older browsers
    const textarea = document.createElement('textarea')
    textarea.value = props.message.content
    textarea.style.position = 'fixed'
    textarea.style.opacity = '0'
    document.body.appendChild(textarea)
    textarea.select()
    document.execCommand('copy')
    document.body.removeChild(textarea)
    copied.value = true
    setTimeout(() => {
      copied.value = false
    }, 2000)
  }
}
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
        <div class="bubble-header-right">
          <button
            v-if="message.role === 'assistant'"
            class="copy-btn"
            :class="{ copied }"
            :title="copied ? '已复制' : '复制内容'"
            @click.stop="copyContent"
          >
            <template v-if="copied">
              <AppIcon name="check" :size="14" />
              已复制
            </template>
            <template v-if="!copied">
              <AppIcon name="copy" :size="14" />
              复制
            </template>
          </button>
          <span class="bubble-time">{{ displayTime }}</span>
        </div>
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

.bubble-header-right {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
}

.copy-btn {
  font-size: var(--font-size-xs);
  color: var(--color-text-tertiary);
  padding: 2px 8px;
  border-radius: var(--radius-sm);
  border: 1px solid var(--color-border-light);
  background: transparent;
  transition: all var(--transition-fast) ease;
  white-space: nowrap;
}

.copy-btn:hover {
  color: var(--color-accent);
  border-color: var(--color-accent);
  background: var(--color-bg-hover);
}

.copy-btn.copied {
  color: var(--color-success);
  border-color: var(--color-success);
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
  overflow-x: auto;
}

.markdown-body :deep(code) {
  font-family: var(--font-mono);
  font-size: 0.85em;
  padding: 0.15em 0.35em;
  border-radius: 3px;
  background: var(--color-bg-tertiary);
}

.markdown-body :deep(pre code) {
  display: block;
  padding: var(--space-md);
  background: none !important;
  color: #e0e0e0;
}

/* 代码块头部：语言标签 */
.markdown-body :deep(pre)::before {
  content: attr(data-lang);
  position: absolute;
  top: 0;
  right: 0;
  padding: 2px 8px;
  font-size: 10px;
  font-family: var(--font-mono);
  color: var(--color-text-tertiary);
  background: var(--color-bg-tertiary);
  border-radius: 0 var(--radius-md) 0 var(--radius-sm);
  z-index: 1;
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