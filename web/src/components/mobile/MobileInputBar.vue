<script setup lang="ts">
import { ref, computed, watch, onMounted, nextTick } from 'vue'
import { useSessionStore } from '@/stores/session'
import { useUiStore } from '@/stores/ui'
import { MAX_MESSAGE_LENGTH } from '@/utils/constants'
import { estimateTokens, formatTokenCount } from '@/utils/formatters'
import AppIcon from '@/components/common/AppIcon.vue'

const props = defineProps<{
  isDisabled: boolean
  isProcessing: boolean
}>()

const emit = defineEmits<{
  send: [text: string, images?: string[]]
  stop: []
  openCommand: []
}>()

const inputText = ref('')
const textareaRef = ref<HTMLTextAreaElement | null>(null)
const fileInputRef = ref<HTMLInputElement | null>(null)
const sessionStore = useSessionStore()
const uiStore = useUiStore()

const inputHistory: string[] = []
let historyIndex = -1

const pastedFullText = ref('')
const pasteLabel = ref('')
const PASTE_THRESHOLD = 200

const uploadedImages = ref<string[]>([])
const lightboxImage = ref<string | null>(null)

function readImageAsBase64(file: File): void {
  const reader = new FileReader()
  reader.onload = () => {
    const dataUrl = reader.result as string
    uploadedImages.value.push(dataUrl)
  }
  reader.readAsDataURL(file)
}

function removeImage(index: number): void {
  uploadedImages.value.splice(index, 1)
}

const charCount = computed(() => inputText.value.length)
const tokenEstimate = computed(() => estimateTokens(inputText.value))
const canSend = computed(() => (inputText.value.trim().length > 0 || uploadedImages.value.length > 0) && !props.isDisabled)

const contextUsage = computed(() => {
  const tokens = sessionStore.currentSession?.currentContextTokens ?? 0
  return formatTokenCount(tokens)
})

const sessionTokens = computed(() => {
  const usage = sessionStore.currentSession?.tokenUsage
  return {
    input: usage?.input ?? 0,
    output: usage?.output ?? 0,
    total: (usage?.input ?? 0) + (usage?.output ?? 0),
  }
})

function focusTextarea(): void {
  nextTick(() => {
    textareaRef.value?.focus()
  })
}

onMounted(focusTextarea)

watch(() => uiStore.focusInputCounter, focusTextarea)

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    send()
  }
}

function pushHistory(text: string) {
  if (!text) return
  if (inputHistory.length > 0 && inputHistory[inputHistory.length - 1] === text) return
  inputHistory.push(text)
  historyIndex = inputHistory.length
}

function send() {
  const rawText = inputText.value.trim()
  if (!canSend.value) return

  const text = (pastedFullText.value && rawText === pasteLabel.value)
    ? pastedFullText.value
    : rawText

  pushHistory(text)
  const images = uploadedImages.value.length > 0 ? [...uploadedImages.value] : undefined
  emit('send', text, images)
  inputText.value = ''
  pastedFullText.value = ''
  pasteLabel.value = ''
  uploadedImages.value = []
  autoResize()
}

