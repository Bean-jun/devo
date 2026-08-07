<script setup lang="ts">
import AppIcon from '@/components/common/AppIcon.vue'
import { useMobilePanelDrawer } from './MobilePanelDrawerController'

const emit = defineEmits<{
  close: []
}>()

const {
  FilesPanel,
  SkillsPanel,
  McpPanel,
  MemoryPanel,
  BackgroundPanel,
  DashboardPanel,
  SettingsPanel,
  allTabs,
  uiStore,
  currentTab,
  activeRightTab,
  selectTab,
  close,
  onTouchStart,
  onTouchEnd,
} = useMobilePanelDrawer(emit as any)
</script>

<template>
  <Teleport to="body">
    <div class="panel-drawer-overlay" @click.self="close" data-test="panel-drawer-overlay">
      <div
        class="panel-drawer"
        data-test="panel-drawer"
        @touchstart.passive="onTouchStart"
        @touchend.passive="onTouchEnd"
      >
        <div class="drawer-header">
          <button class="drawer-back-btn" data-test="drawer-back-btn" @click="close">
            <AppIcon name="caret-left" :size="16" />
            <span>返回</span>
          </button>
          <span class="drawer-title">{{ currentTab?.label || '面板' }}</span>
          <div class="drawer-spacer"></div>
        </div>

        <div class="drawer-tabs">
          <button
            v-for="tab in allTabs"
            :key="tab.key"
            class="drawer-tab-btn"
            :class="{ active: activeRightTab === tab.key }"
            data-test="drawer-tab"
            @click="selectTab(tab)"
          >
            <AppIcon :name="tab.icon" :size="14" />
            <span>{{ tab.label }}</span>
          </button>
        </div>

        <div class="drawer-body">
          <FilesPanel v-if="activeRightTab === 'files'" />
          <SkillsPanel v-else-if="activeRightTab === 'skills'" />
          <McpPanel v-else-if="activeRightTab === 'mcp'" />
          <MemoryPanel v-else-if="activeRightTab === 'memory'" />
          <BackgroundPanel v-else-if="activeRightTab === 'background'" />
          <DashboardPanel v-else-if="activeRightTab === 'dashboard'" />
          <SettingsPanel v-else-if="activeRightTab === 'settings'" />
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped src="./MobilePanelDrawer.css">
</style>