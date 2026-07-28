import { defineStore } from 'pinia'
import { ref } from 'vue'
import { API_BASE } from '@/utils/constants'

export type ReasoningEffortLevel = 'low' | 'medium' | 'high'

export const useReasoningStore = defineStore('reasoning', () => {
  const enableReasoning = ref(false)
  const reasoningEffort = ref<ReasoningEffortLevel>('medium')
  const loading = ref(false)

  async function fetchConfig() {
    try {
      const res = await fetch(`${API_BASE}/global/config`)
      if (!res.ok) return
      const data = await res.json()
      if (data.llm) {
        enableReasoning.value = data.llm.enable_reasoning ?? false
        reasoningEffort.value = data.llm.reasoning_effort ?? 'medium'
      }
    } catch {
      // 静默失败，使用默认值
    }
  }

  async function saveConfig() {
    loading.value = true
    try {
      await fetch(`${API_BASE}/global/config`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          llm: {
            enable_reasoning: enableReasoning.value,
            reasoning_effort: reasoningEffort.value,
          },
        }),
      })
    } finally {
      loading.value = false
    }
  }

  async function setEnableReasoning(val: boolean) {
    enableReasoning.value = val
    await saveConfig()
  }

  async function setReasoningEffort(val: ReasoningEffortLevel) {
    reasoningEffort.value = val
    await saveConfig()
  }

  return {
    enableReasoning,
    reasoningEffort,
    loading,
    fetchConfig,
    setEnableReasoning,
    setReasoningEffort,
  }
})