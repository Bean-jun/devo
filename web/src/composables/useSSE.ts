import { ref, onUnmounted } from 'vue'
import { API_BASE } from '@/utils/constants'
import { SSE_RECONNECT_BASE_MS, SSE_RECONNECT_MAX_MS } from '@/utils/constants'
import type { SSEEventType } from '@/types/sse'

type EventHandler = (data: unknown) => void

export function useSSE() {
  const eventSource = ref<EventSource | null>(null)
  const isConnected = ref(false)
  const reconnectAttempt = ref(0)
  const reconnectTimer = ref<ReturnType<typeof setTimeout> | null>(null)
  const handlers = new Map<string, Set<EventHandler>>()
  let statusHandler: ((connected: boolean) => void) | null = null

  function onStatusChange(fn: (connected: boolean) => void): void {
    statusHandler = fn
  }

  function connect(sessionId: string): void {
    if (eventSource.value) {
      disconnect()
    }

    const url = `${API_BASE}/sessions/${sessionId}/events`
    const es = new EventSource(url)
    eventSource.value = es

    es.onopen = () => {
      isConnected.value = true
      reconnectAttempt.value = 0
      statusHandler?.(true)
    }

    es.onmessage = (event: MessageEvent) => {
      try {
        const raw = JSON.parse(event.data)
        const eventType = raw.type || 'message'
        const eventData = raw.data || raw

        const typeHandlers = handlers.get(eventType)
        if (typeHandlers) {
          typeHandlers.forEach(handler => handler(eventData))
        }

        const allHandlers = handlers.get('*')
        if (allHandlers) {
          allHandlers.forEach(handler => handler({ type: eventType, data: eventData }))
        }
      } catch {
        return
      }
    }

    es.onerror = () => {
      isConnected.value = false
      statusHandler?.(false)
      es.close()
      eventSource.value = null

      if (reconnectAttempt.value < 5) {
        const delay = Math.min(
          SSE_RECONNECT_BASE_MS * Math.pow(2, reconnectAttempt.value),
          SSE_RECONNECT_MAX_MS
        )
        reconnectAttempt.value++
        reconnectTimer.value = setTimeout(() => {
          connect(sessionId)
        }, delay)
      }
    }
  }

  function disconnect(): void {
    if (reconnectTimer.value) {
      clearTimeout(reconnectTimer.value)
      reconnectTimer.value = null
    }
    reconnectAttempt.value = 5
    if (eventSource.value) {
      eventSource.value.close()
      eventSource.value = null
    }
    isConnected.value = false
    handlers.clear()
  }

  function onEvent(eventType: SSEEventType | '*', handler: EventHandler): void {
    if (!handlers.has(eventType)) {
      handlers.set(eventType, new Set())
    }
    handlers.get(eventType)!.add(handler)
  }

  function offEvent(eventType: string, handler: EventHandler): void {
    const typeHandlers = handlers.get(eventType)
    if (typeHandlers) {
      typeHandlers.delete(handler)
    }
  }

  onUnmounted(() => {
    disconnect()
    handlers.clear()
  })

  return {
    isConnected,
    connect,
    disconnect,
    onEvent,
    offEvent,
    onStatusChange,
  }
}