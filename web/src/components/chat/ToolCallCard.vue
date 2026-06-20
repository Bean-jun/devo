<script setup lang="ts">
import { ref } from 'vue'
import type { ToolCall } from '@/types/tool'
import { formatDuration } from '@/utils/formatters'
import { RISK_LABELS, RISK_COLORS } from '@/utils/constants'

const props = defineProps<{
  toolCall: ToolCall
}>()

const showParams = ref(false)
const showResult = ref(true)

const statusIcon: Record<string, string> = {
  pending: '⏳',
  executing: '🔄',
  success: '✅',
  failed: '❌',
  rejected: '🚫',
}

const statusClass: Record<string, string> = {
  pending: 'status-pending',
  executing: 'status-executing',
  success: 'status-success',
  failed: 'status-failed',
  rejected: 'status-rejected',
}
</script>

<template>
  <div class="tool-call-card" :class="statusClass[toolCall.status]" data-test="tool-call-card">
    <div class="tool-header" @click="showParams = !showParams">
      <div class="tool-left">
        <span class="tool-icon">{{ statusIcon[toolCall.status] ?? '🔧' }}</span>
        <span class="tool-name" data-test="tool-name">{{ toolCall.name }}</span>
        <span v-if="toolCall.riskLevel" class="tool-risk" :style="{ color: RISK_COLORS[toolCall.riskLevel] }">
          {{ RISK_LABELS[toolCall.riskLevel] }}
        </span>
      </div>
      <div class="tool-right">
        <span v-if="toolCall.duration" class="tool-duration">{{ formatDuration(toolCall.duration) }}</span>
        <span class="tool-chevron">{{ showParams ? '▾' : '▸' }}</span>
      </div>
    </div>

    <div v-if="showParams" class="tool-params">
      <div class="tool-section-title">参数</div>
      <pre class="tool-json">{{ JSON.stringify(toolCall.parameters, null, 2) }}</pre>
    </div>

    <div v-if="toolCall.status !== 'pending' && showResult" class="tool-result">
      <div class="tool-section-title" @click="showResult = !showResult">
        结果 {{ showResult ? '▾' : '▸' }}
      </div>
      <div v-if="toolCall.result" class="tool-result-content">
        <div v-if="toolCall.result.success !== undefined" class="result-status">
          {{ toolCall.result.success ? '✅ 成功' : '❌ 失败' }}
        </div>
        <div v-if="toolCall.result.error" class="result-error">{{ toolCall.result.error }}</div>
        <pre v-if="toolCall.result.stdout" class="tool-json">{{ toolCall.result.stdout }}</pre>
        <pre v-if="toolCall.result.stderr" class="tool-json stderr">{{ toolCall.result.stderr }}</pre>
      </div>
    </div>
  </div>
</template>

<style scoped>
.tool-call-card {
  margin-bottom: var(--space-md);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-bg-primary);
  overflow: hidden;
  animation: fadeIn var(--transition-fast) ease;
}

.tool-call-card.status-pending {
  border-left: 3px solid var(--color-warning);
}

.tool-call-card.status-executing {
  border-left: 3px solid var(--color-accent);
}

.tool-call-card.status-success {
  border-left: 3px solid var(--color-success);
}

.tool-call-card.status-failed {
  border-left: 3px solid var(--color-error);
}

.tool-call-card.status-rejected {
  border-left: 3px solid var(--color-text-tertiary);
  opacity: 0.6;
}

.tool-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-sm) var(--space-md);
  cursor: pointer;
  user-select: none;
}

.tool-header:hover {
  background: var(--color-bg-hover);
}

.tool-left {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
}

.tool-icon {
  font-size: var(--font-size-base);
}

.tool-name {
  font-family: var(--font-mono);
  font-size: var(--font-size-sm);
  font-weight: 600;
  color: var(--color-text-primary);
}

.tool-risk {
  font-size: 10px;
  font-weight: 600;
  padding: 1px 6px;
  border-radius: var(--radius-full);
  background: var(--color-bg-tertiary);
}

.tool-right {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
}

.tool-duration {
  font-size: var(--font-size-xs);
  color: var(--color-text-secondary);
  font-family: var(--font-mono);
}

.tool-chevron {
  font-size: var(--font-size-xs);
  color: var(--color-text-tertiary);
}

.tool-params,
.tool-result {
  border-top: 1px solid var(--color-border-light);
  padding: var(--space-sm) var(--space-md);
}

.tool-section-title {
  font-size: var(--font-size-xs);
  font-weight: 600;
  color: var(--color-text-secondary);
  margin-bottom: var(--space-xs);
  cursor: pointer;
  user-select: none;
}

.tool-json {
  font-family: var(--font-mono);
  font-size: var(--font-size-xs);
  background: var(--color-bg-secondary);
  padding: var(--space-sm);
  border-radius: var(--radius-sm);
  overflow-x: auto;
  max-height: 200px;
  overflow-y: auto;
  white-space: pre-wrap;
  word-break: break-all;
  color: var(--color-text-primary);
  line-height: 1.5;
}

.tool-json.stderr {
  color: var(--color-error);
}

.result-status {
  font-size: var(--font-size-sm);
  margin-bottom: var(--space-xs);
}

.result-error {
  font-size: var(--font-size-sm);
  color: var(--color-error);
  margin-bottom: var(--space-xs);
}
</style>