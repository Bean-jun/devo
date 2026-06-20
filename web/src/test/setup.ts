import { vi } from 'vitest'

class MockEventSource {
  static instances: MockEventSource[] = []
  onopen: (() => void) | null = null
  onmessage: ((event: MessageEvent) => void) | null = null
  onerror: (() => void) | null = null
  url: string
  readyState: number = 0

  constructor(url: string) {
    this.url = url
    MockEventSource.instances.push(this)
  }

  close() {
    this.readyState = 2
  }

  dispatchEvent(type: string, data: string) {
    if (type === 'open' && this.onopen) this.onopen()
    if (type === 'message' && this.onmessage) {
      this.onmessage(new MessageEvent('message', { data }))
    }
    if (type === 'error' && this.onerror) this.onerror()
  }

  static resetAll() {
    MockEventSource.instances = []
  }
}

globalThis.EventSource = MockEventSource as any

globalThis.fetch = vi.fn()

Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: vi.fn().mockImplementation((query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  })),
})

globalThis.ResizeObserver = vi.fn().mockImplementation(() => ({
  observe: vi.fn(),
  unobserve: vi.fn(),
  disconnect: vi.fn(),
}))

globalThis.IntersectionObserver = vi.fn().mockImplementation(() => ({
  observe: vi.fn(),
  unobserve: vi.fn(),
  disconnect: vi.fn(),
  root: null,
  rootMargin: '',
  thresholds: [],
}))

Element.prototype.scrollTo = vi.fn() as any
Element.prototype.scrollIntoView = vi.fn() as any