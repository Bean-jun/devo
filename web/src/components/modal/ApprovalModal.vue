<script setup lang="ts">
import { useApprovalModal } from './ApprovalModalController'

const {
  approvalStore,
  isOpen,
  approval,
  riskColor,
  riskLabel,
  highlightedDiff,
  getDiffSummary,
  handleApprove,
  handleReject,
  handleKeydown,
} = useApprovalModal()
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

<style scoped src="./ApprovalModal.css">
</style>