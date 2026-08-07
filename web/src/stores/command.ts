import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { IconName } from '@/components/common/AppIconController'

export interface Command {
  id: string
  name: string
  description: string
  placeholder?: string
  group?: 'panel' | 'session' | 'workspace' | 'app'
  mobileLabel?: string
  icon?: IconName
}

export const useCommandStore = defineStore('command', () => {
  const isOpen = ref(false)
  const query = ref('')
  const selectedIndex = ref(0)
  const commands = ref<Command[]>([])
  const onSelectCallback = ref<((cmd: Command) => void) | null>(null)

  const filteredCommands = computed(() => {
    if (!query.value) return commands.value
    const lowerQuery = query.value.toLowerCase()
    return commands.value.filter(
      cmd =>
        cmd.name.toLowerCase().includes(lowerQuery) ||
        cmd.description.toLowerCase().includes(lowerQuery)
    )
  })

  const selectedCommand = computed(() => {
    if (filteredCommands.value.length === 0) return null
    const idx = Math.min(selectedIndex.value, filteredCommands.value.length - 1)
    return filteredCommands.value[idx]
  })

  function open(cmds: Command[], onSelect?: (cmd: Command) => void): void {
    commands.value = cmds
    onSelectCallback.value = onSelect ?? null
    isOpen.value = true
    query.value = ''
    selectedIndex.value = 0
  }

  function close(): void {
    isOpen.value = false
    query.value = ''
    selectedIndex.value = 0
  }

  function setQuery(value: string): void {
    query.value = value
    selectedIndex.value = 0
  }

  function moveUp(): void {
    if (selectedIndex.value > 0) {
      selectedIndex.value--
    }
  }

  function moveDown(): void {
    if (selectedIndex.value < filteredCommands.value.length - 1) {
      selectedIndex.value++
    }
  }

  function select(index?: number): void {
    const cmd = index !== undefined ? filteredCommands.value[index] : selectedCommand.value
    if (cmd) {
      onSelectCallback.value?.(cmd)
      close()
    }
  }

  return {
    isOpen,
    query,
    selectedIndex,
    commands,
    filteredCommands,
    selectedCommand,
    open,
    close,
    setQuery,
    moveUp,
    moveDown,
    select,
  }
})