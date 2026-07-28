<script setup lang="ts">
import { computed, ref } from 'vue'
import { useChatStore } from '@/stores/chat'

const chatStore = useChatStore()

const hasContent = computed(() => chatStore.streamingContent.length > 0)
const hasReasoning = computed(() => chatStore.streamingReasoning.length > 0)
const isReasoningActive = computed(() => chatStore.isReasoningActive)

const reasoningExpanded = ref(false)

function toggleReasoning() {
  reasoningExpanded.value = !reasoningExpanded.value
}
</script>

<template>
  <div class="thinking-indicator" data-test="thinking-indicator">
    <div v-if="hasReasoning" class="reasoning-section" data-test="reasoning-section">
      <div class="reasoning-header" @click="toggleReasoning" data-test="reasoning-toggle">
        <span class="reasoning-icon">💭</span>
        <span class="reasoning-title">
          {{ isReasoningActive ? '正在思考...' : '思考过程' }}
        </span>
        <span v-if="isReasoningActive" class="thinking-dots">
          <span class="dot"></span>
          <span class="dot"></span>
          <span class="dot"></span>
        </span>
        <span class="toggle-icon">{{ reasoningExpanded ? '▼' : '▶' }}</span>
      </div>
      <div v-show="reasoningExpanded" class="reasoning-content" data-test="reasoning-content">
        <pre class="reasoning-text">{{ chatStore.streamingReasoning }}</pre>
      </div>
    </div>

    <div class="streaming-bubble">
      <div class="bubble-header">
        <span class="bubble-role">
          Devo
          <span v-if="!hasContent && isReasoningActive" class="thinking-dots">
            <span class="dot"></span>
            <span class="dot"></span>
            <span class="dot"></span>
          </span>
        </span>
      </div>

      <div v-if="hasContent" class="bubble-content">
        <pre class="streaming-text">{{ chatStore.streamingContent }}<span class="cursor-blink">|</span></pre>
      </div>
      <div v-else-if="!isReasoningActive" class="bubble-content bubble-empty">
        <span class="empty-hint">正在思考...</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.thinking-indicator {
  margin-bottom: var(--space-lg);
  animation: fadeIn var(--transition-fast) ease;
}

.reasoning-section {
  margin-bottom: var(--space-sm);
  border: 1px dashed var(--color-border-light);
  border-radius: var(--radius-md);
  background: var(--color-bg-tertiary);
  opacity: 0.9;
  overflow: hidden;
}

.reasoning-header {
  display: flex;
  align-items: center;
  gap: var(--space-xs);
  padding: var(--space-xs) var(--space-md);
  cursor: pointer;
  user-select: none;
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
  transition: background var(--transition-fast) ease;
}

.reasoning-header:hover {
  background: var(--color-bg-hover);
}

.reasoning-icon {
  font-size: var(--font-size-base);
}

.reasoning-title {
  font-weight: 600;
  flex: 1;
}

.toggle-icon {
  font-size: 10px;
  opacity: 0.7;
}

.reasoning-content {
  padding: var(--space-sm) var(--space-md);
  border-top: 1px dashed var(--color-border-light);
  max-height: 300px;
  overflow-y: auto;
}

.reasoning-text {
  margin: 0;
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
}

.streaming-bubble {
  background: var(--color-assistant-bubble);
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-lg);
  max-width: 100%;
  overflow: hidden;
}

.bubble-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-xs) var(--space-md);
  border-bottom: 1px solid var(--color-border-light);
  background: var(--color-bg-tertiary);
}

.bubble-role {
  font-size: var(--font-size-sm);
  font-weight: 600;
  color: var(--color-text-secondary);
  display: flex;
  align-items: center;
  gap: var(--space-xs);
}

.thinking-dots {
  display: flex;
  gap: 3px;
  align-items: center;
}

.dot {
  width: 4px;
  height: 4px;
  border-radius: 50%;
  background: var(--color-accent);
  animation: bounce 1.4s ease-in-out infinite;
}

.dot:nth-child(1) { animation-delay: 0s; }
.dot:nth-child(2) { animation-delay: 0.2s; }
.dot:nth-child(3) { animation-delay: 0.4s; }

@keyframes bounce {
  0%, 80%, 100% {
    transform: translateY(0);
    opacity: 0.4;
  }
  40% {
    transform: translateY(-4px);
    opacity: 1;
  }
}

.bubble-content {
  padding: var(--space-md);
}

.bubble-empty {
  display: flex;
  align-items: center;
  justify-content: center;
}

.empty-hint {
  font-size: var(--font-size-sm);
  color: var(--color-text-tertiary);
  font-style: italic;
}

.streaming-text {
  margin: 0;
  padding: 0;
  font-family: var(--font-sans);
  font-size: var(--font-size-base);
  line-height: 1.65;
  color: var(--color-text-primary);
  white-space: pre-wrap;
  word-break: break-word;
  background: none;
  border: none;
  overflow: visible;
}

.cursor-blink {
  animation: blink 1s step-end infinite;
  color: var(--color-accent);
  font-weight: 400;
}

@keyframes blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0; }
}
</style>
