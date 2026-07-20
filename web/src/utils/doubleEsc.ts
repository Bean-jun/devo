export type DoubleEscDecision = 'trigger' | 'arm' | 'noop'

export interface DecideDoubleEscParams {
  state: string | undefined
  now: number
  lastEscAt: number
  windowMs: number
}

export function decideDoubleEsc(params: DecideDoubleEscParams): DoubleEscDecision {
  const { state, now, lastEscAt, windowMs } = params
  if (state !== 'idle' && state !== 'cancelled') return 'noop'
  if (now - lastEscAt < windowMs) return 'trigger'
  return 'arm'
}
