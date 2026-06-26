<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { API_BASE } from '@/utils/constants'

const todayUsage = ref({ input: 0, output: 0 })
const monthUsage = ref({ input: 0, output: 0 })
const isLoading = ref(false)

async function fetchDashboard() {
  isLoading.value = true
  try {
    const res = await fetch(`${API_BASE}/stats/dashboard`)
    if (res.ok) {
      const data = await res.json()
      todayUsage.value = data.today || { input: 0, output: 0 }
      monthUsage.value = data.month || { input: 0, output: 0 }
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
      <button class="refresh-btn" @click="fetchDashboard" :disabled="isLoading">🔄</button>
    </div>
    <div v-if="isLoading" class="dash-loading">加载中...</div>
    <div v-else class="dash-cards">
      <div class="dash-card">
        <div class="card-label">今日用量</div>
        <div class="card-stats">
          <div class="stat"><span class="stat-label">输入</span><span class="stat-value">{{ todayUsage.input.toLocaleString() }}</span></div>
          <div class="stat"><span class="stat-label">输出</span><span class="stat-value">{{ todayUsage.output.toLocaleString() }}</span></div>
        </div>
      </div>
      <div class="dash-card">
        <div class="card-label">本月用量</div>
        <div class="card-stats">
          <div class="stat"><span class="stat-label">输入</span><span class="stat-value">{{ monthUsage.input.toLocaleString() }}</span></div>
          <div class="stat"><span class="stat-label">输出</span><span class="stat-value">{{ monthUsage.output.toLocaleString() }}</span></div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.dashboard-panel { padding: 12px; }
.dash-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 12px; }
.dash-title { font-size: 14px; font-weight: 600; color: var(--color-text-primary); }
.refresh-btn { background: none; border: none; cursor: pointer; font-size: 14px; }
.dash-loading { text-align: center; color: var(--color-text-tertiary); padding: 24px; }
.dash-cards { display: flex; flex-direction: column; gap: 12px; }
.dash-card { border: 1px solid var(--color-border); border-radius: 8px; padding: 14px; }
.card-label { font-size: 12px; font-weight: 600; color: var(--color-text-secondary); margin-bottom: 10px; }
.card-stats { display: flex; flex-direction: column; gap: 6px; }
.stat { display: flex; justify-content: space-between; font-size: 13px; }
.stat-label { color: var(--color-text-tertiary); }
.stat-value { font-weight: 600; color: var(--color-text-primary); font-family: var(--font-mono); }
</style>