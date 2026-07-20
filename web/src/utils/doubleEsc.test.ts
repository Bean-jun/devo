import { describe, it, expect } from 'vitest'
import { decideDoubleEsc } from './doubleEsc'

describe('decideDoubleEsc', () => {
  const base = {
    state: 'idle' as string | undefined,
    now: 1000,
    lastEscAt: 0,
    windowMs: 500,
  }

  it('arms on first ESC in idle state', () => {
    expect(decideDoubleEsc({ ...base, state: 'idle' })).toBe('arm')
  })

  it('arms on first ESC in cancelled state', () => {
    expect(decideDoubleEsc({ ...base, state: 'cancelled' })).toBe('arm')
  })

  it('triggers on second ESC within window in idle state', () => {
    expect(decideDoubleEsc({ ...base, state: 'idle', now: 1000, lastEscAt: 600, windowMs: 500 })).toBe('trigger')
  })

  it('triggers on second ESC within window in cancelled state', () => {
    expect(decideDoubleEsc({ ...base, state: 'cancelled', now: 1000, lastEscAt: 600, windowMs: 500 })).toBe('trigger')
  })

  it('re-arms when second ESC falls outside window', () => {
    expect(decideDoubleEsc({ ...base, state: 'idle', now: 2000, lastEscAt: 1000, windowMs: 500 })).toBe('arm')
  })

  it('does not trigger at exact window boundary (strict less-than)', () => {
    expect(decideDoubleEsc({ ...base, state: 'idle', now: 1500, lastEscAt: 1000, windowMs: 500 })).toBe('arm')
  })

  it('returns noop for thinking state', () => {
    expect(decideDoubleEsc({ ...base, state: 'thinking' })).toBe('noop')
  })

  it('returns noop for tool_executing state', () => {
    expect(decideDoubleEsc({ ...base, state: 'tool_executing' })).toBe('noop')
  })

  it('returns noop for paused state', () => {
    expect(decideDoubleEsc({ ...base, state: 'paused' })).toBe('noop')
  })

  it('returns noop for awaiting_approval state', () => {
    expect(decideDoubleEsc({ ...base, state: 'awaiting_approval' })).toBe('noop')
  })

  it('returns noop for archived state', () => {
    expect(decideDoubleEsc({ ...base, state: 'archived' })).toBe('noop')
  })

  it('returns noop for undefined state', () => {
    expect(decideDoubleEsc({ ...base, state: undefined })).toBe('noop')
  })
})
