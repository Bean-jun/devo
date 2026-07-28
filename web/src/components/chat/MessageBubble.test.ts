import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import MessageBubble from '@/components/chat/MessageBubble.vue'
import { mockUserMessage, mockAssistantMessage, mockSystemMessage, mockAssistantMessageWithReasoning } from '@/test/fixtures/messages'

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

  it('should render reasoning collapse for assistant message with reasoning', () => {
    const wrapper = mount(MessageBubble, {
      props: { message: mockAssistantMessageWithReasoning },
    })

    const collapse = wrapper.find('[data-test="reasoning-collapse"]')
    expect(collapse.exists()).toBe(true)
    expect(wrapper.text()).toContain('思考过程')
    expect(wrapper.text()).toContain('用户想要一个函数')
  })

  it('should collapse reasoning by default', () => {
    const wrapper = mount(MessageBubble, {
      props: { message: mockAssistantMessageWithReasoning },
    })

    const collapse = wrapper.find('[data-test="reasoning-collapse"]')
    expect(collapse.element.hasAttribute('open')).toBe(false)
  })

  it('should expand reasoning when clicked', async () => {
    const wrapper = mount(MessageBubble, {
      props: { message: mockAssistantMessageWithReasoning },
    })

    const summary = wrapper.find('.reasoning-summary')
    expect(wrapper.find('[data-test="reasoning-collapse"]').element.hasAttribute('open')).toBe(false)
    await summary.trigger('click')
    expect(wrapper.find('[data-test="reasoning-collapse"]').element.hasAttribute('open')).toBe(true)
  })

  it('should not render reasoning collapse for assistant message without reasoning', () => {
    const wrapper = mount(MessageBubble, {
      props: { message: mockAssistantMessage },
    })

    expect(wrapper.find('[data-test="reasoning-collapse"]').exists()).toBe(false)
  })

  it('should not render reasoning collapse for user message', () => {
    const wrapper = mount(MessageBubble, {
      props: { message: { ...mockUserMessage, reasoning: 'some reasoning' } },
    })

    expect(wrapper.find('[data-test="reasoning-collapse"]').exists()).toBe(false)
  })
})