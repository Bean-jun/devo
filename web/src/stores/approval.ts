import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { ApprovalRequest } from '@/types/approval'
import { API_BASE } from '@/utils/constants'

export const useApprovalStore = defineStore('approval', () => {
  const currentApproval = ref<ApprovalRequest | null>(null)
  const approvalHistory = ref<ApprovalRequest[]>([])

  const hasPendingApproval = computed(() => currentApproval.value !== null)
  const approvalTimeout = computed(() => currentApproval.value?.timeout ?? 0)

  function setApproval(approval: ApprovalRequest): void {
    currentApproval.value = approval
  }

  function clearApproval(): void {
    if (currentApproval.value) {
      approvalHistory.value.push(currentApproval.value)
    }
    currentApproval.value = null
  }

  async function approve(_trustAllInSession?: boolean): Promise<void> {
    if (!currentApproval.value) return
    const approvalId = currentApproval.value.id
    const sessionId = currentApproval.value.sessionId

    const res = await fetch(`${API_BASE}/sessions/${sessionId}/approve/${approvalId}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ decision: 'approve' }),
    })
    if (!res.ok) throw new Error('批准失败')
    clearApproval()
  }

  async function reject(): Promise<void> {
    if (!currentApproval.value) return
    const approvalId = currentApproval.value.id
    const sessionId = currentApproval.value.sessionId

    const res = await fetch(`${API_BASE}/sessions/${sessionId}/approve/${approvalId}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ decision: 'reject' }),
    })
    if (!res.ok) throw new Error('拒绝失败')
    clearApproval()
  }

  return {
    currentApproval,
    approvalHistory,
    hasPendingApproval,
    approvalTimeout,
    setApproval,
    clearApproval,
    approve,
    reject,
  }
})