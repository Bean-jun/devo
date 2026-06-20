<script setup lang="ts">
import { useChatStore } from '@/stores/chat'

const chatStore = useChatStore()
</script>

<template>
  <div class="thinking-indicator" data-test="thinking-indicator">
    <div class="thinking-bubble">
      <span class="thinking-label">Devo 思考中</span>
      <span class="thinking-dots">
        <span class="dot"></span>
        <span class="dot"></span>
        <span class="dot"></span>
      </span>
    </div>
    <div v-if="chatStore.streamingContent" class="streaming-preview">
      {{ chatStore.streamingContent.slice(-100) }}
      <span class="cursor-blink">|</span>
    </div>
  </div>
</template>

<style scoped>
.thinking-indicator {
  margin-bottom: var(--space-lg);
  animation: fadeIn var(--transition-fast) ease;
}

.thinking-bubble {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
  padding: var(--space-sm) var(--space-md);
  background: var(--color-assistant-bubble);
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-lg);
  max-width: 200px;
}

.thinking-label {
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
  font-style: italic;
}

.thinking-dots {
  display: flex;
  gap: 3px;
}

.dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: var(--color-text-tertiary);
  animation: bounce 1.4s ease-in-out infinite;
}

.dot:nth-child(1) {
  animation-delay: 0s;
}

.dot:nth-child(2) {
  animation-delay: 0.2s;
}

.dot:nth-child(3) {
  animation-delay: 0.4s;
}

.streaming-preview {
  margin-top: var(--space-xs);
  padding: var(--space-xs) var(--space-md);
  font-size: var(--font-size-sm);
  color: var(--color-text-tertiary);
  font-style: italic;
  max-width: 600px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.cursor-blink {
  animation: pulse 1s ease-in-out infinite;
  color: var(--color-accent);
}
</style>