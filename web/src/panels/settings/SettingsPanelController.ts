import { ref, onMounted } from 'vue'
import { useUiStore } from '@/stores/ui'
import { useSkillsStore } from '@/stores/skills'
import { useMcpStore } from '@/stores/mcp'
import { useSessionStore } from '@/stores/session'
import { useModelStore } from '@/stores/model'
import { API_BASE } from '@/utils/constants'
import AppIcon from '@/components/common/AppIcon.vue'

type SubTab = 'project' | 'global'

interface ProjectConfig {
  skills: string[]
  mcp: string[]
  tool_call_limit?: number
  max_context_tokens?: number
  keep_recent?: number
}

interface ApprovelOp {
  key: string
  label: string
  icon: 'note' | 'pencil' | 'lightning' | 'brain' | 'wrench'
  risk: 'high' | 'low'
  defaultLevel: string
}

const APPROVAL_OPERATIONS: ApprovelOp[] = [
  { key: 'file_write_new', label: '新建文件', icon: 'note', risk: 'high', defaultLevel: 'always_ask' },
  { key: 'file_write_overwrite', label: '覆盖文件', icon: 'note', risk: 'high', defaultLevel: 'always_ask' },
  { key: 'file_edit', label: '编辑文件', icon: 'pencil', risk: 'high', defaultLevel: 'always_ask' },
  { key: 'exec_python', label: '执行Python', icon: 'lightning', risk: 'high', defaultLevel: 'always_ask' },
  { key: 'memory_update', label: '更新记忆', icon: 'brain', risk: 'low', defaultLevel: 'auto_approve' },
  { key: 'solidify_skill', label: '固化技能', icon: 'wrench', risk: 'low', defaultLevel: 'auto_approve' },
]

const APPROVAL_LEVELS: { key: string; label: string; short: string }[] = [
  { key: 'always_ask', label: '始终询问', short: '询问' },
  { key: 'session_trust', label: '本次会话信任', short: '会话' },
  { key: 'full_trust', label: '永久信任', short: '信任' },
  { key: 'auto_approve', label: '自动批准', short: '自动' },
]

const RISK_LABELS: Record<string, string> = {
  high: '高风险',
  low: '低风险',
}

