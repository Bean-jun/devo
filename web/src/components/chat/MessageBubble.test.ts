import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import MessageBubble from '@/components/chat/MessageBubble.vue'
import { mockUserMessage, mockAssistantMessage, mockSystemMessage } from '@/test/fixtures/messages'

describe('MessageBubble', () => {
  it('should render user message with correct alignment', () => {
    const wrapper = mount(MessageBubble, {
      props: { message: mockUserMessage },
    })

    expect(wrapper.find('[data-test="message-bubble"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('Hello, can you help me')
  })

  it('should render assistant message', () => {
    const wrapper = mount(MessageBubble, {
      props: { message: mockAssistantMessage },
    })

    expect(wrapper.text()).toContain('Sure! Here is a function')
  })

  it('should render system message with correct style', () => {
    const wrapper = mount(MessageBubble, {
      props: { message: mockSystemMessage },
    })

    expect(wrapper.text()).toContain('Session created')
  })

  it('should render user role label', () => {
    const wrapper = mount(MessageBubble, {
      props: { message: mockUserMessage },
    })

    expect(wrapper.text()).toContain('你')
  })

  it('should render assistant role label', () => {
    const wrapper = mount(MessageBubble, {
      props: { message: mockAssistantMessage },
    })

    expect(wrapper.text()).toContain('Devo')
  })
})