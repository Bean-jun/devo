<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useUiStore } from '@/stores/ui'

const uiStore = useUiStore()

const inputValue = ref('')
const inputRef = ref<HTMLInputElement | null>(null)
const submitting = ref(false)

const isOpen = computed(() => uiStore.activeModal === 'input-prompt')

watch(isOpen, (val) => {
  if (val) {
    submitting.value = false
    inputValue.value = uiStore.inputPrompt.defaultValue || ''
    inputRef.value?.focus()
    inputRef.value?.select()
  }
})

function handleConfirm() {
  if (submitting.value) return
  const value = inputValue.value.trim()
  if (value) {
    submitting.value = true
    uiStore.inputPrompt.onConfirm(value)
    uiStore.setActiveModal(null)
  }
}

function handleCancel() {
  uiStore.inputPrompt.onCancel?.()
  uiStore.setActiveModal(null)
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter') {
    e.preventDefault()
    handleConfirm()
  } else if (e.key === 'Escape') {
    e.preventDefault()
    handleCancel()
  }
}
</script>

<template>
  <div v-show="isOpen" class="modal-overlay" @click="handleCancel" @keydown="handleKeydown">
    <div class="input-modal" @click.stop>
      <div class="modal-header">
        <h3>{{ uiStore.inputPrompt.title }}</h3>
      </div>
      <div class="modal-body">
        <input
          ref="inputRef"
          v-model="inputValue"
          type="text"
          :placeholder="uiStore.inputPrompt.placeholder || ''"
          class="input-field"
          @keydown="handleKeydown"
        />
      </div>
      <div class="modal-footer">
        <button class="btn btn-cancel" @click="handleCancel">取消</button>
        <button class="btn btn-confirm" @click="handleConfirm">{{ uiStore.inputPrompt.confirmLabel || '确定' }}</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  z-index: 6000;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-overlay);
  animation: fadeIn var(--transition-fast) ease;
}

.input-modal {
  width: 400px;
  background: var(--color-bg-primary);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-modal);
  animation: modalIn var(--transition-base) ease;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-lg);
  border-bottom: 1px solid var(--color-border-light);
}

.modal-header h3 {
  font-size: var(--font-size-lg);
  font-weight: 600;
  margin: 0;
}

.modal-body {
  padding: var(--space-lg);
}

.input-field {
  width: 100%;
  padding: var(--space-sm) var(--space-md);
  font-size: var(--font-size-base);
  font-family: var(--font-mono);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-bg-tertiary);
  color: var(--color-text-primary);
  outline: none;
  transition: border-color var(--transition-fast) ease;
  box-sizing: border-box;
}

.input-field:focus {
  border-color: var(--color-accent);
}

.input-field::placeholder {
  color: var(--color-text-tertiary);
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-sm);
  padding: var(--space-md) var(--space-lg);
  border-top: 1px solid var(--color-border-light);
}

.btn {
  padding: var(--space-xs) var(--space-md);
  font-size: var(--font-size-sm);
  border-radius: var(--radius-md);
  cursor: pointer;
  border: none;
  transition: background var(--transition-fast) ease;
}

.btn-cancel {
  background: var(--color-bg-tertiary);
  color: var(--color-text-secondary);
}

.btn-cancel:hover {
  background: var(--color-bg-hover);
}

.btn-confirm {
  background: var(--color-accent);
  color: #fff;
}

.btn-confirm:hover {
  filter: brightness(1.1);
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

@keyframes modalIn {
  from { opacity: 0; transform: scale(0.95) translateY(-8px); }
  to { opacity: 1; transform: scale(1) translateY(0); }
}
</style>