import { ref, computed } from 'vue'
import { useReasoningStore, type ReasoningEffortLevel } from '@/stores/reasoning'

export function useReasoningEffortToggle() {
  const reasoningStore = useReasoningStore()

  const open = ref(false)

  interface EffortOption {
    value: 'off' | ReasoningEffortLevel
    label: string
    icon: 'prohibit' | 'brain'
  }

  const EFFORT_OPTIONS: EffortOption[] = [
    { value: 'off', label: '关闭', icon: 'prohibit' },
    { value: 'low', label: '低', icon: 'brain' },
    { value: 'medium', label: '中', icon: 'brain' },
    { value: 'high', label: '高', icon: 'brain' },
  ]

  const currentLabel = computed(() => {
    if (!reasoningStore.enableReasoning) return '思考'
    const opt = EFFORT_OPTIONS.find(o => o.value === reasoningStore.reasoningEffort)
    return `思考 · ${opt?.label ?? ''}`
  })

  function isSelected(opt: EffortOption): boolean {
    if (opt.value === 'off') {
      return !reasoningStore.enableReasoning
    }
    return reasoningStore.enableReasoning && reasoningStore.reasoningEffort === opt.value
  }

  function select(opt: EffortOption) {
    if (opt.value === 'off') {
      reasoningStore.setEnableReasoning(false)
    } else {
      reasoningStore.setEnableReasoning(true)
      reasoningStore.setReasoningEffort(opt.value as ReasoningEffortLevel)
    }
    open.value = false
  }

  function toggle() {
    open.value = !open.value
  }

  function onBlur(e: FocusEvent) {
    const target = e.relatedTarget as HTMLElement | null
    if (!target || !target.closest('.reasoning-toggle')) {
      open.value = false
    }
  }

  return { reasoningStore, open, EFFORT_OPTIONS, currentLabel, isSelected, select, toggle, onBlur }
}