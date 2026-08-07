import { ref, computed, watch, onMounted } from 'vue'
import { useSessionStore } from '@/stores/session'
import { API_BASE } from '@/utils/constants'
import AppIcon from '@/components/common/AppIcon.vue'

type UsageGroup = { key: string; input_tokens: number; output_tokens: number; total_tokens: number }
type UsageStep = { step_seq: number; input_tokens: number; output_tokens: number; created_at: string }

function formatTokens(n: number): string {
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + 'M'
  if (n >= 1_000) return (n / 1_000).toFixed(1) + 'k'
  return String(n)
}

export function useDashboardPanel() {
  const sessionStore = useSessionStore()

  const sessionUsage = ref({ input: 0, output: 0, total: 0, compression_count: 0 })
  const sessionSteps = ref<UsageStep[]>([])
  const projectSummary = ref({ input: 0, output: 0, total: 0 })
  const projectGroups = ref<UsageGroup[]>([])
  const isSessionLoading = ref(false)
  const isProjectLoading = ref(false)
  const groupBy = ref<'date' | 'session'>('date')

  const currentSessionId = computed(() => sessionStore.currentSession?.id)
  const currentWorkspace = computed(() => sessionStore.currentSession?.workingDirectory || sessionStore.workingDirectory)

  const maxStepTokens = computed(() => {
    if (sessionSteps.value.length === 0) return 1
    return Math.max(...sessionSteps.value.map(s => s.input_tokens + s.output_tokens))
  })

  const maxGroupTokens = computed(() => {
    if (projectGroups.value.length === 0) return 1
    return Math.max(...projectGroups.value.map(g => g.total_tokens))
  })

  async function fetchSessionUsage() {
    const sid = currentSessionId.value
    if (!sid) return
    isSessionLoading.value = true
    try {
      const res = await fetch(`${API_BASE}/sessions/${sid}/usage`)
      if (res.ok) {
        const data = await res.json()
        sessionUsage.value = {
          input: data.total_input_tokens ?? 0,
          output: data.total_output_tokens ?? 0,
          total: data.total_tokens ?? 0,
          compression_count: data.compression_count ?? 0,
        }
        sessionSteps.value = data.steps ?? []
      }
    } catch {}
    isSessionLoading.value = false
  }

  async function fetchProjectUsage() {
    const ws = currentWorkspace.value
    if (!ws) return
    isProjectLoading.value = true
    try {
      const params = new URLSearchParams({ project: ws, group_by: groupBy.value })
      const res = await fetch(`${API_BASE}/usage/stats?${params}`)
      if (res.ok) {
        const data = await res.json()
        projectSummary.value = data.summary || { input: 0, output: 0, total: 0 }
        projectGroups.value = data.groups || []
      }
    } catch {}
    isProjectLoading.value = false
  }

  function fetchAll() {
    fetchSessionUsage()
    fetchProjectUsage()
  }

  function stepBarWidth(s: UsageStep): number {
    if (maxStepTokens.value === 0) return 0
    return ((s.input_tokens + s.output_tokens) / maxStepTokens.value) * 100
  }

  function stepInputPct(s: UsageStep): number {
    const total = s.input_tokens + s.output_tokens
    if (total === 0) return 0
    return (s.input_tokens / total) * 100
  }

  function groupBarWidth(g: UsageGroup): number {
    if (maxGroupTokens.value === 0) return 0
    return (g.total_tokens / maxGroupTokens.value) * 100
  }

  function groupInputPct(g: UsageGroup): number {
    if (g.total_tokens === 0) return 0
    return (g.input_tokens / g.total_tokens) * 100
  }

  watch(currentSessionId, () => {
    if (currentSessionId.value) {
      fetchAll()
    }
  })

  watch(groupBy, () => {
    fetchProjectUsage()
  })

  onMounted(() => { fetchAll() })

  return {
    currentSessionId,
    currentWorkspace,
    sessionUsage,
    sessionSteps,
    projectSummary,
    projectGroups,
    isSessionLoading,
    isProjectLoading,
    groupBy,
    fetchAll,
    formatTokens,
    stepBarWidth,
    stepInputPct,
    groupBarWidth,
    groupInputPct,
  }
}