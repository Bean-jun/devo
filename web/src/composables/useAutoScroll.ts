import { ref } from 'vue'
import { AUTO_SCROLL_THRESHOLD } from '@/utils/constants'

export function useAutoScroll() {
  const containerRef = ref<HTMLElement | null>(null)
  const isUserScrolledUp = ref(false)
  const showScrollToBottom = ref(false)

  function isNearBottom(): boolean {
    const el = containerRef.value
    if (!el) return true
    return el.scrollHeight - el.scrollTop - el.clientHeight < AUTO_SCROLL_THRESHOLD
  }

  function scrollToBottom(smooth = true): void {
    const el = containerRef.value
    if (!el) return
    el.scrollTo({
      top: el.scrollHeight,
      behavior: smooth ? 'smooth' : 'auto',
    })
    isUserScrolledUp.value = false
    showScrollToBottom.value = false
  }

  function scrollToMessage(messageId: string): void {
    const el = containerRef.value
    if (!el) return
    const targetEl = el.querySelector(`[data-message-id="${messageId}"]`) as HTMLElement | null
    if (!targetEl) return

    const containerRect = el.getBoundingClientRect()
    const targetRect = targetEl.getBoundingClientRect()
    const offset = targetRect.top - containerRect.top + el.scrollTop - 80

    el.scrollTo({
      top: offset,
      behavior: 'smooth',
    })
  }

  function onScroll(): void {
    const nearBottom = isNearBottom()
    isUserScrolledUp.value = !nearBottom
    showScrollToBottom.value = !nearBottom
  }

  return {
    containerRef,
    showScrollToBottom,
    scrollToBottom,
    scrollToMessage,
    onScroll,
    isNearBottom,
  }
}