function handlePaste(e: ClipboardEvent) {
  const items = e.clipboardData?.items
  if (items) {
    for (let i = 0; i < items.length; i++) {
      const item = items[i]
      if (item.type.startsWith('image/')) {
        e.preventDefault()
        const file = item.getAsFile()
        if (file) {
          readImageAsBase64(file)
        }
        return
      }
    }
  }

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

function triggerFileInput(): void {
  fileInputRef.value?.click()
}

function handleFileInputChange(e: Event): void {
  const input = e.target as HTMLInputElement
  const files = input.files
  if (!files) return
  for (let i = 0; i < files.length; i++) {
    const file = files[i]
    if (file.type.startsWith('image/')) {
      readImageAsBase64(file)
    }
  }
  input.value = ''
}

function openLightbox(src: string): void {
  lightboxImage.value = src
}

function closeLightbox(): void {
  lightboxImage.value = null
}

function openCommand() {
  emit('openCommand')
}
</script>

<template>
  <div class="mobile-input-bar">
    <div v-if="uploadedImages.length > 0" class="mobile-image-previews">
      <div v-for="(img, idx) in uploadedImages" :key="idx" class="mobile-image-preview-item">
        <img :src="img" alt="preview" class="mobile-image-preview-thumb" @click="openLightbox(img)" />
        <button class="mobile-image-remove-btn" @click="removeImage(idx)" aria-label="移除">&times;</button>
      </div>
    </div>

    <div class="mobile-input-row">
      <input
        ref="fileInputRef"
        type="file"
        accept="image/*"
        multiple
        class="file-input-hidden"
        @change="handleFileInputChange"
      />

      <button
        class="command-btn"
        :class="{ pressed: false }"
        aria-label="命令面板"
        data-test="mobile-command-btn"
        @click="openCommand"
        @touchstart.prevent
        @touchend.prevent="openCommand"
      >
        <span class="command-btn-text">/</span>
      </button>

      <div class="mobile-input-wrapper">
        <textarea
          ref="textareaRef"
          v-model="inputText"
          class="mobile-input-field"
          placeholder="输入消息..."
          :disabled="isDisabled"
          :maxlength="MAX_MESSAGE_LENGTH"
          rows="1"
          data-test="mobile-input-textarea"
          @keydown="handleKeydown"
          @paste="handlePaste"
        />
        <button
          class="mobile-input-image-btn"
          aria-label="上传图片"
          @click="triggerFileInput"
          @touchstart.prevent
          @touchend.prevent="triggerFileInput"
        >
          <AppIcon name="image" :size="18" />
        </button>
      </div>

      <button
        v-if="isProcessing"
        class="mobile-btn-stop"
        aria-label="停止"
        data-test="mobile-stop-btn"
        @click="emit('stop')"
      >
        <AppIcon name="stop" :size="18" />
      </button>
      <button
        v-else
        class="mobile-btn-send"
        :disabled="!canSend"
        aria-label="发送"
        @click="send"
      >
        <AppIcon name="caret-right" :size="18" color="white" />
      </button>
    </div>

    <div class="mobile-input-footer" data-test="mobile-input-footer">
      <div class="footer-row">
        <span class="footer-item">Context <span class="context-warn">{{ contextUsage }}</span></span>
        <span class="footer-sep">·</span>
        <span class="footer-item">Tokens {{ formatTokenCount(sessionTokens.total) }} (<AppIcon name="arrow-up" :size="10" />{{ formatTokenCount(sessionTokens.input) }} <AppIcon name="arrow-down" :size="10" />{{ formatTokenCount(sessionTokens.output) }})</span>
      </div>
    </div>

    <Teleport to="body">
      <div v-if="lightboxImage" class="mobile-image-lightbox" @click="closeLightbox">
        <img :src="lightboxImage" alt="enlarged" class="mobile-lightbox-image" @click.stop />
        <button class="mobile-lightbox-close" @click="closeLightbox" aria-label="关闭">&times;</button>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.mobile-input-bar {
  flex-shrink: 0;
  padding: var(--space-sm) var(--space-md);
  padding-bottom: calc(var(--space-sm) + env(safe-area-inset-bottom, 0px));
  background: var(--color-bg-primary);
  border-top: 1px solid var(--color-border-light);
}

.mobile-input-row {
  display: flex;
  align-items: flex-end;
  gap: var(--space-sm);
}

.command-btn {
  flex-shrink: 0;
  width: 40px;
  height: 40px;
  border-radius: var(--radius-md);
  background: var(--color-accent);
  color: white;
  border: none;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: transform var(--transition-fast);
  -webkit-tap-highlight-color: transparent;
  touch-action: manipulation;
  min-width: 44px;
  min-height: 44px;
}

.command-btn:active {
  transform: scale(0.95);
}

.command-btn-text {
  font-size: 18px;
  font-weight: 600;
  line-height: 1;
}

.mobile-input-field {
  flex: 1;
  min-height: 40px;
  max-height: 120px;
  padding: 10px var(--space-md);
  padding-left: 36px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  background: var(--color-bg-secondary);
  color: var(--color-text-primary);
  font-family: var(--font-sans);
  font-size: 15px;
  line-height: 1.4;
  resize: none;
  outline: none;
  overflow-y: hidden;
  transition: border-color var(--transition-fast);
  width: 100%;
}

.mobile-input-field:focus {
  border-color: var(--color-accent);
}

.mobile-input-field:disabled {
  opacity: 0.5;
}

.mobile-input-wrapper {
  position: relative;
  flex: 1;
  display: flex;
  align-items: flex-end;
}

.mobile-input-image-btn {
  position: absolute;
  left: 8px;
  bottom: 8px;
  width: 28px;
  height: 28px;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text-tertiary);
  border: none;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  -webkit-tap-highlight-color: transparent;
  touch-action: manipulation;
  transition: color var(--transition-fast), background var(--transition-fast);
  padding: 0;
}

