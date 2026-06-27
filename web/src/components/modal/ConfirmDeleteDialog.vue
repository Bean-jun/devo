<script setup lang="ts">
import { ref, watch, nextTick } from 'vue'

const props = defineProps<{
  visible: boolean
  serverName: string
  deleting: boolean
  entityType?: string
}>()

const emit = defineEmits<{
  confirm: []
  cancel: []
}>()

const typeLabel = props.entityType || 'MCP 服务器'

const inputValue = ref('')
const inputRef = ref<HTMLInputElement | null>(null)

watch(() => props.visible, (val) => {
  if (val) {
    inputValue.value = ''
    nextTick(() => {
      inputRef.value?.focus()
    })
  }
})

const isMatch = () => inputValue.value.trim() === props.serverName

function handleConfirm() {
  if (!isMatch() || props.deleting) return
  emit('confirm')
}
</script>

<template>
  <div v-if="visible" class="modal-overlay" @click.self="emit('cancel')">
    <div class="confirm-dialog" @click.stop>
      <div class="dialog-header">
        <div class="dialog-icon">!</div>
        <h3 class="dialog-title">删除{{ typeLabel }}</h3>
      </div>

      <div class="dialog-body">
        <p class="dialog-warning">
          此操作不可撤销。将永久删除{{ typeLabel }}
          <strong>{{ serverName }}</strong>
          及其所有工具配置。
        </p>
        <p class="dialog-hint">
          请输入名称 <code>{{ serverName }}</code> 以确认删除：
        </p>
        <input
          ref="inputRef"
          v-model="inputValue"
          type="text"
          class="dialog-input"
          :placeholder="serverName"
          @keydown.enter="handleConfirm"
        />
      </div>

      <div class="dialog-footer">
        <button class="btn-cancel" @click="emit('cancel')">取消</button>
        <button
          class="btn-delete"
          :disabled="!isMatch() || deleting"
          @click="handleConfirm"
        >
          {{ deleting ? '删除中...' : '确认删除' }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  z-index: 1000;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.5);
  backdrop-filter: blur(2px);
}

.confirm-dialog {
  width: 420px;
  max-width: 90vw;
  background: var(--color-bg-primary);
  border: 1px solid var(--color-border);
  border-radius: 12px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
  overflow: hidden;
}

.dialog-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 16px 20px;
  border-bottom: 1px solid var(--color-border);
  background: rgba(239, 68, 68, 0.06);
}

.dialog-icon {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: var(--color-error, #ef4444);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
  font-weight: 700;
  flex-shrink: 0;
}

.dialog-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin: 0;
}

.dialog-body {
  padding: 20px;
}

.dialog-warning {
  font-size: 13px;
  color: var(--color-text-secondary);
  margin: 0 0 12px;
  line-height: 1.5;
}

.dialog-warning strong {
  color: var(--color-error, #ef4444);
  font-weight: 600;
}

.dialog-hint {
  font-size: 12px;
  color: var(--color-text-tertiary);
  margin: 0 0 8px;
}

.dialog-hint code {
  font-family: var(--font-mono);
  background: var(--color-bg-secondary);
  padding: 1px 6px;
  border-radius: 3px;
  font-size: 12px;
  color: var(--color-error, #ef4444);
}

.dialog-input {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  background: var(--color-bg-primary);
  color: var(--color-text-primary);
  font-size: 13px;
  font-family: var(--font-mono);
  box-sizing: border-box;
  transition: border-color var(--transition-fast);
}

.dialog-input:focus {
  outline: none;
  border-color: var(--color-error, #ef4444);
  box-shadow: 0 0 0 3px rgba(239, 68, 68, 0.12);
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding: 12px 20px;
  border-top: 1px solid var(--color-border);
  background: var(--color-bg-secondary);
}

.btn-cancel {
  padding: 7px 16px;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  background: var(--color-bg-primary);
  color: var(--color-text-secondary);
  cursor: pointer;
  font-size: 13px;
  transition: all var(--transition-fast);
}

.btn-cancel:hover {
  background: var(--color-bg-hover);
}

.btn-delete {
  padding: 7px 16px;
  border: none;
  border-radius: 6px;
  background: var(--color-error, #ef4444);
  color: #fff;
  cursor: pointer;
  font-size: 13px;
  font-weight: 500;
  transition: all var(--transition-fast);
}

.btn-delete:hover:not(:disabled) {
  background: #dc2626;
}

.btn-delete:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
</style>