import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useSessionStore } from '@/stores/session'
import { mockSession, mockSessions } from '@/test/fixtures/sessions'

describe('SessionStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.stubGlobal('fetch', vi.fn())
  })

  describe('createSession', () => {
    it('should create a session and set it as current', async () => {
      const store = useSessionStore()
      const mockResponse = {
        id: 'sess-001',
        title: 'My Project',
        state: 'idle',
        created_at: '2026-01-01T00:00:00Z',
      }

      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockResponse),
      } as Response)

      await store.createSession({ title: 'My Project' })

      expect(store.currentSession).toBeTruthy()
      expect(store.currentSession?.id).toBe('sess-001')
      expect(store.sessions).toHaveLength(1)
    })

    it('should handle API error on create', async () => {
      const store = useSessionStore()

      vi.mocked(fetch).mockResolvedValueOnce({
        ok: false,
        status: 500,
        statusText: 'Internal Server Error',
      } as Response)

      await expect(store.createSession({ title: 'Test' })).rejects.toThrow()
      expect(store.currentSession).toBeNull()
    })
  })

  describe('fetchSessions', () => {
    it('should fetch all sessions', async () => {
      const store = useSessionStore()

      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ sessions: mockSessions }),
      } as Response)

      await store.fetchSessions()
      expect(store.sessions).toHaveLength(3)
    })

    it('should handle empty sessions array', async () => {
      const store = useSessionStore()

      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ sessions: [] }),
      } as Response)

      await store.fetchSessions()
      expect(store.sessions).toHaveLength(0)
    })
  })

  describe('switchSession', () => {
    it('should update currentSession when switching', () => {
      const store = useSessionStore()
      store.sessions = [...mockSessions]
      store.currentSession = store.sessions[0]

      store.switchSession(store.sessions[1])

      expect(store.currentSession?.id).toBe('sess-002')
    })
  })

  describe('switchSessionById', () => {
    it('should switch to existing session in list', async () => {
      const store = useSessionStore()
      store.sessions = [...mockSessions]

      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockSessions[1]),
      } as Response)

      const ok = await store.switchSessionById('sess-002')

      expect(ok).toBe(true)
      expect(store.currentSession?.id).toBe('sess-002')
    })

    it('should return false for non-existent session', async () => {
      const store = useSessionStore()

      vi.mocked(fetch).mockResolvedValueOnce({
        ok: false,
        status: 404,
      } as Response)

      const ok = await store.switchSessionById('sess-999')
      expect(ok).toBe(false)
    })
  })

  describe('renameSession', () => {
    it('should rename current session', async () => {
      const store = useSessionStore()
      store.currentSession = { ...mockSession }

      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({}),
      } as Response)

      await store.renameSession('sess-001', 'New Name')
      expect(store.currentSession?.title).toBe('New Name')
    })
  })

  describe('archiveSession', () => {
    it('should archive current session', async () => {
      const store = useSessionStore()
      store.currentSession = { ...mockSession }

      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({}),
      } as Response)

      await store.archiveSession('sess-001')
      expect(store.currentSession?.state).toBe('archived')
    })
  })

  describe('computed', () => {
    it('should return correct session status', () => {
      const store = useSessionStore()
      store.currentSession = { ...mockSession, state: 'processing' }

      expect(store.isProcessing).toBe(true)
      expect(store.isAwaitingApproval).toBe(false)
      expect(store.sessionStatus).toBe('processing')
    })

    it('should detect archived session as inactive', () => {
      const store = useSessionStore()
      store.currentSession = { ...mockSession, state: 'archived' }

      expect(store.isArchived).toBe(true)
      expect(store.isSessionActive).toBe(false)
    })
  })
})