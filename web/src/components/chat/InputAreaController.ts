import { ref, computed, watch, onMounted, nextTick } from 'vue'
import { useCommandStore } from '@/stores/command'
import { useSessionStore } from '@/stores/session'
import { useUiStore } from '@/stores/ui'
import { useModelStore } from '@/stores/model'
import { MAX_MESSAGE_LENGTH } from '@/utils/constants'
import { estimateTokens, formatTokenCount } from '@/utils/formatters'
import {
  useInputSegments,
  createPasteSegment,
  parseDOMToSegments,
  chipLabel,
  shouldFoldPaste,
  type PasteSegment,
} from '@/composables/useInputSegments'

export interface InputAreaProps {
  isDisabled: boolean
  isProcessing: boolean
}

export interface InputAreaEmits {
  send: [text: string, images?: string[]]
  stop: []
  clear: []
  openCommand: []
  executeCommand: [text: string]
}

export function useInputArea(props: InputAreaProps, emit: (e: string, ...args: any[]) => void) {
  const editorRef = ref<HTMLDivElement | null>(null)
  const commandStore = useCommandStore()
  const sessionStore = useSessionStore()
  const uiStore = useUiStore()
  const modelStore = useModelStore()

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

const uploadedImages = ref<string[]>([])

const fileInputRef = ref<HTMLInputElement | null>(null)
const lightboxImage = ref<string | null>(null)

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

const charCount = computed(() => totalLength())
const tokenEstimate = computed(() => estimateTokens(serialize()))
const canSend = computed(() => (!isEmpty() || uploadedImages.value.length > 0) && !props.isDisabled)
const showPlaceholder = computed(() => isEmpty())

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

const workingDir = computed(() => sessionStore.currentSession?.workingDirectory ?? '')

const activeModelName = computed(() => {
  const m = modelStore.models.find(m => m.id === modelStore.activeModelId)
  return m?.name ?? ''
})

function focusEditor(): void {
  nextTick(() => {
    editorRef.value?.focus()
  })
}

onMounted(() => {
  renderEditor()
  focusEditor()
  modelStore.fetchModels()
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
    const images = uploadedImages.value.length > 0 ? [...uploadedImages.value] : undefined
    emit('send', text, images)
  }
  reset()
  pasteMap.value.clear()
  uploadedImages.value = []
  renderEditor()
}

function autoResize(): void {
  const el = editorRef.value
  if (!el) return
  el.style.height = 'auto'
  el.style.height = Math.min(el.scrollHeight, 200) + 'px'
}

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

function handleImageDrop(e: DragEvent): void {
  e.preventDefault()
  const files = e.dataTransfer?.files
  if (!files) return
  for (let i = 0; i < files.length; i++) {
    const file = files[i]
    if (file.type.startsWith('image/')) {
      readImageAsBase64(file)
    }
  }
}

function handleDragOver(e: DragEvent): void {
  e.preventDefault()
}

  return {
    editorRef,
    commandStore,
    sessionStore,
    uiStore,
    segments,
    pasteMap,
    isComposing,
    uploadedImages,
    fileInputRef,
    lightboxImage,
    triggerFileInput,
    handleFileInputChange,
    openLightbox,
    closeLightbox,
    charCount,
    tokenEstimate,
    canSend,
    showPlaceholder,
    contextUsage,
    sessionTokens,
    workingDir,
    activeModelName,
    focusEditor,
    handleInput,
    handlePaste,
    handleKeydown,
    handleCompositionStart,
    handleCompositionEnd,
    handleImageDrop,
    handleDragOver,
    removeImage,
    send,
    reset,
    autoResize,
    isEmpty,
    MAX_MESSAGE_LENGTH,
    formatTokenCount,
  }
}