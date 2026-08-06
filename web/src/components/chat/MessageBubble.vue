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

const contentRef = ref<HTMLElement | null>(null)
const lightboxImage = ref<string | null>(null)

function openLightbox(src: string): void {
  lightboxImage.value = src
}

function closeLightbox(): void {
  lightboxImage.value = null
}

async function handleCodeBlockCopy(e: MouseEvent): Promise<void> {
  const target = e.target as HTMLElement
  const btn = target.closest('.code-block-copy') as HTMLButtonElement | null
  if (!btn) return
  const text = btn.getAttribute('data-code') || ''
  try {
    await navigator.clipboard.writeText(text)
  } catch {
    const textarea = document.createElement('textarea')
    textarea.value = text
    textarea.style.position = 'fixed'
    textarea.style.opacity = '0'
    document.body.appendChild(textarea)
    textarea.select()
    document.execCommand('copy')
    document.body.removeChild(textarea)
  }
  const orig = btn.textContent
  btn.textContent = '已复制'
  btn.classList.add('copied')
  setTimeout(() => {
    btn.textContent = orig
    btn.classList.remove('copied')
  }, 2000)
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

      <details v-if="message.role === 'assistant' && message.reasoning" class="reasoning-collapse" data-test="reasoning-collapse">
        <summary class="reasoning-summary">
          <AppIcon name="brain" :size="14" class="reasoning-summary-icon" />
          思考过程
        </summary>
        <pre class="reasoning-text">{{ message.reasoning }}</pre>
      </details>

      <div
        v-if="message.role === 'assistant'"
        ref="contentRef"
        class="bubble-content markdown-body"
        v-html="renderedContent"
        @click="handleCodeBlockCopy"
      />
      <div v-else class="bubble-content">
        <div v-if="message.images && message.images.length > 0" class="message-images">
          <img
            v-for="(img, idx) in message.images"
            :key="idx"
            :src="img"
            alt="uploaded image"
            class="message-image"
            @click="openLightbox(img)"
          />
        </div>
        {{ message.content }}
      </div>
    </div>

    <Teleport to="body">
      <div v-if="lightboxImage" class="image-lightbox" @click="closeLightbox">
        <img :src="lightboxImage" alt="enlarged" class="lightbox-image" @click.stop />
        <button class="lightbox-close" @click="closeLightbox" aria-label="关闭">&times;</button>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.message-bubble {
  margin-bottom: var(--space-xl);
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

.message-bubble.role-user .bubble-inner ::selection {
  background: var(--color-user-bubble-text);
  color: var(--color-user-bubble);
}

.message-bubble.role-assistant .bubble-inner {
  background: var(--color-assistant-bubble);
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
  line-height: 1.7;
  word-break: break-word;
}

.role-system .bubble-content {
  font-size: var(--font-size-xs);
  color: var(--color-text-tertiary);
  text-align: center;
}

/* Markdown 内容样式 */
.markdown-body :deep(p) {
  margin-bottom: var(--space-md);
}

.markdown-body :deep(p:last-child) {
  margin-bottom: 0;
}

/* 标题 */
.markdown-body :deep(h1) {
  font-size: 1.3em;
  font-weight: 600;
  margin: var(--space-lg) 0 var(--space-md);
  padding-bottom: var(--space-xs);
  border-bottom: 1px solid var(--color-border-light);
}

.markdown-body :deep(h2) {
  font-size: 1.2em;
  font-weight: 600;
  margin: var(--space-lg) 0 var(--space-md);
  padding-bottom: var(--space-xs);
  border-bottom: 1px solid var(--color-border-light);
}

.markdown-body :deep(h3) {
  font-size: 1.1em;
  font-weight: 600;
  margin: var(--space-lg) 0 var(--space-sm);
}

.markdown-body :deep(h4) {
  font-size: 1em;
  font-weight: 600;
  margin: var(--space-md) 0 var(--space-sm);
}

.markdown-body :deep(h5) {
  font-size: 0.95em;
  font-weight: 600;
  margin: var(--space-md) 0 var(--space-sm);
}

.markdown-body :deep(h6) {
  font-size: 0.9em;
  font-weight: 600;
  color: var(--color-text-secondary);
  margin: var(--space-md) 0 var(--space-sm);
}

/* 代码块容器 */
.markdown-body :deep(.code-block) {
  margin: var(--space-md) 0;
  border-radius: var(--radius-md);
  overflow: hidden;
  border: 1px solid var(--color-border-light);
  box-shadow: var(--shadow-sm);
}

.markdown-body :deep(.code-block-header) {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-xs) var(--space-md);
  background: rgba(125, 125, 125, 0.08);
  border-bottom: 1px solid var(--color-border-light);
}

.markdown-body :deep(.code-block-lang) {
  font-size: var(--font-size-xs);
  font-family: var(--font-mono);
  color: var(--color-text-tertiary);
  text-transform: lowercase;
}

.markdown-body :deep(.code-block-copy) {
  font-size: var(--font-size-xs);
  color: var(--color-text-tertiary);
  padding: 2px 8px;
  border-radius: var(--radius-sm);
  border: 1px solid transparent;
  background: transparent;
  cursor: pointer;
  transition: all var(--transition-fast) ease;
}

.markdown-body :deep(.code-block-copy:hover) {
  color: var(--color-accent);
  border-color: var(--color-accent);
  background: var(--color-bg-hover);
}

.markdown-body :deep(.code-block-copy.copied) {
  color: var(--color-success);
  border-color: var(--color-success);
}

.markdown-body :deep(pre) {
  margin: 0;
  background: var(--color-code-bg);
  overflow-x: auto;
}

.markdown-body :deep(pre code) {
  display: block;
  padding: var(--space-md) var(--space-lg);
  background: none;
  color: #e0e0e0;
  line-height: 1.6;
  font-family: var(--font-mono);
  font-size: 0.85em;
}

/* 行内代码 */
.markdown-body :deep(code) {
  font-family: var(--font-mono);
  font-size: 0.85em;
  padding: 0.15em 0.35em;
  border-radius: 3px;
  background: rgba(125, 125, 125, 0.15);
  color: inherit;
}

.markdown-body :deep(pre code) {
  background: none;
  padding: var(--space-md) var(--space-lg);
  color: #e0e0e0;
}

/* 引用块 */
.markdown-body :deep(blockquote) {
  margin: var(--space-md) 0;
  padding: var(--space-sm) var(--space-lg);
  border-left: 4px solid var(--color-accent);
  background: rgba(125, 125, 125, 0.08);
  border-radius: 0 var(--radius-sm) var(--radius-sm) 0;
  color: var(--color-text-secondary);
}

.markdown-body :deep(blockquote p:last-child) {
  margin-bottom: 0;
}

/* 列表 */
.markdown-body :deep(ul),
.markdown-body :deep(ol) {
  padding-left: var(--space-xl);
  margin: var(--space-md) 0;
}

.markdown-body :deep(li) {
  margin: var(--space-sm) 0;
}

.markdown-body :deep(li::marker) {
  color: var(--color-text-tertiary);
}

/* GFM 任务列表 */
.markdown-body :deep(input[type="checkbox"]) {
  margin-right: var(--space-sm);
  accent-color: var(--color-accent);
  vertical-align: middle;
}

/* 表格 */
.markdown-body :deep(.table-wrap) {
  overflow-x: auto;
  margin: var(--space-md) 0;
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border);
}

