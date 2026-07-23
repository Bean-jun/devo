<script setup lang="ts">
import { ref, computed, watch, onMounted, nextTick } from 'vue'
import { useFps } from '@/composables/useFps'
import { useCommandStore } from '@/stores/command'
import { useSessionStore } from '@/stores/session'
import { useUiStore } from '@/stores/ui'
import { MAX_MESSAGE_LENGTH } from '@/utils/constants'
import { estimateTokens, formatTokenCount } from '@/utils/formatters'
import AppIcon from '@/components/common/AppIcon.vue'
import {
  useInputSegments,
  createPasteSegment,
  parseDOMToSegments,
  chipLabel,
  shouldFoldPaste,
  type PasteSegment,
} from '@/composables/useInputSegments'

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

const editorRef = ref<HTMLDivElement | null>(null)
const commandStore = useCommandStore()
const sessionStore = useSessionStore()
const uiStore = useUiStore()

const {
  segments,
  nextPasteId,
  reset,
  setText,
  serialize,
  totalLength,
  isEmpty,
} = useInputSegments()

const pasteMap = ref(new Map<number, PasteSegment>())
const isComposing = ref(false)
let isPatchingDOM = false

const inputHistory: string[] = []
let historyIndex = -1

const charCount = computed(() => totalLength())
const tokenEstimate = computed(() => estimateTokens(serialize()))
const canSend = computed(() => !isEmpty() && !props.isDisabled)
const showPlaceholder = computed(() => isEmpty())

const contextUsage = computed(() => {
  const tokens = sessionStore.currentSession?.currentContextTokens ?? 0
  return formatTokenCount(tokens)
})

const sessionTokenUsage = computed(() => {
  const usage = sessionStore.currentSession?.tokenUsage
  const input = usage?.input ?? 0
  const output = usage?.output ?? 0
  const total = input + output
  return `${formatTokenCount(total)} (↑${formatTokenCount(input)} ↓${formatTokenCount(output)})`
})

const appVersion = import.meta.env.VITE_APP_VERSION || 'dev'

const { fps } = useFps()

const workingDir = computed(() => sessionStore.currentSession?.workingDirectory ?? '')

function focusEditor(): void {
  nextTick(() => {
    editorRef.value?.focus()
  })
}

onMounted(() => {
  renderEditor()
  focusEditor()
})

watch(() => uiStore.focusInputCounter, focusEditor)

watch(() => commandStore.isOpen, (open) => {
  if (!open) focusEditor()
})

watch(() => uiStore.pendingCommand, (cmd) => {
  if (cmd) {
    setText(cmd)
    uiStore.clearPendingCommand()
    renderEditor()
    moveCursorToEnd()
    focusEditor()
  }
})

function renderEditor(): void {
  const editor = editorRef.value
  if (!editor) return
  isPatchingDOM = true
  editor.textContent = ''
  for (const seg of segments.value) {
    if (seg.kind === 'text') {
      if (seg.value) editor.appendChild(document.createTextNode(seg.value))
    } else {
      const span = document.createElement('span')
      span.contentEditable = 'false'
      span.dataset.pasteId = String(seg.id)
      span.className = 'paste-chip'
      span.textContent = chipLabel(seg)
      editor.appendChild(span)
    }
  }
  nextTick(() => {
    isPatchingDOM = false
    autoResize()
  })
}

function syncFromDOM(): void {
  const editor = editorRef.value
  if (!editor) return
  segments.value = parseDOMToSegments(editor, pasteMap.value)
}

function handleInput(): void {
  if (isPatchingDOM) return
  if (isComposing.value) return
  syncFromDOM()
  autoResize()
}

function handlePaste(e: ClipboardEvent): void {
  if (isComposing.value) return
  const pasted = e.clipboardData?.getData('text') || ''
  if (!pasted) return

  e.preventDefault()

  const editor = editorRef.value
  if (!editor) return

  const sel = window.getSelection()
  if (!sel || !sel.rangeCount) return

  const range = sel.getRangeAt(0)
  range.deleteContents()

  if (shouldFoldPaste(pasted)) {
    const id = nextPasteId()
    const pasteSeg = createPasteSegment(id, pasted)
    pasteMap.value.set(id, pasteSeg)

    const span = document.createElement('span')
    span.contentEditable = 'false'
    span.dataset.pasteId = String(id)
    span.className = 'paste-chip'
    span.textContent = chipLabel(pasteSeg)

    range.insertNode(span)

    const newRange = document.createRange()
    newRange.setStartAfter(span)
    newRange.collapse(true)
    sel.removeAllRanges()
    sel.addRange(newRange)
  } else {
    document.execCommand('insertText', false, pasted)
  }

  syncFromDOM()
  autoResize()
}

function insertNewline(): void {
  const editor = editorRef.value
  if (!editor) return
  editor.focus()
  document.execCommand('insertLineBreak')
  syncFromDOM()
  autoResize()
}

function handleCompositionStart(): void {
  isComposing.value = true
}

