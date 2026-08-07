import { ref, computed, watch } from 'vue'
import { useCommand } from '@/composables/useCommand'
import type { Command } from '@/stores/command'

export function useMobileCommandSheet(emit: (e: string, ...args: any[]) => void) {
  const { allMobileCommands } = useCommand()
  const searchQuery = ref('')

const groupedCommands = computed(() => {
  const q = searchQuery.value.toLowerCase().trim()
  const filtered = q
    ? allMobileCommands.filter(
        cmd =>
          cmd.name.toLowerCase().includes(q) ||
          cmd.description.toLowerCase().includes(q) ||
          (cmd.mobileLabel || '').toLowerCase().includes(q)
      )
    : allMobileCommands

  const groups: Record<string, { label: string; commands: Command[] }> = {
    session: { label: 'Session', commands: [] },
    panel: { label: 'Panel', commands: [] },
    workspace: { label: 'Workspace', commands: [] },
    app: { label: 'App', commands: [] },
  }

  for (const cmd of filtered) {
    const g = cmd.group || 'session'
    if (groups[g]) {
      groups[g].commands.push(cmd)
    }
  }

  return Object.values(groups).filter(g => g.commands.length > 0)
})

function selectCommand(cmd: Command) {
  emit('select', cmd)
}

function onBackdropClick() {
  emit('close')
}

function onSheetClick(e: Event) {
  e.stopPropagation()
}

let startY = 0
let currentY = 0

function onTouchStart(e: TouchEvent) {
  startY = e.touches[0].clientY
}

function onTouchMove(e: TouchEvent) {
  currentY = e.touches[0].clientY
  const delta = currentY - startY
  if (delta > 80) {
    emit('close')
  }
}

  return {
    searchQuery,
    groupedCommands,
    selectCommand,
    onBackdropClick,
    onSheetClick,
    onTouchStart,
    onTouchMove,
  }
}