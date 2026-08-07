import { ref, computed } from 'vue'
import type { Message } from '@/types/message'

export interface ToolCallGroupProps {
  messages: Message[]
  yoloMode?: boolean
}

export function useToolCallGroup(props: ToolCallGroupProps) {
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

  return {
    expanded,
    toolNames,
    successCount,
    failedCount,
    pendingCount,
    summaryText,
    toggle,
  }
}