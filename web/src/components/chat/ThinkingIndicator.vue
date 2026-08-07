<script setup lang="ts">
import AppIcon from '@/components/common/AppIcon.vue'
import { useThinkingIndicator } from './ThinkingIndicatorController'

const { chatStore, hasContent, hasReasoning, isReasoningActive, reasoningExpanded, toggleReasoning } = useThinkingIndicator()
</script>

<template>
  <div class="thinking-indicator" data-test="thinking-indicator">
    <div v-if="hasReasoning" class="reasoning-section" data-test="reasoning-section">
      <div class="reasoning-header" @click="toggleReasoning" data-test="reasoning-toggle">
        <AppIcon name="brain" :size="16" class="reasoning-icon" />
        <span class="reasoning-title">
          {{ isReasoningActive ? '正在思考...' : '思考过程' }}
        </span>
        <span v-if="isReasoningActive" class="thinking-dots">
          <span class="dot"></span>
          <span class="dot"></span>
          <span class="dot"></span>
        </span>
        <AppIcon
          :name="reasoningExpanded ? 'caret-down' : 'caret-right'"
          :size="12"
          class="toggle-icon"
        />
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

<style scoped src="./ThinkingIndicator.css">
</style>