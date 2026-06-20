import { vi } from 'vitest'

export interface MockApiOptions {
  shouldFail?: boolean
  errorStatus?: number
  delay?: number
}

export function createMockFetch(overrides: Partial<MockApiOptions> = {}) {
  const options: MockApiOptions = {
    shouldFail: false,
    errorStatus: 500,
    delay: 0,
    ...overrides,
  }

  return vi.fn().mockImplementation(async (_url: string, _init?: RequestInit) => {
    if (options.delay) {
      await new Promise(r => setTimeout(r, options.delay))
    }

    if (options.shouldFail) {
      return {
        ok: false,
        status: options.errorStatus,
        statusText: 'Mock Error',
        json: () => Promise.resolve({ error: 'Mock error' }),
      } as Response
    }

    return createMockResponse({})
  })
}

function createMockResponse(data: unknown): Response {
  return {
    ok: true,
    status: 200,
    json: () => Promise.resolve(data),
    text: () => Promise.resolve(JSON.stringify(data)),
  } as Response
}