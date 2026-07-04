import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useSSE } from '@/composables/useSSE'

describe('useSSE', () => {
  beforeEach(() => {
    (globalThis.EventSource as any).resetAll()
  })

  it('should create an EventSource connection', () => {
    const { connect } = useSSE()

    connect('sess-001')

    const instances = (globalThis.EventSource as any).instances
    expect(instances).toHaveLength(1)
    expect(instances[0].url).toContain('sess-001')
  })

  it('should call handler for specific event type', () => {
    const { connect, onEvent } = useSSE()
    const handler = vi.fn()

    onEvent('streaming_token', handler)
    connect('sess-001')

    const instance = (globalThis.EventSource as any).instances[0]
    instance.dispatchEvent('message', JSON.stringify({ type: 'streaming_token', data: { content: 'Hello' } }))

    expect(handler).toHaveBeenCalledWith({ content: 'Hello' })
  })

  it('should call wildcard handler for all events', () => {
    const { connect, onEvent } = useSSE()
    const handler = vi.fn()

    onEvent('*', handler)
    connect('sess-001')

    const instance = (globalThis.EventSource as any).instances[0]
    instance.dispatchEvent('message', JSON.stringify({ type: 'streaming_token', data: { content: 'Hello' } }))

    expect(handler).toHaveBeenCalledWith({ type: 'streaming_token', data: { content: 'Hello' } })
  })

  it('should set isConnected to true on open', () => {
    const { connect, isConnected } = useSSE()

    connect('sess-001')

    const instance = (globalThis.EventSource as any).instances[0]
    instance.dispatchEvent('open', '')

    expect(isConnected.value).toBe(true)
  })

  it('should disconnect and not reconnect', () => {
    vi.useFakeTimers()
    const { connect, disconnect } = useSSE()

    connect('sess-001')
    disconnect()

    const instance = (globalThis.EventSource as any).instances[0]
    instance.dispatchEvent('error', '')

    vi.advanceTimersByTime(10000)
    expect((globalThis.EventSource as any).instances).toHaveLength(1)

    vi.useRealTimers()
  })

  it('should preserve handlers after reconnect', () => {
    vi.useFakeTimers()
    const { connect, onEvent } = useSSE()
    const handler = vi.fn()

    onEvent('streaming_token', handler)
    connect('sess-001')

    const instance = (globalThis.EventSource as any).instances[0]
    instance.dispatchEvent('error', '')

    vi.advanceTimersByTime(2000)

    const reconnectedInstance = (globalThis.EventSource as any).instances[1]
    reconnectedInstance.dispatchEvent('message', JSON.stringify({ type: 'streaming_token', data: { content: 'Hello after reconnect' } }))

    expect(handler).toHaveBeenCalledWith({ content: 'Hello after reconnect' })

    vi.useRealTimers()
  })

  it('should handle malformed JSON gracefully', () => {
    const { connect, onEvent } = useSSE()
    const handler = vi.fn()

    onEvent('*', handler)
    connect('sess-001')

    const instance = (globalThis.EventSource as any).instances[0]
    instance.dispatchEvent('message', 'not json')

    expect(handler).not.toHaveBeenCalled()
  })

  it('should remove event handler', () => {
    const { connect, onEvent, offEvent } = useSSE()
    const handler = vi.fn()

    onEvent('streaming_token', handler)
    offEvent('streaming_token', handler)
    connect('sess-001')

    const instance = (globalThis.EventSource as any).instances[0]
    instance.dispatchEvent('message', JSON.stringify({ type: 'streaming_token', data: { content: 'Hello' } }))

    expect(handler).not.toHaveBeenCalled()
  })
})