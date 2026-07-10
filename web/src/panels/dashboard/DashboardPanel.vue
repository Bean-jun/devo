<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useSessionStore } from '@/stores/session'
import { API_BASE } from '@/utils/constants'
import AppIcon from '@/components/common/AppIcon.vue'

const sessionStore = useSessionStore()

type UsageGroup = { key: string; input_tokens: number; output_tokens: number; total_tokens: number }
type UsageStep = { step_seq: number; input_tokens: number; output_tokens: number; created_at: string }

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

function formatTokens(n: number): string {
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + 'M'
  if (n >= 1_000) return (n / 1_000).toFixed(1) + 'k'
  return String(n)
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
</script>

<template>
  <div class="dashboard-panel">
    <div class="dash-header">
      <span class="dash-title">Dashboard</span>
      <button
        class="reload-btn"
        :class="{ spinning: isSessionLoading || isProjectLoading }"
        :disabled="isSessionLoading || isProjectLoading"
        @click="fetchAll"
        title="刷新"
      ><AppIcon name="arrow-clockwise" :size="16" /></button>
    </div>

    <div class="dash-cards">
      <div class="dash-card">
        <div class="card-label">当前会话</div>
        <div v-if="!currentSessionId" class="card-empty">暂无活跃会话</div>
        <template v-else>
          <div class="card-stats">
            <div class="stat">
              <span class="stat-label">输入</span>
              <span class="stat-value stat-input">{{ formatTokens(sessionUsage.input) }}</span>
            </div>
            <div class="stat">
              <span class="stat-label">输出</span>
              <span class="stat-value stat-output">{{ formatTokens(sessionUsage.output) }}</span>
            </div>
            <div class="stat">
              <span class="stat-label">合计</span>
              <span class="stat-value stat-total">{{ formatTokens(sessionUsage.total) }}</span>
            </div>
            <div class="stat">
              <span class="stat-label">压缩次数</span>
              <span class="stat-value">{{ sessionUsage.compression_count }}</span>
            </div>
          </div>

          <div v-if="sessionSteps.length > 0" class="chart-section">
            <div class="chart-label">步骤消耗</div>
            <div class="bar-chart">
              <div
                v-for="s in sessionSteps"
                :key="s.step_seq"
                class="bar-item"
                :title="`步骤 ${s.step_seq}: 输入 ${formatTokens(s.input_tokens)} / 输出 ${formatTokens(s.output_tokens)}`"
              >
                <div class="bar-track">
                  <div class="bar-fill" :style="{ width: stepBarWidth(s) + '%' }">
                    <div
                      class="bar-segment bar-input"
                      :style="{ width: stepInputPct(s) + '%' }"
                    />
                    <div
                      class="bar-segment bar-output"
                      :style="{ width: (100 - stepInputPct(s)) + '%' }"
                    />
                  </div>
                </div>
                <span class="bar-label">{{ s.step_seq }}</span>
              </div>
            </div>
            <div class="chart-legend">
              <span class="legend-item"><span class="legend-dot legend-input" />输入</span>
              <span class="legend-item"><span class="legend-dot legend-output" />输出</span>
            </div>
          </div>
        </template>
      </div>

      <div class="dash-card">
        <div class="card-label-row">
          <span class="card-label">项目消耗</span>
          <div class="group-by-switch">
            <button
              :class="{ active: groupBy === 'date' }"
              @click="groupBy = 'date'"
            >按日期</button>
            <button
              :class="{ active: groupBy === 'session' }"
              @click="groupBy = 'session'"
            >按会话</button>
          </div>
        </div>
        <div v-if="!currentWorkspace" class="card-empty">暂未绑定工作区</div>
        <template v-else>
          <div class="card-stats">
            <div class="stat">
              <span class="stat-label">输入</span>
              <span class="stat-value stat-input">{{ formatTokens(projectSummary.input) }}</span>
            </div>
            <div class="stat">
              <span class="stat-label">输出</span>
              <span class="stat-value stat-output">{{ formatTokens(projectSummary.output) }}</span>
            </div>
            <div class="stat">
              <span class="stat-label">合计</span>
              <span class="stat-value stat-total">{{ formatTokens(projectSummary.total) }}</span>
            </div>
          </div>

          <div v-if="projectGroups.length > 0" class="chart-section">
            <div class="bar-chart">
              <div
                v-for="g in projectGroups"
                :key="g.key"
                class="bar-item"
                :title="`${g.key}: 输入 ${formatTokens(g.input_tokens)} / 输出 ${formatTokens(g.output_tokens)}`"
              >
                <div class="bar-track">
                  <div class="bar-fill" :style="{ width: groupBarWidth(g) + '%' }">
                    <div
                      class="bar-segment bar-input"
                      :style="{ width: groupInputPct(g) + '%' }"
                    />
                    <div
                      class="bar-segment bar-output"
                      :style="{ width: (100 - groupInputPct(g)) + '%' }"
                    />
                  </div>
                </div>
                <span class="bar-label">{{ g.key }}</span>
                <span class="bar-value">{{ formatTokens(g.total_tokens) }}</span>
              </div>
            </div>
            <div class="chart-legend">
              <span class="legend-item"><span class="legend-dot legend-input" />输入</span>
              <span class="legend-item"><span class="legend-dot legend-output" />输出</span>
            </div>
          </div>

          <div v-else class="card-empty">暂无数据</div>
        </template>
      </div>
    </div>
  </div>
</template>

<style scoped>
.dashboard-panel { padding: 12px; }
.dash-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 12px; }
.dash-title { font-size: 14px; font-weight: 600; color: var(--color-text-primary); }
.reload-btn { width: 28px; height: 28px; border: 1px solid var(--color-border); border-radius: 4px; background: transparent; color: var(--color-text-secondary); cursor: pointer; font-size: 14px; display: flex; align-items: center; justify-content: center; transition: all var(--transition-fast); flex-shrink: 0; }
.reload-btn:hover:not(:disabled) { background: var(--color-bg-hover); color: var(--color-accent); border-color: var(--color-accent); }
.reload-btn:disabled { opacity: 0.4; cursor: not-allowed; }
.reload-btn.spinning { animation: spin 0.8s linear infinite; }

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.dash-cards { display: flex; flex-direction: column; gap: 12px; }
.dash-card { border: 1px solid var(--color-border); border-radius: 8px; padding: 14px; }

