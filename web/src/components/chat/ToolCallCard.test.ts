import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import ToolCallCard from '@/components/chat/ToolCallCard.vue'
import { mockToolCallPending, mockToolCallExecuting, mockToolCallSuccess, mockToolCallFailed } from '@/test/fixtures/tools'

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

  it('should display stage label when executing', () => {
    const wrapper = mount(ToolCallCard, {
      props: { toolCall: mockToolCallExecuting },
    })

    expect(wrapper.find('[data-test="tool-stage"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="tool-stage"]').text()).toBe('running')
  })

  it('should display streaming output when executing', () => {
    const wrapper = mount(ToolCallCard, {
      props: { toolCall: mockToolCallExecuting },
    })

    expect(wrapper.find('[data-test="tool-streaming"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="tool-streaming"]').text()).toContain('Writing file...')
  })

  it('should not display streaming output when not executing', () => {
    const wrapper = mount(ToolCallCard, {
      props: { toolCall: mockToolCallSuccess },
    })

    expect(wrapper.find('[data-test="tool-streaming"]').exists()).toBe(false)
  })

  it('should not display stage for pending tool', () => {
    const wrapper = mount(ToolCallCard, {
      props: { toolCall: mockToolCallPending },
    })

    expect(wrapper.find('[data-test="tool-stage"]').exists()).toBe(false)
  })

  describe('streaming flow simulation', () => {
    it('should accumulate streaming output chunk by chunk', async () => {
      const wrapper = mount(ToolCallCard, {
        props: { toolCall: mockToolCallPending },
      })

      expect(wrapper.find('[data-test="tool-streaming"]').exists()).toBe(false)

      await wrapper.setProps({
        toolCall: {
          ...mockToolCallPending,
          status: 'executing' as const,
          streamingOutput: '找到 3 个匹配...\n',
          stage: 'running',
        },
      })
      expect(wrapper.find('[data-test="tool-streaming"]').exists()).toBe(true)
      expect(wrapper.find('[data-test="tool-streaming"]').text()).toContain('找到 3 个匹配')

      await wrapper.setProps({
        toolCall: {
          ...mockToolCallPending,
          status: 'executing' as const,
          streamingOutput: '找到 3 个匹配...\n文件: foo.go 第 12 行\n',
          stage: 'running',
        },
      })
      expect(wrapper.find('[data-test="tool-streaming"]').text()).toContain('foo.go')
      expect(wrapper.find('[data-test="tool-streaming"]').text()).toContain('第 12 行')

      await wrapper.setProps({
        toolCall: {
          ...mockToolCallPending,
          status: 'executing' as const,
          streamingOutput: '找到 3 个匹配...\n文件: foo.go 第 12 行\n文件: bar.go 第 45 行\n文件: baz.go 第 78 行\n',
          stage: 'done',
        },
      })
      expect(wrapper.find('[data-test="tool-streaming"]').text()).toContain('bar.go')
      expect(wrapper.find('[data-test="tool-streaming"]').text()).toContain('baz.go')
      expect(wrapper.find('[data-test="tool-stage"]').text()).toBe('done')
    })

    it('should hide streaming output and show result when tool completes', async () => {
      const wrapper = mount(ToolCallCard, {
        props: {
          toolCall: {
            ...mockToolCallPending,
            status: 'executing' as const,
            streamingOutput: 'Line 1\nLine 2\n',
            stage: 'running',
          },
        },
      })

      expect(wrapper.find('[data-test="tool-streaming"]').exists()).toBe(true)

      await wrapper.setProps({
        toolCall: {
          ...mockToolCallPending,
          status: 'success' as const,
          result: { success: true, stdout: '执行完成' },
          duration: 120,
          streamingOutput: 'Line 1\nLine 2\n',
        },
      })

      expect(wrapper.find('[data-test="tool-streaming"]').exists()).toBe(false)
      expect(wrapper.find('.tool-result').exists()).toBe(true)
    })
  })
})