export function useSettingsPanel() {
  const uiStore = useUiStore()
  const skillsStore = useSkillsStore()
  const mcpStore = useMcpStore()
  const sessionStore = useSessionStore()
  const modelStore = useModelStore()

  const activeTab = ref<SubTab>('project')

  const config = ref<ProjectConfig>({ skills: [], mcp: [] })
  const configLoading = ref(false)

  const toolCallLimit = ref<number | null>(null)
  const maxContextTokens = ref<number | null>(null)
  const keepRecent = ref<number | null>(null)

  const globalToolCallLimit = ref<number | null>(null)
  const globalMaxContextTokens = ref<number | null>(null)
  const globalKeepRecent = ref<number | null>(null)
  const globalMaxTokens = ref<number | null>(null)

  const projectApprovalPolicy = ref<Record<string, string>>({})
  const globalApprovalPolicy = ref<Record<string, string>>({})

  const showAddModelForm = ref(false)
  const testingModelId = ref<string | null>(null)
  const testResult = ref<{ id: string; success: boolean; error?: string } | null>(null)

  const teamMode = ref(false)
  const teamModeLoading = ref(false)

  function getProjectPolicyLevel(key: string): string {
    return projectApprovalPolicy.value[key] ?? ''
  }

  function getGlobalPolicyLevel(key: string): string {
    return globalApprovalPolicy.value[key] ?? ''
  }

  function isDefaultPolicy(policy: Record<string, string>, key: string): boolean {
    const op = APPROVAL_OPERATIONS.find(o => o.key === key)
    return !policy[key] || policy[key] === (op?.defaultLevel ?? '')
  }

  async function handleProjectApprovalChange(key: string, level: string) {
    if (getProjectPolicyLevel(key) === level) return
    const prev = { ...projectApprovalPolicy.value }
    try {
      projectApprovalPolicy.value = { ...projectApprovalPolicy.value, [key]: level }
      await sessionStore.setProjectApprovalPolicy(projectApprovalPolicy.value)
      uiStore.showToast('success', '项目审批策略已更新')
    } catch {
      projectApprovalPolicy.value = prev
      uiStore.showToast('error', '保存失败')
    }
  }

  async function handleGlobalApprovalChange(key: string, level: string) {
    if (getGlobalPolicyLevel(key) === level) return
    const prev = { ...globalApprovalPolicy.value }
    try {
      globalApprovalPolicy.value = { ...globalApprovalPolicy.value, [key]: level }
      await sessionStore.setGlobalApprovalPolicy(globalApprovalPolicy.value)
      uiStore.showToast('success', '全局审批策略已更新')
    } catch {
      globalApprovalPolicy.value = prev
      uiStore.showToast('error', '保存失败')
    }
  }

  async function fetchConfig() {
    configLoading.value = true
    try {
      const res = await fetch(`${API_BASE}/project/config`)
      if (res.ok) {
        const data = await res.json()
        config.value = { skills: data.skills || [], mcp: data.mcp || [] }
        projectApprovalPolicy.value = data.approval_policy || {}
        toolCallLimit.value = data.tool_call_limit ?? null
        maxContextTokens.value = data.max_context_tokens ?? null
        keepRecent.value = data.keep_recent ?? null
      }
    } catch {
      // ignore
    } finally {
      configLoading.value = false
    }
  }

  async function saveProjectParams() {
    try {
      const body: Record<string, unknown> = {
        skills: config.value.skills,
        mcp: config.value.mcp,
        approval_policy: projectApprovalPolicy.value,
      }
      if (toolCallLimit.value != null) body.tool_call_limit = toolCallLimit.value
      if (maxContextTokens.value != null) body.max_context_tokens = maxContextTokens.value
      if (keepRecent.value != null) body.keep_recent = keepRecent.value

      await fetch(`${API_BASE}/project/config`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })
      uiStore.showToast('success', '项目设置已保存')
    } catch {
      uiStore.showToast('error', '保存失败')
    }
  }

  async function fetchGlobalConfig() {
    try {
      const res = await fetch(`${API_BASE}/global/config`)
      if (res.ok) {
        const data = await res.json()
        globalToolCallLimit.value = data.tool_call_limit ?? null
        globalMaxContextTokens.value = data.max_context_tokens ?? null
        globalKeepRecent.value = data.keep_recent ?? null
        globalMaxTokens.value = data.llm?.max_tokens ?? null
        globalApprovalPolicy.value = data.approval_policy || {}
        teamMode.value = data.team_mode ?? false
      }
    } catch {
      // ignore
    }
  }

  async function saveGlobalParams() {
    try {
      const body: Record<string, unknown> = {}
      if (globalToolCallLimit.value != null) body.tool_call_limit = globalToolCallLimit.value
      if (globalMaxContextTokens.value != null) body.max_context_tokens = globalMaxContextTokens.value
      if (globalKeepRecent.value != null) body.keep_recent = globalKeepRecent.value
      if (globalMaxTokens.value != null) {
        body.llm = { max_tokens: globalMaxTokens.value }
      }

      await fetch(`${API_BASE}/global/config`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })
      uiStore.showToast('success', '全局设置已保存')
    } catch {
      uiStore.showToast('error', '保存失败')
    }
  }

  async function handleModelActivate(id: string) {
    try {
      await modelStore.activateModel(id)
      uiStore.showToast('success', '模型已切换')
    } catch (e: any) {
      uiStore.showToast('error', e.message || '切换失败')
    }
  }

  async function handleModelDelete(id: string) {
    if (!confirm('确定要删除此模型吗？')) return
    try {
      await modelStore.deleteModel(id)
      uiStore.showToast('success', '模型已删除')
    } catch (e: any) {
      uiStore.showToast('error', e.message || '删除失败')
    }
  }

  async function handleModelTest(id: string) {
    testingModelId.value = id
    testResult.value = null
    try {
      const result = await modelStore.testModel(id)
      testResult.value = { id, success: result.success, error: result.error }
    } catch (e: any) {
      testResult.value = { id, success: false, error: e.message || '测试失败' }
    } finally {
      testingModelId.value = null
    }
  }

  function openAddModelForm() {
    showAddModelForm.value = true
  }

  function onModelAdded() {
    showAddModelForm.value = false
  }

  async function handleTeamModeToggle() {
    teamModeLoading.value = true
    const prev = teamMode.value
    try {
      await sessionStore.setTeamMode(!prev)
      teamMode.value = !prev
      uiStore.showToast('success', teamMode.value ? 'Team Mode 已开启' : 'Team Mode 已关闭')
    } catch {
      teamMode.value = prev
      uiStore.showToast('error', '设置 Team Mode 失败')
    } finally {
      teamModeLoading.value = false
    }
  }

  onMounted(async () => {
    await fetchConfig()
    await fetchGlobalConfig()
    await skillsStore.fetchSkills()
    await mcpStore.fetchServers()
    await modelStore.fetchModels()
  })

  return {
    activeTab,
    config,
    configLoading,
    uiStore,
    skillsStore,
    mcpStore,
    toolCallLimit,
    maxContextTokens,
    keepRecent,
    globalToolCallLimit,
    globalMaxContextTokens,
    globalKeepRecent,
    globalMaxTokens,
    APPROVAL_OPERATIONS,
    APPROVAL_LEVELS,
    RISK_LABELS,
    projectApprovalPolicy,
    globalApprovalPolicy,
    getProjectPolicyLevel,
    getGlobalPolicyLevel,
    isDefaultPolicy,
    handleProjectApprovalChange,
    handleGlobalApprovalChange,
    saveProjectParams,
    saveGlobalParams,
    modelStore,
    showAddModelForm,
    testingModelId,
    testResult,
    handleModelActivate,
    handleModelDelete,
    handleModelTest,
    openAddModelForm,
    onModelAdded,
    teamMode,
    teamModeLoading,
    handleTeamModeToggle,
  }
}