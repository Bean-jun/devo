import { ref, onMounted, onUnmounted, watch } from 'vue'

export interface VirtualMessageItemProps {
  index: number
  onHeightChange: (index: number, height: number) => void
}

export function useVirtualMessageItem(props: VirtualMessageItemProps) {
  const rootRef = ref<HTMLElement | null>(null)
  let observer: ResizeObserver | null = null

onMounted(() => {
  if (!rootRef.value) return
  observer = new ResizeObserver((entries) => {
    for (const entry of entries) {
      const h = Math.round(entry.contentRect.height)
      if (h > 0) {
        props.onHeightChange(props.index, h)
      }
    }
  })
  observer.observe(rootRef.value)
})

onUnmounted(() => {
  if (observer) {
    observer.disconnect()
    observer = null
  }
})

  return { rootRef }
}