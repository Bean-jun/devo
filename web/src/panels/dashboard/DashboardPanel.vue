<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { API_BASE } from '@/utils/constants'

const summary = ref({ input: 0, output: 0, total: 0 })
const groups = ref<Array<{ key: string; input_tokens: number; output_tokens: number; total_tokens: number; estimated_cost: string }>>([])
const isLoading = ref(false)

async function fetchDashboard() {
  isLoading.value = true
  try {
    const res = await fetch(`${API_BASE}/usage/stats`)
    if (res.ok) {
      const data = await res.json()
      summary.value = data.summary || { input: 0, output: 0, total: 0 }
      groups.value = data.groups || []
    }
  } catch {}
  isLoading.value = false
}

onMounted(() => { fetchDashboard() })
</script>

<template>
  <div class="dashboard-panel">
    <div class="dash-header">
      <span class="dash-title">Dashboard</span>
      <button class="reload-btn" :class="{ spinning: isLoading }" :disabled="isLoading" @click="fetchDashboard" title="刷新">
        ↻
      </button>
    </div>
    <div v-if="isLoading" class="dash-loading">加载中...</div>
    <div v-else class="dash-cards">
      <div class="dash-card">
        <div class="card-label">总计用量</div>
        <div class="card-stats">
          <div class="stat"><span class="stat-label">输入</span><span class="stat-value">{{ summary.input.toLocaleString() }}</span></div>
          <div class="stat"><span class="stat-label">输出</span><span class="stat-value">{{ summary.output.toLocaleString() }}</span></div>
          <div class="stat"><span class="stat-label">合计</span><span class="stat-value">{{ summary.total.toLocaleString() }}</span></div>
        </div>
      </div>
      <div v-if="groups.length > 0" class="dash-card">
        <div class="card-label">按日期分组</div>
        <div class="groups-list">
          <div v-for="g in groups" :key="g.key" class="group-item">
            <span class="group-key">{{ g.key }}</span>
            <span class="group-tokens">{{ g.total_tokens.toLocaleString() }} tokens</span>
          </div>
        </div>
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
.dash-loading { text-align: center; color: var(--color-text-tertiary); padding: 24px; }
.dash-cards { display: flex; flex-direction: column; gap: 12px; }
.dash-card { border: 1px solid var(--color-border); border-radius: 8px; padding: 14px; }
.card-label { font-size: 12px; font-weight: 600; color: var(--color-text-secondary); margin-bottom: 10px; }
.card-stats { display: flex; flex-direction: column; gap: 6px; }
.stat { display: flex; justify-content: space-between; font-size: 13px; }
.stat-label { color: var(--color-text-tertiary); }
.stat-value { font-weight: 600; color: var(--color-text-primary); font-family: var(--font-mono); }
</style>