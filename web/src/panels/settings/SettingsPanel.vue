<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useUiStore } from '@/stores/ui'
import { useSkillsStore } from '@/stores/skills'
import { useMcpStore } from '@/stores/mcp'
import { useSessionStore } from '@/stores/session'
import { API_BASE } from '@/utils/constants'
import AppIcon from '@/components/common/AppIcon.vue'

const uiStore = useUiStore()
const skillsStore = useSkillsStore()
const mcpStore = useMcpStore()
const sessionStore = useSessionStore()

type SubTab = 'project' | 'global'
const activeTab = ref<SubTab>('project')

interface ProjectConfig {
  skills: string[]
  mcp: string[]
  tool_call_limit?: number
  max_context_tokens?: number
  keep_recent?: number
}

const config = ref<ProjectConfig>({ skills: [], mcp: [] })
const configLoading = ref(false)

const toolCallLimit = ref<number | null>(null)
const maxContextTokens = ref<number | null>(null)
const keepRecent = ref<number | null>(null)

const globalToolCallLimit = ref<number | null>(null)
const globalMaxContextTokens = ref<number | null>(null)
const globalKeepRecent = ref<number | null>(null)
const globalMaxTokens = ref<number | null>(null)

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

const projectApprovalPolicy = ref<Record<string, string>>({})
const globalApprovalPolicy = ref<Record<string, string>>({})

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

onMounted(async () => {
  await fetchConfig()
  await fetchGlobalConfig()
  await skillsStore.fetchSkills()
  await mcpStore.fetchServers()
})
</script>

