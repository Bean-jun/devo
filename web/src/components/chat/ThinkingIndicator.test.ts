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
})