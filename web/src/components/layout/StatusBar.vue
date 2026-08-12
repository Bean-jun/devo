<script setup lang="ts">
import AppIcon from '@/components/common/AppIcon.vue'
import ReasoningEffortToggle from '@/components/chat/ReasoningEffortToggle.vue'
import { useKeyboard } from '@/composables/useKeyboard'
import { useStatusBar } from './StatusBarController'

const props = withDefaults(defineProps<{
  density?: 'compact' | 'tablet' | 'full'
}>(), {
  density: 'full',
})

const {
  sessionStore,
  uiStore,
  isRenaming,
  renameValue,
  renameInputRef,
  yoloLoading,
  spinnerChar,
  activityText,
  sessionName,
  statusLabel,
  statusColor,
  isProcessing,
  themeIconName,
  themeLabel,
  startRename,
  confirmRename,
  cancelRename,
  toggleTheme,
  toggleYolo,
  connectionStatusText,
  connectionColor,
  serverPort,
  density,
  showQuick,
  statusbarRef,
  quickPanelStyle,
  toggleQuick,
  closeQuick,
  hasUpdate,
  latestVersion,
  showUpdateModal,
} = useStatusBar(() => props.density)

defineExpose({ startRename })

useKeyboard([
  {
    key: 'F2',
    handler: () => startRename(),
  },
])
</script>

<template>
  <header
    ref="statusbarRef"
    v-if="sessionStore.currentSession"
    class="statusbar"
    :class="[density, { yolo: sessionStore.yoloEnabled }]"
    data-test="status-bar"
  >
    <div class="statusbar-left">
      <input
        v-if="isRenaming"
        ref="renameInputRef"
        v-model="renameValue"
        class="session-name-input"
        @keydown.enter="confirmRename"
        @keydown.escape.stop="cancelRename"
        @blur="confirmRename"
      />
      <span v-else class="session-name" :title="sessionName" @dblclick="startRename">{{ sessionName }}</span>
      <span
        class="status-indicator"
        :class="{ processing: isProcessing }"
        :style="{ color: density === 'compact' ? 'transparent' : statusColor }"
      >
        <span class="status-dot" :style="{ background: statusColor }"></span>
        <span v-if="density !== 'compact'" class="status-label">{{ statusLabel }}</span>
      </span>
      <button
        class="yolo-toggle"
        :class="{
          active: sessionStore.yoloEnabled,
          mini: density === 'compact',
        }"
        :title="sessionStore.yoloEnabled ? 'YOLO 模式已开启 - 点击关闭' : 'YOLO 模式 - 点击开启自动批准'"
        :disabled="yoloLoading"
        @click="toggleYolo"
      >
        <AppIcon :name="yoloLoading ? 'hourglass' : 'fire'" :size="16" :color="sessionStore.yoloEnabled ? '#ff9500' : undefined" class="yolo-icon" />
        <span v-if="density !== 'compact'" class="yolo-label" :class="{ on: sessionStore.yoloEnabled }">YOLO</span>
      </button>
    </div>
    <div v-if="density === 'full' && uiStore.activityActive" class="statusbar-center">
      <span class="activity-spinner">{{ spinnerChar }}</span>
      <span class="activity-text">{{ activityText }}</span>
    </div>
    <div class="statusbar-right">
      <ReasoningEffortToggle v-if="density !== 'compact'" />
      <button class="theme-toggle" :class="{ 'icon-only': density === 'compact' }" :title="themeLabel" @click="toggleTheme">
        <AppIcon :name="themeIconName" :size="16" />
      </button>
      <button
        v-if="hasUpdate"
        class="update-indicator"
        :class="{ 'icon-only': density === 'compact' }"
        :title="'新版本可用: ' + latestVersion"
        @click="showUpdateModal"
        data-test="update-indicator"
      >
        <AppIcon name="arrow-circle-up" :size="16" color="#34c759" />
        <span v-if="density !== 'compact'" class="update-label">更新</span>
      </button>
      <span class="connection-status" :title="connectionStatusText">
        <AppIcon name="circle" :size="10" weight="fill" :color="connectionColor" />
        <span v-if="density !== 'compact'" class="connection-text">{{ connectionStatusText }}</span>
      </span>
      <span v-if="density === 'full'" class="port-info" title="后端端口">:{{ serverPort }}</span>
      <button
        v-if="density !== 'full'"
        class="quick-toggle"
        :class="{ active: showQuick }"
        title="更多信息"
        @click="toggleQuick"
        data-test="quick-toggle"
      >
        <span class="quick-toggle-icon">☰</span>
      </button>
    </div>
  </header>

  <Teleport v-if="density !== 'full'" to="body">
    <div v-if="showQuick" class="quick-backdrop" @click="closeQuick" data-test="quick-backdrop" />
    <div v-if="showQuick" class="quick-panel" :style="quickPanelStyle" data-test="quick-panel">
      <div v-if="density === 'compact'" class="quick-row">
        <span class="quick-label">推理</span>
        <ReasoningEffortToggle />
      </div>
      <div v-if="activityText" class="quick-row">
        <span class="activity-spinner-inline">{{ spinnerChar }}</span>
        <span class="activity-text-inline">{{ activityText }}</span>
      </div>
      <div class="quick-row quick-info">
        <span class="quick-item">
          <AppIcon name="circle" :size="8" weight="fill" :color="connectionColor" />
          {{ connectionStatusText }}
        </span>
        <span class="quick-sep">·</span>
        <span class="quick-item">:{{ serverPort }}</span>
      </div>
    </div>
  </Teleport>
</template>

<style scoped src="./StatusBar.css">
</style>