<template>
  <div class="settings-panel">
    <div class="settings-tabs">
      <button
        class="subtab-btn"
        :class="{ active: activeTab === 'project' }"
        @click="activeTab = 'project'"
      >项目设置</button>
      <button
        class="subtab-btn"
        :class="{ active: activeTab === 'global' }"
        @click="activeTab = 'global'"
      >全局设置</button>
    </div>

    <div v-if="activeTab === 'project'" class="settings-content">
      <div class="setting-section">
        <div class="section-title">工作目录</div>
        <div class="setting-item">
          <span class="setting-value">{{ uiStore.activeWorkspace || '无' }}</span>
        </div>
      </div>

      <div class="setting-section">
        <div class="section-title">LLM 参数</div>
        <div class="setting-item">
          <label>工具调用上限</label>
          <input
            v-model.number="toolCallLimit"
            type="number"
            min="1"
            placeholder="50"
            class="setting-input"
          />
        </div>
        <div class="setting-item">
          <label>最大上下文 Tokens</label>
          <input
            v-model.number="maxContextTokens"
            type="number"
            min="1"
            placeholder="128000"
            class="setting-input"
          />
        </div>
        <div class="setting-item">
          <label>保留最近轮数</label>
          <input
            v-model.number="keepRecent"
            type="number"
            min="1"
            placeholder="30"
            class="setting-input"
          />
        </div>
        <button class="save-btn" @click="saveProjectParams">保存参数</button>
      </div>

      <div class="setting-section">
        <div class="section-title">技能 (Skills)</div>
        <div v-if="configLoading" class="setting-hint">加载中...</div>
        <div v-else-if="config.skills.length === 0" class="setting-hint">全部启用（使用默认配置）</div>
        <div v-else class="config-list">
          <span
            v-for="name in config.skills"
            :key="name"
            class="config-tag"
            :class="{ active: skillsStore.skills.find(s => s.name === name)?.enabled }"
          >{{ name }}</span>
        </div>
      </div>

      <div class="setting-section">
        <div class="section-title">MCP 服务器</div>
        <div v-if="configLoading" class="setting-hint">加载中...</div>
        <div v-else-if="config.mcp.length === 0" class="setting-hint">无 MCP 服务器配置</div>
        <div v-else class="config-list">
          <span
            v-for="id in config.mcp"
            :key="id"
            class="config-tag"
            :class="{ active: mcpStore.servers.find(s => s.server_id === id)?.status === 'connected' }"
          >{{ id }}</span>
        </div>
      </div>

      <div class="setting-section">
        <div class="section-title">审批策略</div>
        <div class="approval-table">
          <div
            v-for="op in APPROVAL_OPERATIONS"
            :key="op.key"
            class="approval-row"
            :class="`risk-${op.risk}`"
          >
            <div class="approval-info">
              <AppIcon :name="op.icon" :size="16" class="approval-icon" />
              <span class="approval-label">{{ op.label }}</span>
              <span class="risk-badge" :class="`risk-${op.risk}`">{{ RISK_LABELS[op.risk] }}</span>
              <span v-if="isDefaultPolicy(projectApprovalPolicy, op.key)" class="approval-default">默认</span>
            </div>
            <div class="approval-pills">
              <button
                v-for="lv in APPROVAL_LEVELS"
                :key="lv.key"
                class="pill-btn"
                :class="{ active: (getProjectPolicyLevel(op.key) || op.defaultLevel) === lv.key }"
                @click="handleProjectApprovalChange(op.key, lv.key)"
              >{{ lv.short }}</button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div v-else class="settings-content">
      <div class="setting-section">
        <div class="section-title">LLM 参数（全局默认）</div>
        <div class="setting-item">
          <label>工具调用上限</label>
          <input
            v-model.number="globalToolCallLimit"
            type="number"
            min="1"
            placeholder="50"
            class="setting-input"
          />
        </div>
        <div class="setting-item">
          <label>最大上下文 Tokens</label>
          <input
            v-model.number="globalMaxContextTokens"
            type="number"
            min="1"
            placeholder="128000"
            class="setting-input"
          />
        </div>
        <div class="setting-item">
          <label>保留最近轮数</label>
          <input
            v-model.number="globalKeepRecent"
            type="number"
            min="1"
            placeholder="30"
            class="setting-input"
          />
        </div>
        <div class="setting-item">
          <label>最大输出 Tokens</label>
          <input
            v-model.number="globalMaxTokens"
            type="number"
            min="1"
            placeholder="（服务端默认值）"
            class="setting-input"
          />
        </div>
        <button class="save-btn" @click="saveGlobalParams">保存全局参数</button>
      </div>

      <div class="setting-item">
        <label>主题</label>
        <span class="setting-value">{{ uiStore.theme }}</span>
      </div>
      <div class="setting-item">
        <label>快捷键</label>
        <span class="setting-hint">Ctrl+K 命令面板</span>
      </div>

      <div class="setting-section">
        <div class="section-title">审批策略</div>
        <div class="approval-table">
          <div
            v-for="op in APPROVAL_OPERATIONS"
            :key="op.key"
            class="approval-row"
            :class="`risk-${op.risk}`"
          >
            <div class="approval-info">
              <AppIcon :name="op.icon" :size="16" class="approval-icon" />
              <span class="approval-label">{{ op.label }}</span>
              <span class="risk-badge" :class="`risk-${op.risk}`">{{ RISK_LABELS[op.risk] }}</span>
              <span v-if="isDefaultPolicy(globalApprovalPolicy, op.key)" class="approval-default">默认</span>
            </div>
            <div class="approval-pills">
              <button
                v-for="lv in APPROVAL_LEVELS"
                :key="lv.key"
                class="pill-btn"
                :class="{ active: (getGlobalPolicyLevel(op.key) || op.defaultLevel) === lv.key }"
                @click="handleGlobalApprovalChange(op.key, lv.key)"
              >{{ lv.short }}</button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.settings-panel { display: flex; flex-direction: column; height: 100%; }
