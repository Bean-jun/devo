import type { Session } from '@/types/session'

export const mockSession: Session = {
  id: 'sess-001',
  title: 'Test Project',
  state: 'idle',
  workingDirectory: '/tmp/test',
  createdAt: '2026-01-01T00:00:00Z',
  lastActiveAt: '2026-01-01T12:00:00Z',
  messageCount: 5,
  tokenUsage: { input: 500, output: 200 },
  trustLevel: 'normal',
  approvalPolicy: {},
}

export const mockSessions: Session[] = [
  mockSession,
  {
    id: 'sess-002',
    title: 'Another Project',
    state: 'archived',
    workingDirectory: '/tmp/another',
    createdAt: '2026-01-02T00:00:00Z',
    lastActiveAt: '2026-01-03T00:00:00Z',
    messageCount: 20,
    tokenUsage: { input: 3000, output: 1500 },
    trustLevel: 'elevated',
    approvalPolicy: {},
  },
  {
    id: 'sess-003',
    title: 'Processing Project',
    state: 'processing',
    workingDirectory: '/tmp/processing',
    createdAt: '2026-01-04T00:00:00Z',
    lastActiveAt: '2026-01-04T12:00:00Z',
    messageCount: 8,
    tokenUsage: { input: 800, output: 400 },
    trustLevel: 'low',
    approvalPolicy: { exec_python: 'always_ask' },
  },
]