.markdown-body :deep(table) {
  width: 100%;
  border-collapse: collapse;
  font-size: var(--font-size-sm);
}

.markdown-body :deep(th),
.markdown-body :deep(td) {
  padding: var(--space-sm) var(--space-md);
  border: 1px solid var(--color-border);
  text-align: left;
}

.markdown-body :deep(th) {
  background: rgba(125, 125, 125, 0.1);
  font-weight: 600;
  border-bottom: 2px solid var(--color-border);
}

.markdown-body :deep(tr:nth-child(even)) td {
  background: rgba(125, 125, 125, 0.05);
}

/* 链接 */
.markdown-body :deep(a) {
  color: var(--color-accent);
  text-decoration: none;
  transition: opacity var(--transition-fast) ease;
}

.markdown-body :deep(a:hover) {
  text-decoration: underline;
  opacity: 0.85;
}

/* 图片 */
.markdown-body :deep(img) {
  max-width: 100%;
  border-radius: var(--radius-md);
}

/* 分割线 */
.markdown-body :deep(hr) {
  margin: var(--space-lg) 0;
  border: none;
  border-top: 1px solid var(--color-border-light);
}

.reasoning-collapse {
  margin: 0 0 var(--space-sm) 0;
  padding: var(--space-xs) var(--space-sm);
  border: 1px dashed var(--color-border-light);
  border-radius: var(--radius-md);
  background: var(--color-bg-tertiary);
}

.reasoning-summary {
  font-size: var(--font-size-xs);
  font-weight: 600;
  color: var(--color-text-secondary);
  cursor: pointer;
  user-select: none;
  list-style: none;
  padding: 2px 0;
}

.reasoning-summary::-webkit-details-marker {
  display: none;
}

.reasoning-summary::before {
  content: '▸ ';
  font-size: 10px;
  opacity: 0.7;
}

.reasoning-collapse[open] .reasoning-summary::before {
  content: '▾ ';
}

.reasoning-summary-icon {
  vertical-align: middle;
  margin-right: 2px;
}

.reasoning-text {
  margin: var(--space-xs) 0 0 0;
  padding: 0;
  font-family: var(--font-mono);
  font-size: var(--font-size-xs);
  line-height: 1.6;
  color: var(--color-text-tertiary);
  font-style: italic;
  white-space: pre-wrap;
  word-break: break-word;
  background: none;
  border: none;
  max-height: 400px;
  overflow-y: auto;
}

.message-images {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 8px;
}

.message-image {
  max-width: 200px;
  max-height: 200px;
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border);
  object-fit: cover;
  cursor: pointer;
  transition: transform var(--transition-fast);
}

.message-image:hover {
  transform: scale(1.02);
}

.image-lightbox {
  position: fixed;
  top: 0;
  left: 0;
  width: 100vw;
  height: 100vh;
  background: rgba(0, 0, 0, 0.85);
  z-index: 9999;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
}

.lightbox-image {
  max-width: 90vw;
  max-height: 90vh;
  object-fit: contain;
  border-radius: var(--radius-md);
  cursor: default;
}

.lightbox-close {
  position: absolute;
  top: 16px;
  right: 16px;
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.15);
  color: #fff;
  border: none;
  font-size: 24px;
  line-height: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  padding: 0;
  transition: background var(--transition-fast);
}

.lightbox-close:hover {
  background: rgba(255, 255, 255, 0.3);
}
</style>