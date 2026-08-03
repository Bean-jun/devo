<script setup lang="ts">
import { ref, computed } from 'vue'
import type { ToolCall } from '@/types/tool'
import { formatDuration } from '@/utils/formatters'
import { RISK_LABELS, RISK_COLORS } from '@/utils/constants'
import AppIcon from '@/components/common/AppIcon.vue'

const props = defineProps<{
  toolCall: ToolCall
  yoloMode?: boolean
}>()

const showParams = ref(false)
const showResult = ref(!props.yoloMode)

const statusIcon = {
  pending: 'clock',
  executing: 'arrow-clockwise',
  success: 'check-circle',
  failed: 'x-circle',
  rejected: 'prohibit',
} as const

const statusColor: Record<string, string> = {
  pending: 'var(--color-warning)',
  executing: 'var(--color-accent)',
  success: 'var(--color-success)',
  failed: 'var(--color-error)',
  rejected: 'var(--color-text-tertiary)',
}

const statusClass: Record<string, string> = {
  pending: 'status-pending',
  executing: 'status-executing',
  success: 'status-success',
  failed: 'status-failed',
  rejected: 'status-rejected',
}

/**
 * 渲染 diff 文本，高亮 +/- 行，长内容截断
 */
function renderDiff(diff: string): string {
  const MAX_LINES = 200
  const lines = diff.split('\n')
  const truncated = lines.length > MAX_LINES
  const displayLines = lines.slice(0, MAX_LINES)

  const highlighted = displayLines
    .map((line) => {
      const escaped = escapeHtml(line)
      if (line.startsWith('+')) {
        return `<span class="diff-add">${escaped}</span>`
      }
      if (line.startsWith('-')) {
        return `<span class="diff-remove">${escaped}</span>`
      }
      if (line.startsWith('@@')) {
        return `<span class="diff-hunk">${escaped}</span>`
      }
      return escaped
    })
    .join('\n')

  if (truncated) {
    return highlighted + `\n<span class="diff-truncated">... (共 ${lines.length} 行，仅显示前 ${MAX_LINES} 行)</span>`
  }
  return highlighted
}

function escapeHtml(text: string): string {
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}
</script>

<template>
  <div class="tool-call-card" :class="statusClass[toolCall.status]" data-test="tool-call-card">
    <div class="tool-header" @click="showParams = !showParams">
      <div class="tool-left">
        <AppIcon
          :name="statusIcon[toolCall.status] ?? 'wrench'"
          :size="16"
          :color="statusColor[toolCall.status]"
          class="tool-icon"
        />
        <span class="tool-name" data-test="tool-name">{{ toolCall.name }}</span>
        <span class="tool-id">{{ toolCall.id }}</span>
        <span v-if="toolCall.stage && toolCall.status === 'executing'" class="tool-stage" data-test="tool-stage">{{ toolCall.stage }}</span>
        <span v-if="toolCall.riskLevel" class="tool-risk" :style="{ color: RISK_COLORS[toolCall.riskLevel] }">
          {{ RISK_LABELS[toolCall.riskLevel] }}
        </span>
      </div>
      <div class="tool-right">
        <span v-if="toolCall.duration" class="tool-duration">{{ formatDuration(toolCall.duration) }}</span>
        <AppIcon :name="showParams ? 'caret-down' : 'caret-right'" :size="12" class="tool-chevron" />
      </div>
    </div>

    <div v-if="toolCall.streamingOutput && toolCall.status === 'executing'" class="tool-streaming" data-test="tool-streaming">
      <div class="tool-section-title">实时输出</div>
      <pre class="tool-streaming-content">{{ toolCall.streamingOutput }}</pre>
    </div>

    <div v-if="showParams" class="tool-params">
      <div class="tool-section-title">参数</div>
      <pre class="tool-json">{{ JSON.stringify(toolCall.parameters, null, 2) }}</pre>
    </div>

    <div v-if="toolCall.status !== 'pending'" class="tool-result">
      <div class="tool-section-title" @click="showResult = !showResult">
        结果 <AppIcon :name="showResult ? 'caret-down' : 'caret-right'" :size="12" />
      </div>
      <div v-if="showResult && toolCall.result" class="tool-result-content">
        <div v-if="toolCall.result.success !== undefined" class="result-status">
          <AppIcon
            :name="toolCall.result.success ? 'check-circle' : 'x-circle'"
            :size="14"
            :color="toolCall.result.success ? 'var(--color-success)' : 'var(--color-error)'"
            class="result-status-icon"
          />
          {{ toolCall.result.success ? ' 成功' : ' 失败' }}
        </div>
        <div v-if="toolCall.result.error" class="result-error">{{ toolCall.result.error }}</div>

        <div v-if="toolCall.result.diff" class="diff-section">
          <div class="diff-header">变更对比</div>
          <pre class="diff-content"><code v-html="renderDiff(toolCall.result.diff as string)"></code></pre>
        </div>

        <pre v-if="toolCall.result.stdout" class="tool-json">{{ toolCall.result.stdout }}</pre>
        <pre v-if="toolCall.result.stderr" class="tool-json stderr">{{ toolCall.result.stderr }}</pre>
      </div>
    </div>
  </div>
