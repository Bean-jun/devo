<script setup lang="ts">
import { useUiStore } from '@/stores/ui'

const uiStore = useUiStore()
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

<style scoped>
.terminal-panel { display: flex; flex-direction: column; height: 100%; background: #1e1e1e; }
.terminal-header { display: flex; align-items: center; justify-content: space-between; padding: 6px 12px; background: #2d2d2d; border-bottom: 1px solid #3c3c3c; color: #ccc; font-size: 12px; }
.terminal-cwd { color: #888; font-size: 11px; font-family: var(--font-mono); }
.terminal-content { flex: 1; display: flex; align-items: center; justify-content: center; }
.terminal-placeholder { color: #666; font-size: 13px; }
</style>