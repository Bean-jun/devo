<script setup lang="ts">
import AppSidebar from '@/components/layout/AppSidebar.vue'
import StatusBar from '@/components/layout/StatusBar.vue'
import RightPanel from '@/components/layout/RightPanel.vue'
import GlobalModals from '@/components/layout/GlobalModals.vue'
import AppIcon from '@/components/common/AppIcon.vue'
import { useBrowserLayout } from './BrowserLayoutController'

const {
  uiStore,
  sidebarWidth,
  sidebarCollapsed,
  rightPanelVisible,
  collapsedWidth,
  rightPanelWidth,
  leftWrapperRef,
  rightWrapperRef,
  startResize,
} = useBrowserLayout()
</script>

<template>
  <div class="browser-layout">
    <div ref="leftWrapperRef" class="sidebar-wrapper" :class="{ collapsed: sidebarCollapsed }" :style="{ width: (sidebarCollapsed ? collapsedWidth : sidebarWidth) + 'px' }">
      <AppSidebar :collapsed="sidebarCollapsed" />
    </div>
    <div class="resize-handle" @mousedown="startResize('left')">
      <div class="handle-hit"></div>
      <span class="handle-icon">║</span>
    </div>
    <div class="browser-main">
      <StatusBar />
      <div class="browser-content">
        <router-view />
      </div>
    </div>
    <div v-if="rightPanelVisible" class="resize-handle" @mousedown="startResize('right')">
      <div class="handle-hit"></div>
      <span class="handle-icon">║</span>
    </div>
    <div 
      ref="rightWrapperRef" 
      class="right-wrapper"
      :class="{ collapsed: !rightPanelVisible }"
      :style="{ width: (rightPanelVisible ? rightPanelWidth : 0) + 'px' }"
    >
      <RightPanel />
    </div>
    <div
      v-if="!rightPanelVisible"
      class="right-panel-expand-btn"
      @click="uiStore.toggleRightPanel()"
      title="展开面板"
    >
      <AppIcon name="caret-left" :size="14" class="expand-icon" />
    </div>
    <GlobalModals />
  </div>
</template>

<style scoped src="./BrowserLayout.css">
</style>