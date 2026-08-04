<script setup lang="ts">
import { computed, defineAsyncComponent } from 'vue'
import { useUiStore, type RightTabType } from '@/stores/ui'
import AppIcon, { type IconName } from '@/components/common/AppIcon.vue'

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

const uiStore = useUiStore()

const currentTab = computed(() => allTabs.find(t => t.key === uiStore.activeRightTab))

const activeRightTab = computed(() => uiStore.activeRightTab)

function selectTab(tab: TabDef) {
  uiStore.setActiveRightTab(tab.key)
}

function close() {
  emit('close')
}

const emit = defineEmits<{
  close: []
}>()

let touchStartX = 0

function onTouchStart(e: TouchEvent) {
  touchStartX = e.touches[0].clientX
}

function onTouchEnd(e: TouchEvent) {
  const delta = e.changedTouches[0].clientX - touchStartX
  if (delta > 60) {
    close()
  }
}
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

<style scoped>
.panel-drawer-overlay {
  position: fixed;
  inset: 0;
  z-index: 4500;
  background: var(--color-overlay);
  animation: fadeIn var(--transition-fast) ease;
  display: flex;
  justify-content: flex-end;
}

.panel-drawer {
  width: 100%;
  max-width: 420px;
  height: 100vh;
  height: 100dvh;
  background: var(--color-bg-primary);
  display: flex;
  flex-direction: column;
  animation: slideInRight 0.3s ease;
  overflow: hidden;
}

.drawer-header {
  display: flex;
  align-items: center;
  padding: var(--space-md) var(--space-lg);
  padding-top: calc(var(--space-md) + env(safe-area-inset-top, 0px));
  border-bottom: 1px solid var(--color-border);
  background: var(--color-bg-secondary);
  flex-shrink: 0;
}

.drawer-back-btn {
  display: flex;
  align-items: center;
  gap: var(--space-xs);
  padding: var(--space-xs) var(--space-sm);
  border: none;
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--color-accent);
  font-size: var(--font-size-base);
  cursor: pointer;
  min-height: 44px;
  min-width: 44px;
  -webkit-tap-highlight-color: transparent;
}

.drawer-title {
  flex: 1;
  text-align: center;
  font-size: var(--font-size-lg);
  font-weight: 600;
  color: var(--color-text-primary);
}

.drawer-spacer {
  width: 60px;
}

.drawer-tabs {
  display: flex;
  gap: var(--space-xs);
  padding: var(--space-sm) var(--space-md);
  background: var(--color-bg-secondary);
  border-bottom: 1px solid var(--color-border);
  overflow-x: auto;
  scroll-behavior: smooth;
  -webkit-overflow-scrolling: touch;
  scrollbar-width: none;
  flex-shrink: 0;
}

.drawer-tabs::-webkit-scrollbar {
  display: none;
}

.drawer-tab-btn {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 6px 12px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--color-text-secondary);
  font-size: var(--font-size-sm);
  cursor: pointer;
  white-space: nowrap;
  flex-shrink: 0;
  transition: all var(--transition-fast);
  min-height: 36px;
  -webkit-tap-highlight-color: transparent;
}

.drawer-tab-btn.active {
  background: var(--color-accent);
  border-color: var(--color-accent);
  color: white;
}

.drawer-body {
  flex: 1;
  overflow-y: auto;
  min-height: 0;
  -webkit-overflow-scrolling: touch;
}

@keyframes slideInRight {
  from {
    transform: translateX(100%);
  }
  to {
    transform: translateX(0);
  }
}

@keyframes fadeIn {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}
</style>
