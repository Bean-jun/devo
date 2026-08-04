<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useCommand } from '@/composables/useCommand'
import type { Command } from '@/stores/command'
import AppIcon from '@/components/common/AppIcon.vue'

const emit = defineEmits<{
  select: [cmd: Command]
  close: []
}>()

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
</script>

<template>
  <Teleport to="body">
    <div class="command-sheet-overlay" @click.self="onBackdropClick" data-test="command-sheet-overlay">
      <div
        class="command-sheet"
        @click="onSheetClick"
        data-test="command-sheet"
        @touchstart.passive="onTouchStart"
        @touchmove.passive="onTouchMove"
      >
        <div class="sheet-drag-handle">
          <div class="drag-bar"></div>
        </div>

        <div class="sheet-search">
          <AppIcon name="magnifying-glass" :size="16" class="search-icon" />
          <input
            v-model="searchQuery"
            class="search-input"
            type="text"
            placeholder="搜索命令..."
            autofocus
            data-test="command-search-input"
          />
        </div>

        <div class="sheet-body">
          <div
            v-for="group in groupedCommands"
            :key="group.label"
            class="command-group"
          >
            <div class="group-label">{{ group.label }}</div>
            <button
              v-for="cmd in group.commands"
              :key="cmd.id"
              class="command-item"
              data-test="command-item"
              @click="selectCommand(cmd)"
            >
              <AppIcon v-if="cmd.icon" :name="cmd.icon" :size="18" class="command-icon" />
              <span class="command-name">{{ cmd.name }}</span>
              <span v-if="cmd.placeholder" class="command-placeholder">{{ cmd.placeholder }}</span>
              <span class="command-desc">{{ cmd.description }}</span>
            </button>
          </div>

          <div v-if="groupedCommands.length === 0" class="sheet-empty" data-test="sheet-empty">
            未找到匹配的命令
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.command-sheet-overlay {
  position: fixed;
  inset: 0;
  z-index: 5000;
  background: var(--color-overlay);
  animation: fadeIn var(--transition-fast) ease;
  display: flex;
  align-items: flex-end;
}

.command-sheet {
  width: 100%;
  max-height: 85vh;
  max-height: 85dvh;
  background: var(--color-bg-primary);
  border-radius: var(--radius-xl) var(--radius-xl) 0 0;
  display: flex;
  flex-direction: column;
  animation: slideUp 0.3s ease;
  padding-bottom: env(safe-area-inset-bottom, 0px);
  overflow: hidden;
}

.sheet-drag-handle {
  display: flex;
  justify-content: center;
  padding: var(--space-sm) 0;
}

.drag-bar {
  width: 36px;
  height: 4px;
  border-radius: 2px;
  background: var(--color-border);
}

.sheet-search {
  padding: 0 var(--space-lg) var(--space-md);
  display: flex;
  align-items: center;
  gap: var(--space-sm);
}

.search-icon {
  flex-shrink: 0;
  color: var(--color-text-secondary);
}

.search-input {
  flex: 1;
  padding: 10px var(--space-md);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-bg-secondary);
  color: var(--color-text-primary);
  font-size: 15px;
  outline: none;
  min-height: 44px;
}

.search-input:focus {
  border-color: var(--color-accent);
}

.sheet-body {
  flex: 1;
  overflow-y: auto;
  padding: 0 var(--space-lg) var(--space-lg);
  -webkit-overflow-scrolling: touch;
}

.command-group {
  margin-bottom: var(--space-lg);
}

.group-label {
  font-size: var(--font-size-xs);
  font-weight: 600;
  color: var(--color-text-tertiary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
  padding: var(--space-xs) 0;
  margin-bottom: var(--space-xs);
  border-bottom: 1px solid var(--color-border-light);
}

.command-item {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
  width: 100%;
  padding: var(--space-md);
  border: none;
  border-radius: var(--radius-md);
  background: transparent;
  cursor: pointer;
  text-align: left;
  transition: background var(--transition-fast);
  min-height: 44px;
  -webkit-tap-highlight-color: transparent;
}

.command-icon {
  color: var(--color-text-tertiary);
  flex-shrink: 0;
}

.command-item:active {
  background: var(--color-bg-hover);
}

.command-name {
  font-size: var(--font-size-base);
  color: var(--color-text-primary);
  font-weight: 500;
  font-family: var(--font-mono);
  white-space: nowrap;
  flex-shrink: 0;
}

.command-placeholder {
  font-size: var(--font-size-xs);
  color: var(--color-text-tertiary);
  font-family: var(--font-mono);
  white-space: nowrap;
}

.command-desc {
  font-size: var(--font-size-xs);
  color: var(--color-text-secondary);
  margin-left: auto;
  white-space: nowrap;
}

.sheet-empty {
  text-align: center;
  padding: var(--space-2xl);
  color: var(--color-text-tertiary);
  font-size: var(--font-size-base);
}

@keyframes slideUp {
  from {
    transform: translateY(100%);
  }
  to {
    transform: translateY(0);
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