<script setup lang="ts">
import { ref, computed, watch, onMounted, nextTick } from 'vue'
import { useCommandStore } from '@/stores/command'
import { useSessionStore } from '@/stores/session'
import { useUiStore } from '@/stores/ui'
import { MAX_MESSAGE_LENGTH } from '@/utils/constants'
import { estimateTokens, formatTokenCount } from '@/utils/formatters'

const props = defineProps<{
  isDisabled: boolean
  isProcessing: boolean
}>()

const emit = defineEmits<{
  send: [text: string]
  stop: []
  clear: []
  openCommand: []
  executeCommand: [text: string]
}>()

const inputText = ref('')
const textareaRef = ref<HTMLTextAreaElement | null>(null)
const commandStore = useCommandStore()
const sessionStore = useSessionStore()
const uiStore = useUiStore()

const inputHistory: string[] = []
let historyIndex = -1

const pastedFullText = ref('')
const pasteLabel = ref('')
const PASTE_THRESHOLD = 200

const charCount = computed(() => inputText.value.length)
const tokenEstimate = computed(() => estimateTokens(inputText.value))
const canSend = computed(() => inputText.value.trim().length > 0 && !props.isDisabled)

const contextUsage = computed(() => {
  const tokens = sessionStore.currentSession?.currentContextTokens
  return formatTokenCount(tokens ?? 0)
})

const sessionTokenUsage = computed(() => {
  const usage = sessionStore.currentSession?.tokenUsage
  const input = usage?.input ?? 0
  const output = usage?.output ?? 0
  const total = input + output
  return `${formatTokenCount(total)} (↑${formatTokenCount(input)} ↓${formatTokenCount(output)})`
})

const appVersion = import.meta.env.VITE_APP_VERSION || 'dev'

const workingDir = computed(() => sessionStore.currentSession?.workingDirectory ?? '')

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

watch(() => uiStore.pendingCommand, (cmd) => {
  if (cmd) {
    inputText.value = cmd
    uiStore.clearPendingCommand()
    focusTextarea()
  }
})

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    send()
  } else if (e.key === '/' && inputText.value === '') {
    e.preventDefault()
    emit('openCommand')
  } else if (e.key === 'ArrowUp' && e.shiftKey) {
    e.preventDefault()
    historyPrev()
  } else if (e.key === 'ArrowDown' && e.shiftKey) {
    e.preventDefault()
    historyNext()
  }
}

function pushHistory(text: string) {
  if (!text) return
  if (inputHistory.length > 0 && inputHistory[inputHistory.length - 1] === text) return
  inputHistory.push(text)
  historyIndex = inputHistory.length
}

function historyPrev() {
  if (inputHistory.length === 0) return
  if (historyIndex > 0) {
    historyIndex--
    inputText.value = inputHistory[historyIndex]
  }
}

function historyNext() {
  if (inputHistory.length === 0) return
  if (historyIndex < inputHistory.length - 1) {
    historyIndex++
    inputText.value = inputHistory[historyIndex]
  } else {
    historyIndex = inputHistory.length
    inputText.value = ''
  }
}

function send() {
  const rawText = inputText.value.trim()
  if (!rawText || !canSend.value) return

  const text = (pastedFullText.value && rawText === pasteLabel.value)
    ? pastedFullText.value
    : rawText

  pushHistory(text)
  if (text.startsWith('/')) {
    emit('executeCommand', text)
  } else {
    emit('send', text)
  }
  inputText.value = ''
  pastedFullText.value = ''
  pasteLabel.value = ''
  autoResize()
}

function handlePaste(e: ClipboardEvent) {
  const pasted = e.clipboardData?.getData('text') || ''
  if (!pasted) return

  const textarea = textareaRef.value
  if (!textarea) return

  const currentText = inputText.value
  const start = textarea.selectionStart
  const end = textarea.selectionEnd

  const newText = currentText.slice(0, start) + pasted + currentText.slice(end)
  const lines = newText.split('\n')

  if (newText.length > PASTE_THRESHOLD || lines.length > 3) {
    e.preventDefault()
    pastedFullText.value = newText
    const prefix = currentText.slice(0, start)
    const suffix = currentText.slice(end)
    pasteLabel.value = prefix + `[已粘贴 ${lines.length} 行文本]` + suffix
    inputText.value = pasteLabel.value
  }
}

watch(inputText, (val) => {
  if (pasteLabel.value && val !== pasteLabel.value) {
    pastedFullText.value = ''
    pasteLabel.value = ''
  }
  autoResize()
})

function autoResize() {
  const el = textareaRef.value
  if (!el) return
  el.style.height = 'auto'
  el.style.height = Math.min(el.scrollHeight, 200) + 'px'
}
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
        @paste="handlePaste"
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
    <div class="input-footer">
      <span class="footer-item">context {{ contextUsage }}</span>
      <span class="footer-sep">·</span>
      <span class="footer-item">Tokens {{ sessionTokenUsage }}</span>
      <span class="footer-sep">·</span>
      <span class="footer-item">v{{ appVersion }}</span>
      <span v-if="workingDir" class="footer-item footer-dir">{{ workingDir }}</span>
    </div>
  </div>
</template>

<style scoped>
.input-area {
  flex-shrink: 0;
  padding: var(--space-md) var(--space-lg);
  padding-bottom: var(--space-xs);
  background: var(--color-bg-primary);
  border-top: 1px solid var(--color-border-light);
}

.input-footer {
  max-width: 800px;
  margin: var(--space-xs) auto 0;
  padding: 0 var(--space-md);
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 10px;
  font-family: var(--font-mono);
  color: var(--color-text-secondary);
  opacity: 0.85;
  user-select: none;
}

.footer-sep {
  opacity: 0.4;
}

.footer-dir {
  margin-left: auto;
  max-width: 300px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
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