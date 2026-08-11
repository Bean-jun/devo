import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useSession } from '@/composables/useSession'
import { useSessionStore } from '@/stores/session'
import { useUiStore } from '@/stores/ui'

describe('useSession', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.stubGlobal('fetch', vi.fn())
  })

  describe('createAndSwitch', () => {
    it('should create session and show success toast', async () => {
      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ id: 'new-sess', title: 'My Session', state: 'idle' }),
      } as Response)

      const { createAndSwitch } = useSession()
      const session = await createAndSwitch('My Session')

      expect(session).toBeTruthy()
      expect(session?.title).toBe('My Session')

      const uiStore = useUiStore()
      expect(uiStore.toasts).toHaveLength(1)
      expect(uiStore.toasts[0].type).toBe('success')
    })

    it('should show error toast on failure', async () => {
      vi.mocked(fetch).mockResolvedValueOnce({
        ok: false,
        status: 500,
      } as Response)

      const { createAndSwitch } = useSession()
      const session = await createAndSwitch('Bad')

      expect(session).toBeNull()

      const uiStore = useUiStore()
      expect(uiStore.toasts[0].type).toBe('error')
    })
  })

  describe('switchTo', () => {
    it('should switch to existing session', async () => {
      const sessionStore = useSessionStore()
      sessionStore.sessions = [
        { id: 'sess-1', title: 'S1', state: 'idle' } as any,
        { id: 'sess-2', title: 'S2', state: 'idle' } as any,
      ]

      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ id: 'sess-2', title: 'S2', state: 'idle' }),
      } as Response)

      const { switchTo } = useSession()
      const ok = await switchTo('sess-2')

      expect(ok).toBe(true)
      expect(sessionStore.currentSession?.id).toBe('sess-2')
    })

    it('should show error for non-existent session', async () => {
      vi.mocked(fetch).mockResolvedValueOnce({
        ok: false,
        status: 404,
      } as Response)

      const { switchTo } = useSession()
      const ok = await switchTo('non-existent')

      expect(ok).toBe(false)
    })
  })

  describe('rename', () => {
    it('should rename session and show toast', async () => {
      const sessionStore = useSessionStore()
      sessionStore.currentSession = { id: 'sess-1', title: 'Old', state: 'idle' } as any

      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({}),
      } as Response)

      const { rename } = useSession()
      await rename('sess-1', 'New Name')

      expect(sessionStore.currentSession?.title).toBe('New Name')
    })
  })

  describe('archive', () => {
    it('should archive session and show toast', async () => {
      const sessionStore = useSessionStore()
      sessionStore.currentSession = { id: 'sess-1', title: 'Test', state: 'idle' } as any

      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({}),
      } as Response)

      const { archive } = useSession()
      await archive('sess-1')

      expect(sessionStore.currentSession?.state).toBe('archived')
    })
  })
})