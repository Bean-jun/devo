import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import ThinkingIndicator from '@/components/chat/ThinkingIndicator.vue'
import { useChatStore } from '@/stores/chat'

describe('ThinkingIndicator', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('should render thinking indicator', () => {
    const wrapper = mount(ThinkingIndicator)

    expect(wrapper.find('[data-test="thinking-indicator"]').exists()).toBe(true)
  })

  it('should show streaming text when content exists', () => {
    const chatStore = useChatStore()
    chatStore.startStreaming()
    chatStore.appendStreamChunk('Streaming text preview...')

    const wrapper = mount(ThinkingIndicator)

    expect(wrapper.find('.streaming-text').exists()).toBe(true)
    expect(wrapper.find('.streaming-text').text()).toContain('Streaming text preview...')
  })

  it('should show empty hint when no content', () => {
    const wrapper = mount(ThinkingIndicator)

    expect(wrapper.find('.streaming-text').exists()).toBe(false)
    expect(wrapper.find('.empty-hint').exists()).toBe(true)
  })

  it('should show reasoning section when reasoning is streaming', () => {
    const chatStore = useChatStore()
    chatStore.startStreaming()
    chatStore.startReasoning()
    chatStore.appendReasoningChunk('正在思考...')

    const wrapper = mount(ThinkingIndicator)

    expect(wrapper.find('[data-test="reasoning-section"]').exists()).toBe(true)
    expect(wrapper.find('.reasoning-title').text()).toContain('正在思考')
  })

  it('should hide reasoning content by default (collapsed)', () => {
    const chatStore = useChatStore()
    chatStore.startStreaming()
    chatStore.startReasoning()
    chatStore.appendReasoningChunk('隐藏的思考过程')

    const wrapper = mount(ThinkingIndicator)

    expect(wrapper.find('[data-test="reasoning-content"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="reasoning-content"]').isVisible()).toBe(false)
  })

  it('should expand reasoning content on toggle click', async () => {
    const chatStore = useChatStore()
    chatStore.startStreaming()
    chatStore.startReasoning()
    chatStore.appendReasoningChunk('展开后能看到我')

    const wrapper = mount(ThinkingIndicator)

    const toggleIcon = wrapper.findComponent('.toggle-icon') as any
    expect(toggleIcon.props('name')).toBe('caret-right')
    await wrapper.find('[data-test="reasoning-toggle"]').trigger('click')
    expect(toggleIcon.props('name')).toBe('caret-down')
    expect(wrapper.find('.reasoning-text').text()).toContain('展开后能看到我')
  })

  it('should show thinking dots in reasoning header while active', () => {
    const chatStore = useChatStore()
    chatStore.startStreaming()
    chatStore.startReasoning()
    chatStore.appendReasoningChunk('思考中...')

    const wrapper = mount(ThinkingIndicator)

    const header = wrapper.find('[data-test="reasoning-toggle"]')
    expect(header.find('.thinking-dots').exists()).toBe(true)
  })

  it('should not show thinking dots in reasoning header after reasoning finishes', () => {
    const chatStore = useChatStore()
    chatStore.startStreaming()
    chatStore.startReasoning()
    chatStore.appendReasoningChunk('done thinking')
    chatStore.finishReasoning()

    const wrapper = mount(ThinkingIndicator)

    const header = wrapper.find('[data-test="reasoning-toggle"]')
    expect(header.find('.thinking-dots').exists()).toBe(false)
    expect(header.text()).toContain('思考过程')
  })

  it('should hide reasoning section when there is no reasoning', () => {
    const chatStore = useChatStore()
    chatStore.startStreaming()
    chatStore.appendStreamChunk('只有正文')

    const wrapper = mount(ThinkingIndicator)

    expect(wrapper.find('[data-test="reasoning-section"]').exists()).toBe(false)
  })

  it('should show both reasoning section and content when both exist', () => {
    const chatStore = useChatStore()
    chatStore.startStreaming()
    chatStore.startReasoning()
    chatStore.appendReasoningChunk('思考完毕')
    chatStore.finishReasoning()
    chatStore.appendStreamChunk('最终答案')

    const wrapper = mount(ThinkingIndicator)

    expect(wrapper.find('[data-test="reasoning-section"]').exists()).toBe(true)
    expect(wrapper.find('.streaming-text').exists()).toBe(true)
    expect(wrapper.find('.streaming-text').text()).toContain('最终答案')
  })
})