.card-label { font-size: 12px; font-weight: 600; color: var(--color-text-secondary); }
.card-label-row { display: flex; align-items: center; justify-content: space-between; margin-bottom: 10px; }

.group-by-switch { display: flex; gap: 2px; background: var(--color-bg-tertiary); border-radius: 4px; padding: 2px; }
.group-by-switch button { padding: 2px 8px; border: none; border-radius: 3px; background: transparent; color: var(--color-text-tertiary); font-size: 11px; cursor: pointer; transition: all var(--transition-fast); }
.group-by-switch button.active { background: var(--color-bg-primary); color: var(--color-accent); font-weight: 600; box-shadow: 0 1px 2px rgba(0,0,0,0.1); }
.group-by-switch button:hover:not(.active) { color: var(--color-text-secondary); }

.card-stats { display: flex; flex-direction: column; gap: 6px; margin-bottom: 8px; }
.stat { display: flex; justify-content: space-between; font-size: 13px; }
.stat-label { color: var(--color-text-tertiary); }
.stat-value { font-weight: 600; color: var(--color-text-primary); font-family: var(--font-mono); }
.stat-input { color: var(--color-accent); }
.stat-output { color: #34c759; }
.stat-total { color: var(--color-text-primary); }

.card-empty { text-align: center; color: var(--color-text-tertiary); padding: 16px 0; font-size: 12px; }

.chart-section { margin-top: 10px; }
.chart-label { font-size: 11px; font-weight: 600; color: var(--color-text-tertiary); margin-bottom: 6px; }

.bar-chart { display: flex; flex-direction: column; gap: 4px; }
.bar-item { display: flex; align-items: center; gap: 6px; }
.bar-track { flex: 1; height: 14px; background: var(--color-bg-tertiary); border-radius: 3px; overflow: hidden; }
.bar-fill { height: 100%; display: flex; border-radius: 3px; transition: width 0.3s ease; min-width: 2px; }
.bar-segment { height: 100%; }
.bar-input { background: var(--color-accent); opacity: 0.85; }
.bar-output { background: #34c759; opacity: 0.7; }
.bar-label { font-size: 10px; color: var(--color-text-tertiary); min-width: 28px; text-align: right; font-family: var(--font-mono); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.bar-value { font-size: 10px; color: var(--color-text-secondary); min-width: 40px; text-align: right; font-family: var(--font-mono); }

.chart-legend { display: flex; gap: 12px; margin-top: 6px; }
.legend-item { display: flex; align-items: center; gap: 4px; font-size: 10px; color: var(--color-text-tertiary); }
.legend-dot { width: 8px; height: 8px; border-radius: 2px; flex-shrink: 0; }
.legend-input { background: var(--color-accent); opacity: 0.85; }
.legend-output { background: #34c759; opacity: 0.7; }
</style>