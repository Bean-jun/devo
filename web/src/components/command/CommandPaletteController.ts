import { watch, nextTick, ref } from 'vue'
import { useCommandStore } from '@/stores/command'
import { useCommand } from '@/composables/useCommand'

export function useCommandPalette() {
  const commandStore = useCommandStore()
  const { openPalette } = useCommand()
  const inputRef = ref<HTMLInputElement | null>(null)
  const listRef = ref<HTMLDivElement | null>(null)

watch(() => commandStore.isOpen, (open) => {
  if (open) {
    nextTick(() => inputRef.value?.focus())
  }
})

watch(() => commandStore.selectedIndex, () => {
  nextTick(() => {
    const selected = listRef.value?.querySelector('.palette-item.selected')
    selected?.scrollIntoView({ block: 'nearest' })
  })
})

function handleInput(e: Event) {
  const target = e.target as HTMLInputElement
  commandStore.setQuery(target.value)
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'ArrowDown') {
    e.preventDefault()
    commandStore.moveDown()
  } else if (e.key === 'ArrowUp') {
    e.preventDefault()
    commandStore.moveUp()
  } else if (e.key === 'Enter') {
    e.preventDefault()
    commandStore.select()
  } else if (e.key === 'Escape') {
    e.preventDefault()
    e.stopPropagation()
    commandStore.close()
  }
}

function handleSelect(index: number) {
  commandStore.select(index)
}

  return {
    commandStore,
    openPalette,
    inputRef,
    listRef,
    handleInput,
    handleKeydown,
    handleSelect,
  }
}