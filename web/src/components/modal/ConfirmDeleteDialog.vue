<script setup lang="ts">
import { useConfirmDeleteDialog } from './ConfirmDeleteDialogController'

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

const {
  typeLabel,
  inputValue,
  inputRef,
  isMatch,
  handleConfirm,
} = useConfirmDeleteDialog(props, emit as any)
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

<style scoped src="./ConfirmDeleteDialog.css">
</style>