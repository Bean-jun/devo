<script setup lang="ts">
import { ref, computed, watch, onMounted, nextTick } from 'vue'
import { useCommandStore } from '@/stores/command'
import { useUiStore } from '@/stores/ui'
import { MAX_MESSAGE_LENGTH } from '@/utils/constants'
import { estimateTokens } from '@/utils/formatters'

const props = defineProps<{
  isDisabled: boolean
  isProcessing: boolean
}>()

const emit = defineEmits<{
  send: [text: string]
  stop: []
  clear: []
  openCommand: []
}>()

const inputText = ref('')
const textareaRef = ref<HTMLTextAreaElement | null>(null)
const commandStore = useCommandStore()
const uiStore = useUiStore()

const charCount = computed(() => inputText.value.length)
const tokenEstimate = computed(() => estimateTokens(inputText.value))
const canSend = computed(() => inputText.value.trim().length > 0 && !props.isDisabled)

function focusTextarea(): void {
  nextTick(() => {
    textareaRef.value?.focus()
  })
}

onMounted(focusTextarea)

watch(() => uiStore.focusInputCounter, focusTextarea)

watch(() => commandStore.isOpen, (open) => {
  if (!open) focusTextarea()
})

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    send()
  } else if (e.key === '/' && inputText.value === '') {
    e.preventDefault()
    emit('openCommand')
  }
}

function send() {
  const text = inputText.value.trim()
  if (!text || !canSend.value) return
  emit('send', text)
  inputText.value = ''
  autoResize()
}

function autoResize() {
  const el = textareaRef.value
  if (!el) return
  el.style.height = 'auto'
  el.style.height = Math.min(el.scrollHeight, 200) + 'px'
}

watch(inputText, () => {
  autoResize()
})
</script>

<template>
  <div class="input-area">
    <div class="input-wrapper">
      <textarea
        ref="textareaRef"
        v-model="inputText"
        class="input-field"
        data-test="message-input"
        placeholder="输入消息，或按 / 使用命令..."
        :disabled="isDisabled"
        :maxlength="MAX_MESSAGE_LENGTH"
        rows="1"
        @keydown="handleKeydown"
      />

      <div class="input-actions">
        <span class="input-info">
          <span v-if="charCount > 0" class="char-count" data-test="char-count">
            {{ charCount }} / {{ MAX_MESSAGE_LENGTH }}
            <span class="token-estimate">~{{ tokenEstimate }} tokens</span>
          </span>
        </span>

        <button
          v-if="isProcessing"
          class="btn-stop"
          data-test="stop-button"
          aria-label="停止"
          @click="emit('stop')"
        >
          <span class="stop-icon">■</span>
          停止
        </button>

        <button
          v-else
          class="btn-send"
          :disabled="!canSend"
          aria-label="发送"
          @click="send"
        >
          发送
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.input-area {
  flex-shrink: 0;
  padding: var(--space-md) var(--space-lg);
  padding-bottom: var(--space-lg);
  background: var(--color-bg-primary);
  border-top: 1px solid var(--color-border-light);
}

.input-wrapper {
  max-width: 800px;
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  gap: var(--space-sm);
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: var(--space-sm) var(--space-md);
  transition: border-color var(--transition-fast);
}

.input-wrapper:focus-within {
  border-color: var(--color-accent);
  box-shadow: 0 0 0 3px var(--color-accent-light);
}

.input-field {
  width: 100%;
  min-height: 24px;
  max-height: 200px;
  resize: none;
  font-size: var(--font-size-base);
  line-height: 1.5;
  color: var(--color-text-primary);
  background: transparent;
  border: none;
  outline: none;
  padding: 0;
}

.input-field::placeholder {
  color: var(--color-text-tertiary);
}

.input-field:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.input-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: var(--space-sm);
}

.input-info {
  flex: 1;
  display: flex;
  align-items: center;
}

.char-count {
  font-size: var(--font-size-xs);
  color: var(--color-text-tertiary);
  font-family: var(--font-mono);
}

.token-estimate {
  margin-left: var(--space-sm);
  color: var(--color-text-tertiary);
  opacity: 0.7;
}

.btn-send,
.btn-stop {
  display: flex;
  align-items: center;
  gap: var(--space-xs);
  padding: var(--space-xs) var(--space-md);
  border-radius: var(--radius-md);
  font-size: var(--font-size-sm);
  font-weight: 500;
  transition: all var(--transition-fast);
}

.btn-send {
  background: var(--color-accent);
  color: var(--color-text-inverse);
}

.btn-send:hover:not(:disabled) {
  background: var(--color-accent-hover);
}

.btn-send:disabled {
  background: var(--color-bg-tertiary);
  color: var(--color-text-tertiary);
}

.btn-stop {
  background: var(--color-error);
  color: var(--color-text-inverse);
}

.btn-stop:hover {
  background: #e0352b;
}

.stop-icon {
  font-size: 10px;
}
</style>