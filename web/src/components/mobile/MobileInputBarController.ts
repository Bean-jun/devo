import { ref, computed, watch, onMounted, nextTick } from 'vue'
import { useSessionStore } from '@/stores/session'
import { useUiStore } from '@/stores/ui'
import { useModelStore } from '@/stores/model'
import { MAX_MESSAGE_LENGTH } from '@/utils/constants'
import { estimateTokens, formatTokenCount } from '@/utils/formatters'

export interface MobileInputBarProps {
  isDisabled: boolean
  isProcessing: boolean
}

export interface MobileInputBarEmits {
  send: [text: string, images?: string[]]
  stop: []
  openCommand: []
}

export function useMobileInputBar(props: MobileInputBarProps, emit: (e: string, ...args: any[]) => void) {
  const inputText = ref('')
  const textareaRef = ref<HTMLTextAreaElement | null>(null)
  const fileInputRef = ref<HTMLInputElement | null>(null)
  const sessionStore = useSessionStore()
  const uiStore = useUiStore()
  const modelStore = useModelStore()

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

const activeModelName = computed(() => {
  const m = modelStore.models.find(m => m.id === modelStore.activeModelId)
  return m?.name ?? ''
})

function focusTextarea(): void {
  nextTick(() => {
    textareaRef.value?.focus()
  })
}

onMounted(() => {
  focusTextarea()
  modelStore.fetchModels()
})

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

  return {
    inputText,
    textareaRef,
    fileInputRef,
    sessionStore,
    uiStore,
    pastedFullText,
    pasteLabel,
    uploadedImages,
    lightboxImage,
    charCount,
    tokenEstimate,
    canSend,
    contextUsage,
    sessionTokens,
    activeModelName,
    focusTextarea,
    handleKeydown,
    send,
    handlePaste,
    autoResize,
    triggerFileInput,
    handleFileInputChange,
    openLightbox,
    closeLightbox,
    openCommand,
    readImageAsBase64,
    removeImage,
    MAX_MESSAGE_LENGTH,
    formatTokenCount,
  }
}