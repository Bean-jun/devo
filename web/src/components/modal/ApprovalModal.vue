<script setup lang="ts">
import { computed } from 'vue'
import { useApprovalStore } from '@/stores/approval'
import { useUiStore } from '@/stores/ui'
import { RISK_LABELS, RISK_COLORS } from '@/utils/constants'

const approvalStore = useApprovalStore()
const uiStore = useUiStore()

const isOpen = computed(() => uiStore.activeModal === 'approval' && approvalStore.hasPendingApproval)
const approval = computed(() => approvalStore.currentApproval)

const riskColor = computed(() => {
  if (!approval.value) return '#ff9500'
  return RISK_COLORS[approval.value.riskLevel] ?? '#ff9500'
})

const riskLabel = computed(() => {
  if (!approval.value) return '未知'
  return RISK_LABELS[approval.value.riskLevel] ?? '未知'
})

const MAX_DIFF_LINES = 300

const highlightedDiff = computed(() => {
  const diff = approval.value?.diff
  if (!diff) return ''

  const lines = diff.split('\n')
  const truncated = lines.length > MAX_DIFF_LINES
  const displayLines = lines.slice(0, MAX_DIFF_LINES)

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
    return highlighted + `\n<span class="diff-truncated">... (共 ${lines.length} 行，仅显示前 ${MAX_DIFF_LINES} 行)</span>`
  }
  return highlighted
})

function escapeHtml(text: string): string {
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

function getDiffSummary(diff: string): string {
  const lines = diff.split('\n')
  const additions = lines.filter(l => l.startsWith('+') && !l.startsWith('+++')).length
  const deletions = lines.filter(l => l.startsWith('-') && !l.startsWith('---')).length
  return `+${additions} −${deletions} 行`
}

function handleApprove() {
  approvalStore.approve()
  uiStore.setActiveModal(null)
}

function handleReject() {
  approvalStore.reject()
  uiStore.setActiveModal(null)
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'y' || e.key === 'Y') {
    handleApprove()
  } else if (e.key === 'n' || e.key === 'N') {
    handleReject()
  } else if (e.key === 'Escape') {
    uiStore.setActiveModal(null)
  }
}
</script>

<template>
  <div
    v-if="isOpen"
    class="modal-overlay"
    @keydown="handleKeydown"
    tabindex="0"
  >
    <div class="approval-modal" data-test="approval-modal" @click.stop>
      <div class="modal-header">
        <h3 class="modal-title">审批请求</h3>
        <span
          class="risk-badge"
          :style="{ background: riskColor + '20', color: riskColor, borderColor: riskColor }"
          data-test="risk-level"
        >
          {{ riskLabel }}
        </span>
      </div>

      <div class="modal-body">
        <div class="approval-info">
          <div class="info-row">
            <span class="info-label">操作</span>
            <span class="info-value">
              <code>{{ approval?.toolName }}</code>
            </span>
          </div>
          <div v-if="approval?.filePath" class="info-row">
            <span class="info-label">文件</span>
            <span class="info-value file-path">{{ approval.filePath }}</span>
          </div>
          <div v-if="approval?.command" class="info-row">
            <span class="info-label">命令</span>
            <pre class="command-preview">{{ approval.command }}</pre>
          </div>
        </div>

        <div v-if="approval?.diff" class="diff-section">
          <div class="info-label">
            变更内容
            <span class="diff-summary">{{ getDiffSummary(approval.diff) }}</span>
          </div>
          <pre class="diff-preview"><code v-html="highlightedDiff"></code></pre>
        </div>

        <div v-if="approval?.parameters && !approval?.diff && !approval?.command" class="params-section">
          <div class="info-label">参数</div>
          <pre class="params-preview">{{ JSON.stringify(approval.parameters, null, 2) }}</pre>
        </div>
      </div>

      <div class="modal-footer">
        <button
          class="btn-approve"
          data-test="approve-button"
          @click="handleApprove"
        >
          批准 (Y)
        </button>
        <button
          class="btn-reject"
          data-test="reject-button"
          @click="handleReject"
        >
          拒绝 (N)
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  z-index: 6000;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-overlay);
  animation: fadeIn var(--transition-fast) ease;
}

