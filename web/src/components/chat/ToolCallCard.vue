<script setup lang="ts">
import AppIcon from '@/components/common/AppIcon.vue'
import HljsPreview from '@/components/editor/HighlightPreview.vue'
import type { ToolCall } from '@/types/tool'
import { useToolCallCard } from './ToolCallCardController'

const props = defineProps<{
  toolCall: ToolCall
  yoloMode?: boolean
}>()

const {
  showParams,
  showResult,
  statusIcon,
  statusColor,
  statusClass,
  renderDiff,
  RISK_LABELS,
  RISK_COLORS,
  formatDuration,
} = useToolCallCard(props)
</script>

<template>
  <div class="tool-call-card" :class="statusClass[toolCall.status]" data-test="tool-call-card">
    <div class="tool-header" @click="showParams = !showParams">
      <div class="tool-left">
        <AppIcon
          :name="statusIcon[toolCall.status] ?? 'wrench'"
          :size="16"
          :color="statusColor[toolCall.status]"
          class="tool-icon"
        />
        <span class="tool-name" data-test="tool-name">{{ toolCall.name }}</span>
        <span class="tool-id">{{ toolCall.id }}</span>
        <span v-if="toolCall.stage && toolCall.status === 'executing'" class="tool-stage" data-test="tool-stage">{{ toolCall.stage }}</span>
        <span v-if="toolCall.riskLevel" class="tool-risk" :style="{ color: RISK_COLORS[toolCall.riskLevel] }">
          {{ RISK_LABELS[toolCall.riskLevel] }}
        </span>
      </div>
      <div class="tool-right">
        <span v-if="toolCall.duration" class="tool-duration">{{ formatDuration(toolCall.duration) }}</span>
        <AppIcon :name="showParams ? 'caret-down' : 'caret-right'" :size="12" class="tool-chevron" />
      </div>
    </div>

    <div v-if="toolCall.streamingOutput && toolCall.status === 'executing'" class="tool-streaming" data-test="tool-streaming">
      <div class="tool-section-title">实时输出</div>
      <pre class="tool-streaming-content">{{ toolCall.streamingOutput }}</pre>
    </div>

    <div v-if="showParams" class="tool-params">
      <div class="tool-section-title">参数</div>
      <pre class="tool-json">{{ JSON.stringify(toolCall.parameters, null, 2) }}</pre>
    </div>

    <div v-if="toolCall.status !== 'pending'" class="tool-result">
      <div class="tool-section-title" @click="showResult = !showResult">
        结果 <AppIcon :name="showResult ? 'caret-down' : 'caret-right'" :size="12" />
      </div>
      <div v-if="showResult && toolCall.result" class="tool-result-content">
        <div v-if="toolCall.result.success !== undefined" class="result-status">
          <AppIcon
            :name="toolCall.result.success ? 'check-circle' : 'x-circle'"
            :size="14"
            :color="toolCall.result.success ? 'var(--color-success)' : 'var(--color-error)'"
            class="result-status-icon"
          />
          {{ toolCall.result.success ? ' 成功' : ' 失败' }}
        </div>
        <div v-if="toolCall.result.error" class="result-error">{{ toolCall.result.error }}</div>

        <div v-if="toolCall.result.diff" class="diff-section">
          <div class="diff-header">变更对比</div>
          <pre class="diff-content"><code v-html="renderDiff(toolCall.result.diff as string)"></code></pre>
        </div>

        <pre v-if="toolCall.result.stdout" class="tool-json">{{ toolCall.result.stdout }}</pre>
        <pre v-if="toolCall.result.stderr" class="tool-json stderr">{{ toolCall.result.stderr }}</pre>
      </div>
    </div>
  </div>
</template>

<style scoped src="./ToolCallCard.css">
</style>