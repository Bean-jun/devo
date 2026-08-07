import { computed } from 'vue'
import { useApprovalStore } from '@/stores/approval'
import { useUiStore } from '@/stores/ui'
import { RISK_LABELS, RISK_COLORS } from '@/utils/constants'

export function useApprovalModal() {
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
      e.stopPropagation()
      uiStore.setActiveModal(null)
    }
  }

  return {
    approvalStore,
    uiStore,
    isOpen,
    approval,
    riskColor,
    riskLabel,
    highlightedDiff,
    getDiffSummary,
    handleApprove,
    handleReject,
    handleKeydown,
  }
}