function handleCompositionEnd(): void {
  isComposing.value = false
  syncFromDOM()
  autoResize()
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    if (!isComposing.value) send()
  } else if (e.key === 'Enter' && e.shiftKey) {
    e.preventDefault()
    insertNewline()
  } else if (e.key === '/' && isEmpty()) {
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

function moveCursorToEnd(): void {
  const editor = editorRef.value
  if (!editor) return
  const range = document.createRange()
  range.selectNodeContents(editor)
  range.collapse(false)
  const sel = window.getSelection()
  if (!sel) return
  sel.removeAllRanges()
  sel.addRange(range)
}

function pushHistory(text: string): void {
  if (!text) return
  if (inputHistory.length > 0 && inputHistory[inputHistory.length - 1] === text) return
  inputHistory.push(text)
  historyIndex = inputHistory.length
}

function historyPrev(): void {
  if (inputHistory.length === 0) return
  if (historyIndex > 0) {
    historyIndex--
    setText(inputHistory[historyIndex])
    renderEditor()
    moveCursorToEnd()
  }
}

function historyNext(): void {
  if (inputHistory.length === 0) return
  if (historyIndex < inputHistory.length - 1) {
    historyIndex++
    setText(inputHistory[historyIndex])
    renderEditor()
    moveCursorToEnd()
  } else {
    historyIndex = inputHistory.length
    setText('')
    renderEditor()
  }
}

function send(): void {
  if (!canSend.value) return
  const text = serialize()
  pushHistory(text)
  if (text.startsWith('/')) {
    emit('executeCommand', text)
  } else {
    emit('send', text)
  }
  reset()
  pasteMap.value.clear()
  renderEditor()
}

function autoResize(): void {
  const el = editorRef.value
  if (!el) return
  el.style.height = 'auto'
  el.style.height = Math.min(el.scrollHeight, 200) + 'px'
}
</script>

<template>
  <div class="input-area">
    <div class="input-row">
      <button
        class="command-btn"
        aria-label="命令面板"
        data-test="command-btn"
        @click="emit('openCommand')"
      >
        <span class="command-btn-text">/</span>
      </button>

      <div class="input-wrapper">
        <div
          ref="editorRef"
          class="input-field"
          :class="{ 'is-empty': showPlaceholder, 'is-disabled': isDisabled }"
          contenteditable="true"
          data-test="message-input"
          role="textbox"
          aria-multiline="true"
          data-placeholder="输入消息，或按 / 使用命令..."
          @input="handleInput"
          @keydown="handleKeydown"
          @paste="handlePaste"
          @compositionstart="handleCompositionStart"
          @compositionend="handleCompositionEnd"
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
            <AppIcon name="stop" :size="14" class="stop-icon" />
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
    <div class="input-footer">
      <span class="footer-item">Context </span><span class="footer-item context-warn">{{ contextUsage }}</span>
      <span class="footer-sep">·</span>
      <span class="footer-item">Tokens {{ sessionTokenUsage }}</span>
      <span class="footer-sep">·</span>
      <span class="fps-counter" :class="{ 'fps-low': fps < 30, 'fps-good': fps >= 55 }">
        {{ fps }} FPS
      </span>
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
  user-select: none;
}

.footer-item {
  opacity: 0.85;
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

.fps-counter {
  font-size: 10px;
  color: var(--color-text-secondary);
  font-family: var(--font-mono);
  white-space: nowrap;
  transition: color var(--transition-fast);
  opacity: 0.85;
}

.fps-counter.fps-good {
  color: var(--color-success);
}

.fps-counter.fps-low {
  color: var(--color-error);
}

.context-warn {
  color: #ff9500 !important;
}

.input-row {
  max-width: 800px;
  margin: 0 auto;
  display: flex;
  align-items: stretch;
  gap: var(--space-sm);
}

.command-btn {
  flex-shrink: 0;
  width: 36px;
  min-height: 36px;
  border-radius: var(--radius-md);
  background: transparent;
  border: 1px solid var(--color-border);
  color: var(--color-text-tertiary);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all var(--transition-fast);
  padding: 0;
}

.command-btn:hover {
  border-color: var(--color-accent);
  color: var(--color-accent);
  background: var(--color-accent-light);
}

.command-btn:active {
  transform: scale(0.95);
}

.command-btn-text {
  font-size: 14px;
  font-weight: 600;
  line-height: 1;
}

.input-wrapper {
  flex: 1;
  min-width: 0;
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
  overflow-y: auto;
  font-size: var(--font-size-base);
  line-height: 1.5;
  color: var(--color-text-primary);
  background: transparent;
  border: none;
  outline: none;
  padding: 0;
  white-space: pre-wrap;
  word-break: break-word;
}

.input-field.is-empty::before {
  content: attr(data-placeholder);
  color: var(--color-text-tertiary);
  pointer-events: none;
}

.input-field.is-disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.paste-chip {
  display: inline-flex;
  align-items: center;
  padding: 1px 6px;
  margin: 0 2px;
  background: var(--color-bg-tertiary);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  color: var(--color-text-secondary);
  font-size: var(--font-size-xs);
  font-family: var(--font-mono);
  white-space: nowrap;
  user-select: none;
  vertical-align: baseline;
  cursor: default;
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
