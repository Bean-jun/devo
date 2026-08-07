<script setup lang="ts">
import { useFloatingNavPanel } from './FloatingNavPanelController'

const props = defineProps<{
  scrollToMessage: (messageId: string) => void
}>()

const {
  navItems,
  activeId,
  handleClick,
} = useFloatingNavPanel(props)
</script>

<template>
  <div v-if="navItems.length > 0" class="nav-panel">
    <div class="nav-list">
      <button
        v-for="item in navItems"
        :key="item.id"
        class="nav-pill"
        :class="{ active: item.id === activeId }"
        @click="handleClick(item.id)"
      >
        <span class="nav-pill-label">{{ item.summary }}</span>
      </button>
    </div>
  </div>
</template>

<style scoped src="./FloatingNavPanel.css">
</style>