import { computed, ref, nextTick, onMounted, onUnmounted } from 'vue'
import { useSessionStore } from '@/stores/session'
import { useUiStore } from '@/stores/ui'
import { useUpdateCheck } from '@/composables/useUpdateCheck'
import { STATUS_LABELS, STATUS_COLORS } from '@/utils/constants'

export type Density = 'compact' | 'tablet' | 'full'

export function useStatusBar(getDensity?: () => string) {
  const sessionStore = useSessionStore()
  const uiStore = useUiStore()
  const { updateInfo } = useUpdateCheck()

  const isRenaming = ref(false)
  const renameValue = ref('')
  const renameInputRef = ref<HTMLInputElement>()
  const yoloLoading = ref(false)
  const teamModeLoading = ref(false)

  const SPINNER_CHARS = ['⣾', '⣽', '⣻', '⢿', '⡿', '⣟', '⣯', '⣷']
  const spinnerFrame = ref(0)
  let spinnerTimer: ReturnType<typeof setInterval> | null = null

  onMounted(() => {
    spinnerTimer = setInterval(() => {
      if (uiStore.activityActive) {
        spinnerFrame.value = (spinnerFrame.value + 1) % SPINNER_CHARS.length
      }
    }, 250)
  })

  onUnmounted(() => {
    if (spinnerTimer) {
      clearInterval(spinnerTimer)
      spinnerTimer = null
    }
  })

const spinnerChar = computed(() => SPINNER_CHARS[spinnerFrame.value])

const activityText = computed(() => {
  if (!uiStore.activityActive || !uiStore.activityStream) return ''
  return uiStore.activityStream.replace(/\n/g, ' ').replace(/\r/g, '')
})

const sessionName = computed(() => sessionStore.currentSession?.title ?? '未连接')
const statusLabel = computed(() => STATUS_LABELS[sessionStore.sessionStatus] ?? '空闲')
const statusColor = computed(() => STATUS_COLORS[sessionStore.sessionStatus] ?? '#34c759')
const isProcessing = computed(() => sessionStore.isProcessing)

const themeIconName = computed(() => uiStore.theme === 'dark' ? 'sun' : 'moon')
const themeLabel = computed(() => uiStore.theme === 'dark' ? '浅色' : '深色')

async function startRename() {
  if (!sessionStore.currentSession) return
  renameValue.value = sessionName.value
  isRenaming.value = true
  await nextTick()
  renameInputRef.value?.focus()
  renameInputRef.value?.select()
}

async function confirmRename() {
  if (!isRenaming.value) return
  const newName = renameValue.value.trim()
  isRenaming.value = false
  if (newName && newName !== sessionName.value && sessionStore.currentSession) {
    try {
      await sessionStore.renameSession(sessionStore.currentSession.id, newName)
      uiStore.showToast('success', '会话已重命名')
    } catch {
      uiStore.showToast('error', '重命名失败')
    }
  }
}

function cancelRename(e?: KeyboardEvent) {
  e?.stopPropagation()
  isRenaming.value = false
}

function toggleTheme(event: MouseEvent) {
  const rect = (event.currentTarget as HTMLElement).getBoundingClientRect()
  const x = rect.left + rect.width / 2
  const y = rect.top + rect.height / 2
  uiStore.toggleThemeWithTransition(x, y)
}

async function toggleYolo() {
  yoloLoading.value = true
  try {
    await sessionStore.toggleYolo()
    const label = sessionStore.yoloEnabled ? 'YOLO 模式已开启' : 'YOLO 模式已关闭'
    uiStore.showToast('success', label)
  } catch {
    uiStore.showToast('error', 'YOLO 切换失败')
  } finally {
    yoloLoading.value = false
  }
}

async function toggleTeamMode() {
  teamModeLoading.value = true
  try {
    await sessionStore.setTeamMode(!sessionStore.teamModeEnabled)
    const label = sessionStore.teamModeEnabled ? 'Team Mode 已开启' : 'Team Mode 已关闭'
    uiStore.showToast('success', label)
  } catch {
    uiStore.showToast('error', 'Team Mode 切换失败')
  } finally {
    teamModeLoading.value = false
  }
}

const connectionStatusText = computed(() => {
  switch (uiStore.connectionStatus) {
    case 'connected': return '已连接'
    case 'connecting': return '连接中'
    case 'disconnected': return '未连接'
  }
})

const connectionColor = computed(() => {
  switch (uiStore.connectionStatus) {
    case 'connected': return '#34c759'
    case 'connecting': return '#ff9500'
    case 'disconnected': return '#ff3b30'
  }
})

const serverPort = computed(() => window.location.port)

const density = computed<Density>(() => {
  const d = getDensity?.() ?? 'full'
  if (d === 'compact' || d === 'tablet' || d === 'full') return d
  return 'full'
})

const showQuick = ref(false)
const statusbarRef = ref<HTMLElement>()

const quickPanelStyle = computed(() => {
  if (!statusbarRef.value) return {}
  const rect = statusbarRef.value.getBoundingClientRect()
  return {
    top: rect.bottom + 'px',
    left: '0px',
    right: '0px',
  }
})

function toggleQuick() {
  showQuick.value = !showQuick.value
}

function closeQuick() {
  showQuick.value = false
}

const hasUpdate = computed(() => updateInfo.value?.has_update ?? false)
const latestVersion = computed(() => updateInfo.value?.latest_version ?? '')

function showUpdateModal() {
  uiStore.setActiveModal('update')
}

  return {
    sessionStore,
    uiStore,
    isRenaming,
    renameValue,
    renameInputRef,
    yoloLoading,
    teamModeLoading,
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
    toggleTeamMode,
    connectionStatusText,
    connectionColor,
    serverPort,
    density,
    showQuick,
    statusbarRef,
    quickPanelStyle,
    toggleQuick,
    closeQuick,
    hasUpdate,
    latestVersion,
    showUpdateModal,
  }
}