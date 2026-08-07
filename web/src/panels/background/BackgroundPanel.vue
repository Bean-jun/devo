<script setup lang="ts">
import AppIcon from '@/components/common/AppIcon.vue'
import { useBackgroundPanel } from './BackgroundPanelController'

const {
  expandedPids,
  stoppingPids,
  error,
  processes,
  runningCount,
  hasProcesses,
  toggleExpand,
  isExpanded,
  isStopping,
  statusLabel,
  statusColor,
  setStdoutRef,
  handleStop,
  handleDismiss,
  refresh,
  formatRelativeTime,
} = useBackgroundPanel()
</script>

<template>
  <div class="background-panel" data-test="background-panel">
    <div class="panel-toolbar">
      <div class="panel-title">
        后台进程
        <span v-if="runningCount > 0" class="running-badge" data-test="running-count">
          {{ runningCount }} 运行中
        </span>
      </div>
      <button class="icon-btn" title="刷新" @click="refresh">
        <AppIcon name="arrow-clockwise" :size="14" />
      </button>
    </div>

    <div v-if="error" class="panel-error" data-test="panel-error">{{ error }}</div>

    <div v-if="!hasProcesses" class="empty-state" data-test="empty-state">
      <AppIcon name="hourglass" :size="32" class="empty-icon" />
      <p>当前没有后台进程</p>
      <p class="hint">exec_python 的 background 模式启动的进程会显示在这里</p>
    </div>

    <div v-else class="process-list" data-test="process-list">
      <div
        v-for="p in processes"
        :key="p.pid"
        class="process-card"
        :class="`status-${p.status}`"
      >
        <div class="card-header" @click="toggleExpand(p.pid)">
          <div class="card-left">
            <AppIcon
              :name="p.status === 'running' ? 'arrow-clockwise' : 'stop'"
              :size="14"
              :color="statusColor(p)"
              class="status-icon"
            />
            <span class="pid">PID {{ p.pid }}</span>
            <span class="status-tag" :style="{ color: statusColor(p) }">
              {{ statusLabel(p) }}
            </span>
          </div>
          <div class="card-right">
            <span class="started-at">{{ formatRelativeTime(p.startedAt.toISOString()) }}</span>
            <AppIcon
              :name="isExpanded(p.pid) ? 'caret-down' : 'caret-right'"
              :size="12"
              class="chevron"
            />
          </div>
        </div>

        <div class="card-meta">
          <span class="cmd" :title="p.cmd">{{ p.cmd || '(无命令)' }}</span>
        </div>

        <div v-if="isExpanded(p.pid)" class="card-body" data-test="card-body">
          <div class="output-section">
            <div class="output-label">stdout</div>
            <pre
              :ref="(el) => setStdoutRef(p.pid, el as HTMLElement | null)"
              class="output-content"
              data-test="stdout-output"
            >{{ p.stdout || '(无输出)' }}</pre>
            <div v-if="p.stderr" class="output-label stderr-label">stderr</div>
            <pre
              v-if="p.stderr"
              class="output-content stderr"
              data-test="stderr-output"
            >{{ p.stderr }}</pre>
          </div>

          <div class="card-actions">
            <button
              v-if="p.status === 'running'"
              class="action-btn danger"
              :disabled="isStopping(p.pid)"
              @click="handleStop(p)"
              data-test="stop-btn"
            >
              <AppIcon name="stop" :size="12" />
              {{ isStopping(p.pid) ? '停止中...' : '停止' }}
            </button>
            <button
              v-else
              class="action-btn"
              @click="handleDismiss(p.pid)"
              data-test="dismiss-btn"
            >
              <AppIcon name="x" :size="12" />
              清除
            </button>
          </div>

          <div v-if="p.stopError" class="stop-error" data-test="stop-error">
            {{ p.stopError }}
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped src="./BackgroundPanel.css">
</style>