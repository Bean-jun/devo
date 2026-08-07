<script setup lang="ts">
import AppIcon from '@/components/common/AppIcon.vue'
import { useCommandPalette } from './CommandPaletteController'

const {
  commandStore,
  inputRef,
  listRef,
  handleInput,
  handleKeydown,
  handleSelect,
} = useCommandPalette()
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
        <span><AppIcon name="arrow-up" :size="12" /><AppIcon name="arrow-down" :size="12" /> 导航</span>
        <span>Enter 选择</span>
        <span>Esc 关闭</span>
      </div>
    </div>
  </div>
</template>

<style scoped src="./CommandPalette.css">
</style>