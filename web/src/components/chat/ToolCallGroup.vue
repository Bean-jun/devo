<script setup lang="ts">
import { ref, computed } from 'vue'
import type { Message } from '@/types/message'
import ToolCallCard from './ToolCallCard.vue'

const props = defineProps<{
  messages: Message[]
}>()

const expanded = ref(false)

const toolNames = computed(() => {
  const names = props.messages
    .map(m => m.toolCall?.name)
    .filter(Boolean) as string[]
  return [...new Set(names)]
})

const successCount = computed(() =>
  props.messages.filter(m => m.toolCall?.status === 'success').length
)

const failedCount = computed(() =>
  props.messages.filter(m => m.toolCall?.status === 'failed').length
)

const pendingCount = computed(() =>
  props.messages.filter(m => m.toolCall?.status === 'pending' || m.toolCall?.status === 'executing').length
)

const summaryText = computed(() => {
  const parts: string[] = []
  if (successCount.value > 0) parts.push(`${successCount.value} 成功`)
  if (failedCount.value > 0) parts.push(`${failedCount.value} 失败`)
  if (pendingCount.value > 0) parts.push(`${pendingCount.value} 进行中`)
  return parts.join('，')
})

function toggle() {
  expanded.value = !expanded.value
}
</script>

<template>
  <div class="tool-call-group" :class="{ expanded }">
    <div class="group-header" @click="toggle">
      <span class="group-chevron">{{ expanded ? '▾' : '▸' }}</span>
      <span class="group-icon">🔧</span>
      <span class="group-title">{{ messages.length }} 个工具调用</span>
      <span class="group-tools">{{ toolNames.join(', ') }}</span>
      <span v-if="summaryText" class="group-summary">{{ summaryText }}</span>
    </div>

    <div v-if="expanded" class="group-body">
      <ToolCallCard
        v-for="msg in messages"
        :key="msg.id"
        :tool-call="msg.toolCall!"
      />
    </div>
  </div>
</template>

<style scoped>
.tool-call-group {
  margin-bottom: var(--space-md);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  overflow: hidden;
}

.group-header {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
  padding: var(--space-sm) var(--space-md);
  cursor: pointer;
  user-select: none;
  background: var(--color-bg-secondary);
  transition: background var(--transition-fast) ease;
}

.group-header:hover {
  background: var(--color-bg-hover);
}

.group-chevron {
  font-size: var(--font-size-xs);
  color: var(--color-text-tertiary);
  width: 12px;
  flex-shrink: 0;
}

.group-icon {
  font-size: var(--font-size-base);
  flex-shrink: 0;
}

.group-title {
  font-size: var(--font-size-sm);
  font-weight: 600;
  color: var(--color-text-primary);
  white-space: nowrap;
}

.group-tools {
  font-family: var(--font-mono);
  font-size: var(--font-size-xs);
  color: var(--color-text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.group-summary {
  font-size: var(--font-size-xs);
  color: var(--color-text-tertiary);
  margin-left: auto;
  white-space: nowrap;
}

.group-body {
  border-top: 1px solid var(--color-border-light);
  padding: var(--space-sm) var(--space-md);
  background: var(--color-bg-primary);
}

.group-body .tool-call-card {
  margin-bottom: var(--space-sm);
}

.group-body .tool-call-card:last-child {
  margin-bottom: 0;
}
</style>