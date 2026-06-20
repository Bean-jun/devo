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

  it('should show streaming preview when content exists', () => {
    const chatStore = useChatStore()
    chatStore.startStreaming()
    chatStore.appendStreamChunk('Streaming text preview...')

    const wrapper = mount(ThinkingIndicator)

    expect(wrapper.find('.streaming-preview').exists()).toBe(true)
  })

  it('should not show streaming preview when no content', () => {
    const wrapper = mount(ThinkingIndicator)

    expect(wrapper.find('.streaming-preview').exists()).toBe(false)
  })
})