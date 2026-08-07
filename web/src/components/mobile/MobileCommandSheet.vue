<script setup lang="ts">
import AppIcon from '@/components/common/AppIcon.vue'
import type { Command } from '@/stores/command'
import { useMobileCommandSheet } from './MobileCommandSheetController'

const emit = defineEmits<{
  select: [cmd: Command]
  close: []
}>()

const {
  searchQuery,
  groupedCommands,
  selectCommand,
  onBackdropClick,
  onSheetClick,
  onTouchStart,
  onTouchMove,
} = useMobileCommandSheet(emit as any)
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

<style scoped src="./MobileCommandSheet.css">
</style>