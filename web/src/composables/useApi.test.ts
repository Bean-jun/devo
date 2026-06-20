import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { useApi, ApiError } from '@/composables/useApi'

describe('useApi', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    vi.stubGlobal('fetch', vi.fn())
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('request', () => {
    it('should make GET request and return data', async () => {
      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        text: () => Promise.resolve('{"data": "hello"}'),
        json: () => Promise.resolve({ data: 'hello' }),
      } as Response)

      const api = useApi()
      const result = await api.request('/sessions')

      expect(result).toEqual({ data: 'hello' })
    })

    it('should make POST request with body', async () => {
      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        text: () => Promise.resolve('{"id": "new"}'),
        json: () => Promise.resolve({ id: 'new' }),
      } as Response)

      const api = useApi()
      const result = await api.post('/sessions', { title: 'Test' })

      expect(result).toEqual({ id: 'new' })
    })

    it('should handle HTTP error', async () => {
      vi.mocked(fetch).mockResolvedValueOnce({
        ok: false,
        status: 404,
        json: () => Promise.resolve({ error: 'Not found' }),
      } as Response)

      const api = useApi()
      await expect(api.get('/sessions/999')).rejects.toThrow(ApiError)
    })

    it('should handle network error', async () => {
      vi.mocked(fetch).mockRejectedValueOnce(new Error('Network error'))

      const api = useApi()
      await expect(api.get('/sessions')).rejects.toThrow(ApiError)
    })

    it('should handle timeout', async () => {
      vi.mocked(fetch).mockImplementationOnce(() => {
        return new Promise((_, reject) => {
          const err = new Error('Aborted')
          err.name = 'AbortError'
          reject(err)
        })
      })

      const api = useApi()
      await expect(api.request('/sessions', { timeout: 1 })).rejects.toThrow(ApiError)
    })
  })

  describe('ApiError', () => {
    it('should create ApiError with status', () => {
      const err = new ApiError('Test error', 500, { detail: 'test' })

      expect(err.message).toBe('Test error')
      expect(err.status).toBe(500)
      expect(err.data).toEqual({ detail: 'test' })
      expect(err.name).toBe('ApiError')
    })
  })
})