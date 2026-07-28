<script setup lang="ts">
import { ref, computed } from 'vue'
import { useReasoningStore, type ReasoningEffortLevel } from '@/stores/reasoning'
import AppIcon from '@/components/common/AppIcon.vue'

const reasoningStore = useReasoningStore()

const open = ref(false)

interface EffortOption {
  value: 'off' | ReasoningEffortLevel
  label: string
  icon: 'prohibit' | 'brain'
}

const EFFORT_OPTIONS: EffortOption[] = [
  { value: 'off', label: '关闭', icon: 'prohibit' },
  { value: 'low', label: '低', icon: 'brain' },
  { value: 'medium', label: '中', icon: 'brain' },
  { value: 'high', label: '高', icon: 'brain' },
]

const currentLabel = computed(() => {
  if (!reasoningStore.enableReasoning) return '思考'
  const opt = EFFORT_OPTIONS.find(o => o.value === reasoningStore.reasoningEffort)
  return `思考 · ${opt?.label ?? ''}`
})

function isSelected(opt: EffortOption): boolean {
  if (opt.value === 'off') {
    return !reasoningStore.enableReasoning
  }
  return reasoningStore.enableReasoning && reasoningStore.reasoningEffort === opt.value
}

function select(opt: EffortOption) {
  if (opt.value === 'off') {
    reasoningStore.setEnableReasoning(false)
  } else {
    reasoningStore.setEnableReasoning(true)
    reasoningStore.setReasoningEffort(opt.value as ReasoningEffortLevel)
  }
  open.value = false
}

function toggle() {
  open.value = !open.value
}

function onBlur(e: FocusEvent) {
  const target = e.relatedTarget as HTMLElement | null
  if (!target || !target.closest('.reasoning-toggle')) {
    open.value = false
  }
}
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

<style scoped>
.reasoning-toggle {
  position: relative;
}

.reasoning-btn {
  display: flex;
  align-items: center;
  gap: 4px;
  height: 24px;
  padding: 0 8px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text-secondary);
  font-size: var(--font-size-xs);
  cursor: pointer;
  white-space: nowrap;
  transition: all var(--transition-fast) ease;
}

.reasoning-btn:hover {
  background: var(--color-bg-hover);
  color: var(--color-text-primary);
}

.reasoning-btn.active {
  border-color: var(--color-accent);
  color: var(--color-accent);
}

.reasoning-label {
  font-weight: 500;
}

.reasoning-arrow {
  opacity: 0.6;
  flex-shrink: 0;
}

.reasoning-dropdown {
  position: absolute;
  top: calc(100% + 4px);
  right: 0;
  min-width: 120px;
  background: var(--color-bg-primary);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-lg);
  padding: 4px;
  z-index: 100;
}

.reasoning-option {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 6px 10px;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text-primary);
  font-size: var(--font-size-sm);
  cursor: pointer;
  transition: background var(--transition-fast) ease;
}

.reasoning-option:hover {
  background: var(--color-bg-hover);
}

.reasoning-option.selected {
  color: var(--color-accent);
}

.option-check {
  margin-left: auto;
  color: var(--color-accent);
}
</style>