import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import ToolCallCard from '@/components/chat/ToolCallCard.vue'
import { mockToolCallPending, mockToolCallSuccess, mockToolCallFailed } from '@/test/fixtures/tools'

describe('ToolCallCard', () => {
  it('should render tool name', () => {
    const wrapper = mount(ToolCallCard, {
      props: { toolCall: mockToolCallPending },
    })

    expect(wrapper.text()).toContain('write_file')
  })

  it('should show success status', () => {
    const wrapper = mount(ToolCallCard, {
      props: { toolCall: mockToolCallSuccess },
    })

    expect(wrapper.text()).toContain('write_file')
  })

  it('should show failed status', () => {
    const wrapper = mount(ToolCallCard, {
      props: { toolCall: mockToolCallFailed },
    })

    expect(wrapper.text()).toContain('write_file')
  })

  it('should show parameters when expanded', async () => {
    const wrapper = mount(ToolCallCard, {
      props: { toolCall: mockToolCallPending },
    })

    await wrapper.find('.tool-header').trigger('click')

    expect(wrapper.find('.tool-params').exists()).toBe(true)
  })
})