.approval-modal {
  width: 600px;
  max-height: 80vh;
  background: var(--color-bg-primary);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-modal);
  animation: modalIn var(--transition-base) ease;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-lg);
  border-bottom: 1px solid var(--color-border-light);
}

.modal-title {
  font-size: var(--font-size-lg);
  font-weight: 600;
}

.risk-badge {
  font-size: var(--font-size-xs);
  font-weight: 600;
  padding: 2px 10px;
  border-radius: var(--radius-full);
  border: 1px solid;
}

.modal-body {
  flex: 1;
  overflow-y: auto;
  padding: var(--space-lg);
}

.approval-info {
  margin-bottom: var(--space-lg);
}

.info-row {
  margin-bottom: var(--space-md);
}

.info-label {
  display: block;
  font-size: var(--font-size-xs);
  font-weight: 600;
  color: var(--color-text-secondary);
  margin-bottom: var(--space-xs);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.info-value {
  font-size: var(--font-size-base);
}

.info-value code {
  font-family: var(--font-mono);
  font-size: var(--font-size-sm);
  background: var(--color-bg-tertiary);
  padding: 2px 8px;
  border-radius: var(--radius-sm);
}

.file-path {
  font-family: var(--font-mono);
  font-size: var(--font-size-sm);
  color: var(--color-accent);
}

.command-preview {
  font-family: var(--font-mono);
  font-size: var(--font-size-sm);
  background: var(--color-code-bg);
  color: #e0e0e0;
  padding: var(--space-sm) var(--space-md);
  border-radius: var(--radius-md);
  overflow-x: auto;
  white-space: pre-wrap;
  word-break: break-all;
}

.diff-section,
.params-section {
  margin-bottom: var(--space-lg);
}

.diff-preview {
  font-family: var(--font-mono);
  font-size: var(--font-size-xs);
  background: var(--color-bg-secondary);
  padding: 0;
  border-radius: var(--radius-md);
  overflow-x: auto;
  max-height: 400px;
  overflow-y: auto;
  line-height: 1.5;
  white-space: pre;
}

.diff-preview code {
  display: block;
  padding: var(--space-sm) var(--space-md);
}

.diff-summary {
  font-weight: 400;
  font-size: var(--font-size-xs);
  color: var(--color-text-tertiary);
  text-transform: none;
  margin-left: var(--space-sm);
}

.diff-preview :deep(.diff-add) {
  background: rgba(0, 200, 0, 0.1);
  color: #2ecc71;
  display: block;
}

.diff-preview :deep(.diff-remove) {
  background: rgba(255, 0, 0, 0.1);
  color: #e74c3c;
  display: block;
}

.diff-preview :deep(.diff-hunk) {
  color: var(--color-accent);
  font-weight: 600;
  display: block;
}

.diff-preview :deep(.diff-truncated) {
  display: block;
  padding: var(--space-sm) var(--space-md);
  color: var(--color-text-tertiary);
  font-style: italic;
  text-align: center;
  border-top: 1px solid var(--color-border-light);
}

.params-preview {
  font-family: var(--font-mono);
  font-size: var(--font-size-xs);
  background: var(--color-bg-secondary);
  padding: var(--space-sm) var(--space-md);
  border-radius: var(--radius-md);
  overflow-x: auto;
  max-height: 200px;
  overflow-y: auto;
  white-space: pre-wrap;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-md);
  padding: var(--space-lg);
  border-top: 1px solid var(--color-border-light);
}

.btn-approve,
.btn-reject {
  padding: var(--space-sm) var(--space-xl);
  border-radius: var(--radius-md);
  font-size: var(--font-size-base);
  font-weight: 500;
  transition: all var(--transition-fast);
}

.btn-approve {
  background: var(--color-success);
  color: white;
}

.btn-approve:hover {
  background: #2db84d;
}

.btn-reject {
  background: var(--color-error);
  color: white;
}

.btn-reject:hover {
  background: #e0352b;
}
</style>