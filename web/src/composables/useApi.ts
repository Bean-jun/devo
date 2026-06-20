import { API_BASE } from '@/utils/constants'

interface RequestOptions {
  method?: string
  body?: unknown
  signal?: AbortSignal
  timeout?: number
}

export function useApi() {
  async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
    const { method = 'GET', body, signal, timeout = 30000 } = options

    const controller = new AbortController()
    const timeoutId = setTimeout(() => controller.abort(), timeout)

    const mergedSignal = signal
      ? AbortSignal.any?.([signal, controller.signal]) ?? controller.signal
      : controller.signal

    try {
      const res = await fetch(`${API_BASE}${path}`, {
        method,
        headers: body ? { 'Content-Type': 'application/json' } : undefined,
        body: body ? JSON.stringify(body) : undefined,
        signal: mergedSignal,
      })

      if (!res.ok) {
        const errorData = await res.json().catch(() => ({}))
        throw new ApiError(
          errorData.error || `请求失败 (${res.status})`,
          res.status,
          errorData
        )
      }

      const text = await res.text()
      if (!text) return {} as T
      return JSON.parse(text) as T
    } catch (err) {
      if (err instanceof ApiError) throw err
      if ((err as Error).name === 'AbortError') {
        throw new ApiError('请求超时', 408)
      }
      throw new ApiError((err as Error).message || '网络错误', 0)
    } finally {
      clearTimeout(timeoutId)
    }
  }

  function get<T>(path: string, signal?: AbortSignal): Promise<T> {
    return request<T>(path, { signal })
  }

  function post<T>(path: string, body?: unknown, signal?: AbortSignal): Promise<T> {
    return request<T>(path, { method: 'POST', body, signal })
  }

  function put<T>(path: string, body?: unknown, signal?: AbortSignal): Promise<T> {
    return request<T>(path, { method: 'PUT', body, signal })
  }

  function del<T>(path: string, signal?: AbortSignal): Promise<T> {
    return request<T>(path, { method: 'DELETE', signal })
  }

  return { get, post, put, del, request }
}

export class ApiError extends Error {
  status: number
  data: unknown

  constructor(message: string, status: number, data?: unknown) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.data = data
  }
}