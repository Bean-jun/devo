import { computed, defineAsyncComponent } from 'vue'
import { useUiStore, type RightTabType } from '@/stores/ui'
import type { IconName } from '@/components/common/AppIconController'

export function useMobilePanelDrawer(emit: (e: string, ...args: any[]) => void) {
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

  return {
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
  }
}