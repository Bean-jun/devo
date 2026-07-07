import { ref, computed, onMounted, onUnmounted, watch, type Ref } from 'vue'

interface VirtualScrollOptions {
  /** 每个 item 的预估高度 (px) */
  estimateHeight: number
  /** 可视区域外的缓冲区 item 数量 */
  bufferSize: number
}

interface VirtualScrollReturn {
  /** 滚动容器 ref */
  containerRef: Ref<HTMLElement | null>
  /** 当前可视 + 缓冲区的 item 索引范围 */
  visibleRange: Ref<{ start: number; end: number }>
  /** 总内容高度 */
  totalHeight: Ref<number>
  /** 偏移量（用于 padding-top） */
  offsetY: Ref<number>
  /** 滚动到指定索引 */
  scrollToIndex: (index: number, smooth?: boolean) => void
  /** 当 item 渲染后调用，记录实际高度 */
  updateItemHeight: (index: number, height: number) => void
}

export function useVirtualScroll(
  itemCount: Ref<number>,
  options: VirtualScrollOptions = { estimateHeight: 200, bufferSize: 5 }
): VirtualScrollReturn {
  const containerRef = ref<HTMLElement | null>(null)
  const visibleRange = ref({ start: 0, end: 20 })
  const offsetY = ref(0)
  const totalHeight = ref(0)
  const scrollTop = ref(0)

  const { estimateHeight, bufferSize } = options

  const heights = new Map<number, number>()
  const heightAccumulator: number[] = []

  let isProgrammaticScroll = false
  let programmaticScrollTimer: ReturnType<typeof setTimeout> | null = null

  function recalculateHeight(): void {
    const count = itemCount.value
    if (count === 0) {
      totalHeight.value = 0
      offsetY.value = 0
      visibleRange.value = { start: 0, end: 0 }
      return
    }

    heightAccumulator.length = 0
    let acc = 0
    for (let i = 0; i < count; i++) {
      acc += heights.get(i) ?? estimateHeight
      heightAccumulator.push(acc)
    }
    totalHeight.value = acc
  }

  function findStartIndex(scrollTopVal: number): number {
    let lo = 0
    let hi = heightAccumulator.length - 1
    while (lo < hi) {
      const mid = Math.floor((lo + hi) / 2)
      if (heightAccumulator[mid] <= scrollTopVal) {
        lo = mid + 1
      } else {
        hi = mid
      }
    }
    return Math.max(0, lo - bufferSize)
  }

  function findEndIndex(scrollTopVal: number, containerHeight: number): number {
    const bottom = scrollTopVal + containerHeight
    let lo = 0
    let hi = heightAccumulator.length - 1
    while (lo < hi) {
      const mid = Math.floor((lo + hi) / 2)
      if (heightAccumulator[mid] < bottom) {
        lo = mid + 1
      } else {
        hi = mid
      }
    }
    return Math.min(heightAccumulator.length - 1, lo + bufferSize)
  }

  function updateVisibleRange(): void {
    const el = containerRef.value
    if (!el || heightAccumulator.length === 0) return

    const st = el.scrollTop
    const ch = el.clientHeight
    scrollTop.value = st

    const start = findStartIndex(st)
    const end = findEndIndex(st, ch)

    visibleRange.value = { start, end }

    if (start > 0) {
      offsetY.value = heightAccumulator[start - 1]
    } else {
      offsetY.value = 0
    }
  }

  let scrollHandler: (() => void) | null = null
  let resizeObserver: ResizeObserver | null = null

  onMounted(() => {
    recalculateHeight()
    updateVisibleRange()

    scrollHandler = () => {
      if (isProgrammaticScroll) return
      updateVisibleRange()
    }

    const el = containerRef.value
    if (el) {
      el.addEventListener('scroll', scrollHandler, { passive: true })
      resizeObserver = new ResizeObserver(() => {
        if (isProgrammaticScroll) return
        recalculateHeight()
        updateVisibleRange()
      })
      resizeObserver.observe(el)
    }
  })

  onUnmounted(() => {
    const el = containerRef.value
    if (el && scrollHandler) {
      el.removeEventListener('scroll', scrollHandler)
    }
    if (resizeObserver) {
      resizeObserver.disconnect()
    }
  })

  watch(
    itemCount,
    (newCount, oldCount) => {
      if (newCount < oldCount) {
        for (const key of heights.keys()) {
          if (key >= newCount) {
            heights.delete(key)
          }
        }
      }
      recalculateHeight()
      updateVisibleRange()
    }
  )

  function isAtBottom(): boolean {
    const el = containerRef.value
    if (!el) return false
    return el.scrollHeight - el.scrollTop - el.clientHeight < 50
  }

  function updateItemHeight(index: number, height: number): void {
    const old = heights.get(index)
    if (old === height) return

    const wasAtBottom = isAtBottom()

    heights.set(index, height)
    recalculateHeight()

    if (wasAtBottom) {
      const el = containerRef.value
      if (el) {
        isProgrammaticScroll = true
        if (programmaticScrollTimer) {
          clearTimeout(programmaticScrollTimer)
        }
        el.scrollTop = el.scrollHeight
        programmaticScrollTimer = setTimeout(() => {
          isProgrammaticScroll = false
          programmaticScrollTimer = null
          updateVisibleRange()
        }, 50)
      }
    } else {
      updateVisibleRange()
    }
  }

  function scrollToIndex(index: number, smooth = true): void {
    const el = containerRef.value
    if (!el) return

    recalculateHeight()

    if (index < 0) index = 0
    if (index >= heightAccumulator.length) index = heightAccumulator.length - 1

    const targetTop = index > 0 ? heightAccumulator[index - 1] : 0

    isProgrammaticScroll = true
    if (programmaticScrollTimer) {
      clearTimeout(programmaticScrollTimer)
    }

    el.scrollTo({ top: targetTop, behavior: smooth ? 'smooth' : 'auto' })

    const resetDelay = smooth ? 400 : 50
    programmaticScrollTimer = setTimeout(() => {
      isProgrammaticScroll = false
      programmaticScrollTimer = null
      updateVisibleRange()
    }, resetDelay)
  }

  return {
    containerRef,
    visibleRange,
    totalHeight,
    offsetY,
    scrollToIndex,
    updateItemHeight,
  }
}