<script setup lang="ts">
import { defineAsyncComponent } from 'vue'
import { useUiStore, type RightTabType } from '@/stores/ui'
import AppIcon, { type IconName } from '@/components/common/AppIcon.vue'

const uiStore = useUiStore()

const FilesPanel = defineAsyncComponent(() => import('@/panels/files/FilesPanel.vue'))
const SkillsPanel = defineAsyncComponent(() => import('@/panels/skills/SkillsPanel.vue'))
const McpPanel = defineAsyncComponent(() => import('@/panels/mcp/McpPanel.vue'))
const MemoryPanel = defineAsyncComponent(() => import('@/panels/memory/MemoryPanel.vue'))
const BackgroundPanel = defineAsyncComponent(() => import('@/panels/background/BackgroundPanel.vue'))
const DashboardPanel = defineAsyncComponent(() => import('@/panels/dashboard/DashboardPanel.vue'))
const SettingsPanel = defineAsyncComponent(() => import('@/panels/settings/SettingsPanel.vue'))

interface TabDef {
  key: RightTabType
  label: string
  icon: IconName
}

const allTabs: TabDef[] = [
  { key: 'files', label: 'Files', icon: 'folder' },
  { key: 'skills', label: 'Skills', icon: 'lightning' },
  { key: 'mcp', label: 'MCP', icon: 'plug' },
  { key: 'memory', label: 'Memory', icon: 'brain' },
  { key: 'background', label: 'BG-Tasks', icon: 'arrow-clockwise' },
  { key: 'dashboard', label: 'Dashboard', icon: 'chart-bar' },
  { key: 'settings', label: 'Settings', icon: 'gear' },
]

function selectTab(tab: TabDef) {
  uiStore.setActiveRightTab(tab.key)
}
</script>

<template>
  <aside class="right-panel">
    <div class="right-panel-header">
      <button
        v-for="tab in allTabs"
        :key="tab.key"
        class="panel-tab-btn"
        :class="{ active: uiStore.activeRightTab === tab.key }"
        :title="tab.label"
        @click="selectTab(tab)"
      >
        <AppIcon :name="tab.icon" :size="16" class="tab-icon" />
        <span class="tab-label">{{ tab.label }}</span>
      </button>
      <button class="panel-collapse-btn" @click="uiStore.toggleRightPanel()" title="折叠面板">
        <AppIcon name="caret-right" :size="14" />
      </button>
    </div>
    <div class="right-panel-body">
      <FilesPanel v-if="uiStore.activeRightTab === 'files'" />
      <SkillsPanel v-else-if="uiStore.activeRightTab === 'skills'" />
      <McpPanel v-else-if="uiStore.activeRightTab === 'mcp'" />
      <MemoryPanel v-else-if="uiStore.activeRightTab === 'memory'" />
      <BackgroundPanel v-else-if="uiStore.activeRightTab === 'background'" />
      <DashboardPanel v-else-if="uiStore.activeRightTab === 'dashboard'" />
      <SettingsPanel v-else-if="uiStore.activeRightTab === 'settings'" />
    </div>
  </aside>
</template>

<style scoped>
.right-panel {
  display: flex;
  flex-direction: column;
  height: 100vh;
  background: var(--color-bg-primary);
  border-left: 1px solid var(--color-border);
  overflow: hidden;
}

.right-panel-header {
  display: flex;
  flex-direction: row;
  flex-wrap: wrap;
  gap: 4px;
  padding: 8px;
  border-bottom: 1px solid var(--color-border);
  background: var(--color-bg-secondary);
}

.panel-tab-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 10px;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  background: transparent;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all var(--transition-fast);
  font-size: 12px;
  white-space: nowrap;
}

.panel-tab-btn:hover {
  background: var(--color-bg-hover);
  color: var(--color-text-primary);
}

.panel-tab-btn.active {
  background: var(--color-accent);
  border-color: var(--color-accent);
  color: white;
}

.tab-icon {
  font-size: 14px;
  line-height: 1;
}

.tab-label {
  font-size: 12px;
}

.panel-collapse-btn {
  margin-left: auto;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 6px 8px;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  background: transparent;
  color: var(--color-text-tertiary);
  cursor: pointer;
  font-size: 10px;
  transition: all var(--transition-fast);
  flex-shrink: 0;
}

.panel-collapse-btn:hover {
  background: var(--color-bg-hover);
  color: var(--color-accent);
  border-color: var(--color-accent);
}

.right-panel-body {
  flex: 1;
  overflow-y: auto;
  min-height: 0;
}
</style>