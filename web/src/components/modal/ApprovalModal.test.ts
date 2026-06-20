import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { useApprovalStore } from '@/stores/approval'
import { useUiStore } from '@/stores/ui'
import ApprovalModal from '@/components/modal/ApprovalModal.vue'

describe('ApprovalModal', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.stubGlobal('fetch', vi.fn())
  })

  it('should not render when no approval', () => {
    const wrapper = mount(ApprovalModal)

    expect(wrapper.find('[data-test="approval-modal"]').exists()).toBe(false)
  })

  it('should render risk level badge', () => {
    const approvalStore = useApprovalStore()
    const uiStore = useUiStore()

    approvalStore.setApproval({
      id: 'appr-1',
      sessionId: 'sess-1',
      toolCallId: 'tc-1',
      toolName: 'delete_file',
      riskLevel: 'high',
      parameters: { path: '/tmp/test.txt' },
      timeout: 30000,
      createdAt: '2026-01-01T00:00:00Z',
    })

    uiStore.setActiveModal('approval')

    const wrapper = mount(ApprovalModal)

    const riskBadge = wrapper.find('[data-test="risk-level"]')
    expect(riskBadge.exists()).toBe(true)
  })

  it('should call approve on button click', async () => {
    const approvalStore = useApprovalStore()
    const uiStore = useUiStore()

    vi.mocked(fetch).mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({}),
    } as Response)

    approvalStore.setApproval({
      id: 'appr-1',
      sessionId: 'sess-1',
      toolCallId: 'tc-1',
      toolName: 'write_file',
      riskLevel: 'medium',
      parameters: { path: '/tmp/test.txt', content: 'hello' },
      timeout: 30000,
      createdAt: '2026-01-01T00:00:00Z',
    })

    uiStore.setActiveModal('approval')

    const wrapper = mount(ApprovalModal)

    await wrapper.find('[data-test="approve-button"]').trigger('click')

    expect(approvalStore.currentApproval).toBeNull()
  })

  it('should call reject on button click', async () => {
    const approvalStore = useApprovalStore()
    const uiStore = useUiStore()

    vi.mocked(fetch).mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({}),
    } as Response)

    approvalStore.setApproval({
      id: 'appr-1',
      sessionId: 'sess-1',
      toolCallId: 'tc-1',
      toolName: 'write_file',
      riskLevel: 'medium',
      parameters: { path: '/tmp/test.txt', content: 'hello' },
      timeout: 30000,
      createdAt: '2026-01-01T00:00:00Z',
    })

    uiStore.setActiveModal('approval')

    const wrapper = mount(ApprovalModal)

    await wrapper.find('[data-test="reject-button"]').trigger('click')

    expect(approvalStore.currentApproval).toBeNull()
  })
})