.mobile-input-image-btn:active {
  color: var(--color-accent);
  background: var(--color-bg-hover);
}

.mobile-btn-send,
.mobile-btn-stop {
  flex-shrink: 0;
  width: 40px;
  height: 40px;
  border-radius: var(--radius-md);
  border: none;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all var(--transition-fast);
  min-width: 44px;
  min-height: 44px;
}

.mobile-btn-send {
  background: var(--color-accent);
  color: white;
}

.mobile-btn-send:disabled {
  background: var(--color-bg-tertiary);
  color: var(--color-text-tertiary);
  cursor: not-allowed;
}

.mobile-btn-stop {
  background: var(--color-error);
  color: white;
}

.mobile-input-footer {
  display: flex;
  align-items: center;
  gap: 2px;
  margin-top: var(--space-xs);
  padding: 0 var(--space-xs);
  font-size: 10px;
  font-family: var(--font-mono);
  color: var(--color-text-secondary);
  user-select: none;
}

.footer-row {
  display: flex;
  align-items: center;
  gap: 4px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.footer-item {
  opacity: 0.85;
  flex-shrink: 0;
}

.footer-sep {
  opacity: 0.4;
  flex-shrink: 0;
}

.context-warn {
  color: #ff9500;
}

.file-input-hidden {
  display: none;
}

.mobile-image-previews {
  display: flex;
  gap: var(--space-sm);
  padding: 0 0 var(--space-sm) 0;
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
}

.mobile-image-preview-item {
  position: relative;
  flex-shrink: 0;
  width: 72px;
  height: 72px;
  border-radius: var(--radius-sm);
  overflow: hidden;
  border: 1px solid var(--color-border);
}

.mobile-image-preview-thumb {
  width: 100%;
  height: 100%;
  object-fit: cover;
  cursor: pointer;
}

.mobile-image-remove-btn {
  position: absolute;
  top: 0;
  right: 0;
  width: 22px;
  height: 22px;
  border-radius: 0 0 0 var(--radius-sm);
  background: rgba(0, 0, 0, 0.6);
  color: #fff;
  border: none;
  font-size: 14px;
  line-height: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  padding: 0;
}

.mobile-image-lightbox {
  position: fixed;
  top: 0;
  left: 0;
  width: 100vw;
  height: 100vh;
  background: rgba(0, 0, 0, 0.85);
  z-index: 9999;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
}

.mobile-lightbox-image {
  max-width: 90vw;
  max-height: 90vh;
  object-fit: contain;
  border-radius: var(--radius-md);
  cursor: default;
}

.mobile-lightbox-close {
  position: absolute;
  top: 16px;
  right: 16px;
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.15);
  color: #fff;
  border: none;
  font-size: 24px;
  line-height: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  padding: 0;
}
</style>