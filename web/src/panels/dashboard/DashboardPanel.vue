<script setup lang="ts">
import AppIcon from '@/components/common/AppIcon.vue'
import { useDashboardPanel } from './DashboardPanelController'

const {
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
} = useDashboardPanel()
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
            <div class="chart-header">
              <span class="chart-label">步骤消耗</span>
              <div class="chart-legend">
                <span class="legend-item"><span class="legend-dot legend-input" />输入</span>
                <span class="legend-item"><span class="legend-dot legend-output" />输出</span>
              </div>
            </div>
            <div class="bar-chart">
              <div
                v-for="(s, idx) in sessionSteps"
                :key="s.step_seq"
                class="bar-item"
                :style="{ animationDelay: idx * 60 + 'ms' }"
              >
                <span class="bar-label">步骤 {{ s.step_seq }}</span>
                <div class="bar-track">
                  <div class="bar-tooltip">
                    <span class="tooltip-row"><span class="tooltip-dot ti-dot" />输入 {{ formatTokens(s.input_tokens) }}</span>
                    <span class="tooltip-row"><span class="tooltip-dot to-dot" />输出 {{ formatTokens(s.output_tokens) }}</span>
                  </div>
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
                <span class="bar-value">{{ formatTokens(s.input_tokens + s.output_tokens) }}</span>
              </div>
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
            <div class="chart-header">
              <span class="chart-label">{{ groupBy === 'date' ? '每日消耗' : '会话消耗' }}</span>
              <div class="chart-legend">
                <span class="legend-item"><span class="legend-dot legend-input" />输入</span>
                <span class="legend-item"><span class="legend-dot legend-output" />输出</span>
              </div>
            </div>
            <div class="bar-chart">
              <div
                v-for="(g, idx) in projectGroups"
                :key="g.key"
                class="bar-item"
                :style="{ animationDelay: idx * 60 + 'ms' }"
              >
                <span class="bar-label">{{ g.key }}</span>
                <div class="bar-track">
                  <div class="bar-tooltip">
                    <span class="tooltip-row"><span class="tooltip-dot ti-dot" />输入 {{ formatTokens(g.input_tokens) }}</span>
                    <span class="tooltip-row"><span class="tooltip-dot to-dot" />输出 {{ formatTokens(g.output_tokens) }}</span>
                  </div>
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
                <span class="bar-value">{{ formatTokens(g.total_tokens) }}</span>
              </div>
            </div>
          </div>

          <div v-else class="card-empty">暂无数据</div>
        </template>
      </div>
    </div>
  </div>
</template>

<style scoped src="./DashboardPanel.css">
</style>