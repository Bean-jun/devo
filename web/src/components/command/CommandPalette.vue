<script setup lang="ts">
import { watch, nextTick, ref } from 'vue'
import { useCommandStore } from '@/stores/command'
import { useCommand } from '@/composables/useCommand'

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
</script>

<template>
  <div v-if="commandStore.isOpen" class="command-overlay" @click="commandStore.close()">
    <div class="command-palette" data-test="command-palette" @click.stop>
      <div class="palette-input-wrapper">
        <span class="palette-prefix">/</span>
        <input
          ref="inputRef"
          type="text"
          class="palette-input"
          placeholder="输入命令..."
          :value="commandStore.query"
          @input="handleInput"
          @keydown="handleKeydown"
        />
      </div>

      <div v-if="commandStore.filteredCommands.length > 0" ref="listRef" class="palette-list">
        <div
          v-for="(cmd, index) in commandStore.filteredCommands"
          :key="cmd.id"
          class="palette-item"
          :class="{ selected: index === commandStore.selectedIndex }"
          @click="handleSelect(index)"
          @mouseenter="commandStore.selectedIndex = index"
          data-test="command-item"
        >
          <div class="palette-item-left">
            <span class="palette-item-name">{{ cmd.name }}</span>
            <span v-if="cmd.placeholder" class="palette-item-placeholder">
              {{ cmd.placeholder }}
            </span>
          </div>
          <span class="palette-item-desc">{{ cmd.description }}</span>
        </div>
      </div>

      <div v-else class="palette-empty">
        无匹配命令
      </div>

      <div class="palette-footer">
        <span>↑↓ 导航</span>
        <span>Enter 选择</span>
        <span>Esc 关闭</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.command-overlay {
  position: fixed;
  inset: 0;
  z-index: 5000;
  display: flex;
  justify-content: center;
  padding-top: 15vh;
  background: var(--color-overlay);
  animation: fadeIn var(--transition-fast) ease;
}

.command-palette {
  width: 520px;
  max-height: 400px;
  background: var(--color-bg-primary);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-modal);
  overflow: hidden;
  animation: modalIn var(--transition-base) ease;
  display: flex;
  flex-direction: column;
}

.palette-input-wrapper {
  display: flex;
  align-items: center;
  padding: var(--space-md) var(--space-lg);
  border-bottom: 1px solid var(--color-border-light);
}

.palette-prefix {
  font-size: var(--font-size-lg);
  font-weight: 600;
  color: var(--color-accent);
  margin-right: var(--space-sm);
}

.palette-input {
  flex: 1;
  font-size: var(--font-size-base);
  color: var(--color-text-primary);
  background: transparent;
  border: none;
  outline: none;
}

.palette-input::placeholder {
  color: var(--color-text-tertiary);
}

.palette-list {
  flex: 1;
  overflow-y: auto;
  padding: var(--space-xs) 0;
}

.palette-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-sm) var(--space-lg);
  cursor: pointer;
  transition: background var(--transition-fast);
}

.palette-item:hover,
.palette-item.selected {
  background: var(--color-accent-light);
}

.palette-item-left {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
}

.palette-item-name {
  font-family: var(--font-mono);
  font-size: var(--font-size-sm);
  font-weight: 600;
  color: var(--color-accent);
}

.palette-item-placeholder {
  font-size: var(--font-size-sm);
  color: var(--color-text-tertiary);
}

.palette-item-desc {
  font-size: var(--font-size-xs);
  color: var(--color-text-secondary);
}

.palette-empty {
  padding: var(--space-xl);
  text-align: center;
  font-size: var(--font-size-sm);
  color: var(--color-text-tertiary);
}

.palette-footer {
  display: flex;
  justify-content: center;
  gap: var(--space-lg);
  padding: var(--space-sm) var(--space-lg);
  border-top: 1px solid var(--color-border-light);
  font-size: var(--font-size-xs);
  color: var(--color-text-tertiary);
}
</style>