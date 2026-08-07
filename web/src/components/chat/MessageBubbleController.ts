import { computed, ref } from 'vue'
import type { Message } from '@/types/message'
import { formatTime } from '@/utils/formatters'
import { renderMarkdown } from '@/utils/markdown'

export interface MessageBubbleProps {
  message: Message
}

export function useMessageBubble(props: MessageBubbleProps) {
  const copied = ref(false)

const renderedContent = computed(() => {
  if (props.message.role === 'assistant') {
    return renderMarkdown(props.message.content)
  }
  return props.message.content
})

const displayTime = computed(() => formatTime(props.message.timestamp))

async function copyContent() {
  try {
    await navigator.clipboard.writeText(props.message.content)
    copied.value = true
    setTimeout(() => {
      copied.value = false
    }, 2000)
  } catch {
    const textarea = document.createElement('textarea')
    textarea.value = props.message.content
    textarea.style.position = 'fixed'
    textarea.style.opacity = '0'
    document.body.appendChild(textarea)
    textarea.select()
    document.execCommand('copy')
    document.body.removeChild(textarea)
    copied.value = true
    setTimeout(() => {
      copied.value = false
    }, 2000)
  }
}

const contentRef = ref<HTMLElement | null>(null)
const lightboxImage = ref<string | null>(null)

function openLightbox(src: string): void {
  lightboxImage.value = src
}

function closeLightbox(): void {
  lightboxImage.value = null
}

async function handleCodeBlockCopy(e: MouseEvent): Promise<void> {
  const target = e.target as HTMLElement
  const btn = target.closest('.code-block-copy') as HTMLButtonElement | null
  if (!btn) return
  const text = btn.getAttribute('data-code') || ''
  try {
    await navigator.clipboard.writeText(text)
  } catch {
    const textarea = document.createElement('textarea')
    textarea.value = text
    textarea.style.position = 'fixed'
    textarea.style.opacity = '0'
    document.body.appendChild(textarea)
    textarea.select()
    document.execCommand('copy')
    document.body.removeChild(textarea)
  }
  const orig = btn.textContent
  btn.textContent = '已复制'
  btn.classList.add('copied')
  setTimeout(() => {
    btn.textContent = orig
    btn.classList.remove('copied')
  }, 2000)
}

  return {
    copied,
    renderedContent,
    displayTime,
    copyContent,
    contentRef,
    lightboxImage,
    openLightbox,
    closeLightbox,
    handleCodeBlockCopy,
  }
}