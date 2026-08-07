<script setup lang="ts">
import { useTerminalPanel } from './TerminalPanelController'

const { uiStore } = useTerminalPanel()
</script>

<template>
  <div class="terminal-panel">
    <div class="terminal-header">
      <span>Terminal</span>
      <span class="terminal-cwd" v-if="uiStore.activeWorkspace">{{ uiStore.activeWorkspace }}</span>
    </div>
    <div class="terminal-content">
      <div class="terminal-placeholder">终端功能待实现</div>
    </div>
  </div>
</template>

<style scoped src="./TerminalPanel.css">
</style>