.settings-tabs { display: flex; gap: 4px; padding: 8px 12px; border-bottom: 1px solid var(--color-border); }
.subtab-btn { padding: 5px 14px; border: 1px solid var(--color-border); border-radius: 4px; background: transparent; color: var(--color-text-secondary); cursor: pointer; font-size: 12px; }
.subtab-btn.active { background: var(--color-accent); border-color: var(--color-accent); color: white; }
.settings-content { flex: 1; overflow-y: auto; padding: 12px; }
.setting-section { margin-bottom: 16px; }
.section-title { font-size: 12px; font-weight: 600; color: var(--color-text-secondary); margin-bottom: 8px; text-transform: uppercase; letter-spacing: 0.5px; }
.setting-item { display: flex; justify-content: space-between; align-items: center; padding: 8px 0; border-bottom: 1px solid var(--color-border); }
.setting-item label { font-size: 13px; color: var(--color-text-primary); }
.setting-value { font-size: 12px; color: var(--color-text-secondary); font-family: var(--font-mono); }
.setting-hint { font-size: 12px; color: var(--color-text-tertiary); padding: 8px 0; }
.config-list { display: flex; flex-wrap: wrap; gap: 4px; padding: 4px 0; }
.config-tag { font-size: 11px; padding: 3px 8px; border-radius: 4px; background: var(--color-bg-tertiary); color: var(--color-text-tertiary); border: 1px solid var(--color-border); }
.config-tag.active { background: rgba(59, 130, 246, 0.12); border-color: rgba(59, 130, 246, 0.3); color: var(--color-accent); }

.approval-table {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.approval-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 14px;
  border-radius: 8px;
  background: var(--color-bg-secondary);
  transition: background 0.15s;
}

.approval-row:hover {
  background: var(--color-bg-tertiary);
}

.approval-row.risk-high {
  border-left: 3px solid rgba(239, 68, 68, 0.5);
}

.approval-row.risk-low {
  border-left: 3px solid rgba(34, 197, 94, 0.5);
}

.approval-info {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}

.approval-icon {
  font-size: 14px;
}

.approval-label {
  font-size: 13px;
  color: var(--color-text-primary);
  font-weight: 500;
}

.risk-badge {
  font-size: 10px;
  padding: 1px 6px;
  border-radius: 8px;
  font-weight: 600;
  letter-spacing: 0.3px;
}

.risk-badge.risk-high {
  background: rgba(239, 68, 68, 0.12);
  color: #ef4444;
}

.risk-badge.risk-low {
  background: rgba(34, 197, 94, 0.12);
  color: #22c55e;
}

.approval-default {
  font-size: 10px;
  padding: 1px 6px;
  border-radius: 8px;
  background: var(--color-bg-tertiary);
  color: var(--color-text-tertiary);
  font-weight: 500;
}

.approval-pills {
  display: flex;
  gap: 2px;
  flex-shrink: 0;
}

.pill-btn {
  font-size: 11px;
  padding: 4px 10px;
  border: 1px solid var(--color-border);
  background: transparent;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all 0.15s;
  outline: none;
  font-weight: 500;
}

.pill-btn:first-child {
  border-radius: 6px 0 0 6px;
}

.pill-btn:last-child {
  border-radius: 0 6px 6px 0;
}

.pill-btn:not(:first-child):not(:last-child) {
  border-radius: 0;
}

.pill-btn:hover:not(.active) {
  background: var(--color-bg-tertiary);
  color: var(--color-text-primary);
  border-color: var(--color-text-tertiary);
}

.pill-btn.active {
  background: var(--color-accent);
  border-color: var(--color-accent);
  color: #fff;
  box-shadow: 0 1px 4px rgba(59, 130, 246, 0.3);
}

.setting-input {
  width: 120px;
  padding: 4px 8px;
  border: 1px solid var(--color-border);
  border-radius: 4px;
  background: var(--color-bg-primary);
  color: var(--color-text-primary);
  font-size: 12px;
  font-family: var(--font-mono);
  text-align: right;
}

.setting-input:focus {
  outline: none;
  border-color: var(--color-accent);
}

.save-btn {
  margin-top: 8px;
  padding: 6px 16px;
  border: none;
  border-radius: 4px;
  background: var(--color-accent);
  color: #fff;
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: opacity 0.15s;
}

.save-btn:hover {
  opacity: 0.9;
}
</style>