<script setup lang="ts">
import AppIcon from '@/components/common/AppIcon.vue'
import { useRightPanel } from './RightPanelController'

const {
  uiStore,
  allTabs,
  selectTab,
  FilesPanel,
  SkillsPanel,
  McpPanel,
  MemoryPanel,
  BackgroundPanel,
  DashboardPanel,
  SettingsPanel,
} = useRightPanel()
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

<style scoped src="./RightPanel.css">
</style>