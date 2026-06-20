import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useApprovalStore } from '@/stores/approval'
import type { ApprovalRequest } from '@/types/approval'

const mockApproval: ApprovalRequest = {
  id: 'approval-001',
  sessionId: 'sess-001',
  toolCallId: 'tc-001',
  toolName: 'delete_file',
  command: 'rm -rf important.txt',
  filePath: '/tmp/important.txt',
  riskLevel: 'high',
  diff: '- old content\n+ new content',
  parameters: { path: '/tmp/important.txt' },
  timeout: 30000,
  createdAt: '2026-01-01T00:00:00Z',
}

describe('ApprovalStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.stubGlobal('fetch', vi.fn())
  })

  describe('setApproval', () => {
    it('should set current approval', () => {
      const store = useApprovalStore()

      store.setApproval(mockApproval)

      expect(store.currentApproval).toEqual(mockApproval)
      expect(store.hasPendingApproval).toBe(true)
    })
  })

  describe('clearApproval', () => {
    it('should clear approval and add to history', () => {
      const store = useApprovalStore()

      store.setApproval(mockApproval)
      store.clearApproval()

      expect(store.currentApproval).toBeNull()
      expect(store.hasPendingApproval).toBe(false)
      expect(store.approvalHistory).toHaveLength(1)
    })
  })

  describe('approve', () => {
    it('should approve and clear current approval', async () => {
      const store = useApprovalStore()

      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({}),
      } as Response)

      store.setApproval(mockApproval)
      await store.approve(false)

      expect(store.currentApproval).toBeNull()
      expect(store.approvalHistory).toHaveLength(1)
    })

    it('should throw on approve failure', async () => {
      const store = useApprovalStore()

      vi.mocked(fetch).mockResolvedValueOnce({
        ok: false,
        status: 500,
      } as Response)

      store.setApproval(mockApproval)
      await expect(store.approve()).rejects.toThrow('批准失败')
    })

    it('should do nothing if no current approval', async () => {
      const store = useApprovalStore()
      await store.approve()
      expect(store.currentApproval).toBeNull()
    })
  })

  describe('reject', () => {
    it('should reject and clear current approval', async () => {
      const store = useApprovalStore()

      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({}),
      } as Response)

      store.setApproval(mockApproval)
      await store.reject()

      expect(store.currentApproval).toBeNull()
    })

    it('should throw on reject failure', async () => {
      const store = useApprovalStore()

      vi.mocked(fetch).mockResolvedValueOnce({
        ok: false,
        status: 500,
      } as Response)

      store.setApproval(mockApproval)
      await expect(store.reject()).rejects.toThrow('拒绝失败')
    })
  })

  describe('computed', () => {
    it('should return false when no pending approval', () => {
      const store = useApprovalStore()
      expect(store.hasPendingApproval).toBe(false)
    })

    it('should return approval timeout', () => {
      const store = useApprovalStore()
      store.setApproval(mockApproval)
      expect(store.approvalTimeout).toBe(30000)
    })
  })
})