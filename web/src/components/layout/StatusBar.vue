<script setup lang="ts">
import AppIcon from '@/components/common/AppIcon.vue'
import ReasoningEffortToggle from '@/components/chat/ReasoningEffortToggle.vue'
import { useKeyboard } from '@/composables/useKeyboard'
import { useStatusBar } from './StatusBarController'

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
} = useStatusBar()

defineExpose({ startRename })

useKeyboard([
  {
    key: 'F2',
    handler: () => startRename(),
  },
])
</script>

<template>
  <header v-if="sessionStore.currentSession" class="statusbar" :class="{ yolo: sessionStore.yoloEnabled }" data-test="status-bar">
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
        :style="{ color: statusColor }"
      >
        <span class="status-dot" :style="{ background: statusColor }"></span>
        {{ statusLabel }}
      </span>
      <button
        class="yolo-toggle"
        :class="{ active: sessionStore.yoloEnabled }"
        :title="sessionStore.yoloEnabled ? 'YOLO 模式已开启 - 点击关闭' : 'YOLO 模式 - 点击开启自动批准'"
        :disabled="yoloLoading"
        @click="toggleYolo"
      >
        <AppIcon :name="yoloLoading ? 'hourglass' : 'fire'" :size="16" :color="sessionStore.yoloEnabled ? '#ff9500' : undefined" class="yolo-icon" />
        <span class="yolo-label" :class="{ on: sessionStore.yoloEnabled }">YOLO</span>
      </button>
    </div>
    <div v-if="uiStore.activityActive" class="statusbar-center">
      <span class="activity-spinner">{{ spinnerChar }}</span>
      <span class="activity-text">{{ activityText }}</span>
    </div>
    <div class="statusbar-right">
      <ReasoningEffortToggle />
      <button class="theme-toggle" :title="themeLabel" @click="toggleTheme">
        <AppIcon :name="themeIconName" :size="16" />
      </button>
      <span class="connection-status" :title="connectionStatusText">
        <AppIcon name="circle" :size="10" weight="fill" :color="connectionColor" />
        {{ connectionStatusText }}
      </span>
      <span class="port-info" title="后端端口">:{{ serverPort }}</span>
    </div>
  </header>
</template>

<style scoped src="./StatusBar.css">
</style>