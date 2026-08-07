<script setup lang="ts">
import AppIcon from '@/components/common/AppIcon.vue'
import { useReasoningEffortToggle } from './ReasoningEffortToggleController'

const { reasoningStore, open, EFFORT_OPTIONS, currentLabel, isSelected, select, toggle, onBlur } = useReasoningEffortToggle()
</script>

<template>
  <div class="reasoning-toggle" @focusout="onBlur">
    <button
      class="reasoning-btn"
      :class="{ active: reasoningStore.enableReasoning }"
      :title="reasoningStore.enableReasoning ? `思考模式：${currentLabel}` : '思考模式：关闭'"
      @click="toggle"
    >
      <AppIcon
        name="brain"
        :size="14"
        :weight="reasoningStore.enableReasoning ? 'fill' : 'regular'"
      />
      <span class="reasoning-label">{{ currentLabel }}</span>
      <AppIcon :name="open ? 'caret-up' : 'caret-down'" :size="10" class="reasoning-arrow" />
    </button>

    <div v-if="open" class="reasoning-dropdown">
      <button
        v-for="opt in EFFORT_OPTIONS"
        :key="opt.value"
        class="reasoning-option"
        :class="{ selected: isSelected(opt) }"
        @click="select(opt)"
      >
        <AppIcon :name="opt.icon" :size="14" />
        <span>{{ opt.label }}</span>
        <AppIcon
          v-if="isSelected(opt)"
          name="check"
          :size="12"
          class="option-check"
        />
      </button>
    </div>
  </div>
</template>

<style scoped src="./ReasoningEffortToggle.css">
</style>