</template>

<style scoped>
.tool-call-card {
  margin-bottom: var(--space-lg);
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-md);
  background: var(--color-bg-secondary);
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

.tool-id {
  font-family: var(--font-mono);
  font-size: 10px;
  color: var(--color-text-tertiary);
  background: var(--color-bg-tertiary);
  padding: 0 4px;
  border-radius: var(--radius-sm);
}

.tool-risk {
  font-size: 10px;
  font-weight: 600;
  padding: 1px 6px;
  border-radius: var(--radius-full);
  background: var(--color-bg-tertiary);
}

.tool-stage {
  font-size: 10px;
  font-weight: 500;
  padding: 1px 6px;
  border-radius: var(--radius-full);
  background: var(--color-accent);
  color: #fff;
  text-transform: uppercase;
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

.tool-streaming {
  border-top: 1px solid var(--color-border-light);
  padding: var(--space-sm) var(--space-md);
}

.tool-streaming-content {
  font-family: var(--font-mono);
  font-size: var(--font-size-xs);
  background: var(--color-bg-tertiary);
  padding: var(--space-sm);
  border-radius: var(--radius-sm);
  max-height: 300px;
  overflow-y: auto;
  white-space: pre-wrap;
  word-break: break-all;
  color: var(--color-text-primary);
  line-height: 1.5;
  margin: 0;
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
  background: var(--color-bg-tertiary);
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

/* Diff 展示样式 */
.diff-section {
  margin-top: var(--space-sm);
}

.diff-header {
  font-size: var(--font-size-xs);
  font-weight: 600;
  color: var(--color-text-secondary);
  margin-bottom: var(--space-xs);
}

.diff-content {
  font-family: var(--font-mono);
  font-size: var(--font-size-xs);
  background: var(--color-bg-secondary);
  padding: 0;
  border-radius: var(--radius-sm);
  overflow-x: auto;
  max-height: 400px;
  overflow-y: auto;
  line-height: 1.5;
  white-space: pre;
}

.diff-content code {
  display: block;
  padding: var(--space-sm) var(--space-md);
}

:deep(.diff-add) {
  background: rgba(0, 200, 0, 0.1);
  color: #2ecc71;
  display: block;
}

:deep(.diff-remove) {
  background: rgba(255, 0, 0, 0.1);
  color: #e74c3c;
  display: block;
}

:deep(.diff-hunk) {
  color: var(--color-accent);
  font-weight: 600;
  display: block;
}

:deep(.diff-truncated) {
  display: block;
  padding: var(--space-sm) var(--space-md);
  color: var(--color-text-tertiary);
  font-style: italic;
  text-align: center;
  border-top: 1px solid var(--color-border-light);
}
</style>