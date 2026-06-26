<script setup lang="ts">
import { ref } from 'vue'
import { useUiStore } from '@/stores/ui'

const uiStore = useUiStore()
type SubTab = 'project' | 'global'
const activeTab = ref<SubTab>('global')
</script>

<template>
  <div class="settings-panel">
    <div class="settings-tabs">
      <button
        class="subtab-btn"
        :class="{ active: activeTab === 'project' }"
        @click="activeTab = 'project'"
      >项目设置</button>
      <button
        class="subtab-btn"
        :class="{ active: activeTab === 'global' }"
        @click="activeTab = 'global'"
      >全局设置</button>
    </div>

    <div v-if="activeTab === 'project'" class="settings-content">
      <div class="setting-item">
        <label>工作目录</label>
        <span class="setting-value">{{ uiStore.activeWorkspace || '无' }}</span>
      </div>
      <div class="setting-item">
        <label>审批策略</label>
        <span class="setting-hint">待实现</span>
      </div>
      <div class="setting-item">
        <label>MCP 工具管理</label>
        <span class="setting-hint">待实现</span>
      </div>
    </div>

    <div v-else class="settings-content">
      <div class="setting-item">
        <label>主题</label>
        <span class="setting-value">{{ uiStore.theme }}</span>
      </div>
      <div class="setting-item">
        <label>快捷键</label>
        <span class="setting-hint">Ctrl+K 命令面板</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.settings-panel { display: flex; flex-direction: column; height: 100%; }
.settings-tabs { display: flex; gap: 4px; padding: 8px 12px; border-bottom: 1px solid var(--color-border); }
.subtab-btn { padding: 5px 14px; border: 1px solid var(--color-border); border-radius: 4px; background: transparent; color: var(--color-text-secondary); cursor: pointer; font-size: 12px; }
.subtab-btn.active { background: var(--color-accent); border-color: var(--color-accent); color: white; }
.settings-content { flex: 1; overflow-y: auto; padding: 12px; }
.setting-item { display: flex; justify-content: space-between; align-items: center; padding: 10px 0; border-bottom: 1px solid var(--color-border); }
.setting-item label { font-size: 13px; color: var(--color-text-primary); }
.setting-value { font-size: 12px; color: var(--color-text-secondary); font-family: var(--font-mono); }
.setting-hint { font-size: 12px; color: var(--color-text-tertiary); }
</style>