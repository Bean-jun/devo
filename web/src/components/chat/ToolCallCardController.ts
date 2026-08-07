import { ref, computed } from 'vue'
import type { ToolCall } from '@/types/tool'
import { formatDuration } from '@/utils/formatters'
import { RISK_LABELS, RISK_COLORS } from '@/utils/constants'

export interface ToolCallCardProps {
  toolCall: ToolCall
  yoloMode?: boolean
}

export function useToolCallCard(props: ToolCallCardProps) {
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

  return {
    showParams,
    showResult,
    statusIcon,
    statusColor,
    statusClass,
    renderDiff,
    RISK_LABELS,
    RISK_COLORS,
    formatDuration,
  }
}