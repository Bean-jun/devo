import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { useCommandStore } from '@/stores/command'
import CommandPalette from '@/components/command/CommandPalette.vue'

describe('CommandPalette', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('should not render when closed', () => {
    const wrapper = mount(CommandPalette)

    expect(wrapper.find('[data-test="command-palette"]').exists()).toBe(false)
  })

  it('should render command list when open', async () => {
    const commandStore = useCommandStore()
    commandStore.open([
      { id: 'new', name: '/new', description: '创建新会话', action: () => {} },
      { id: 'help', name: '/help', description: '显示帮助', action: () => {} },
    ])

    const wrapper = mount(CommandPalette)

    const items = wrapper.findAll('[data-test="command-item"]')
    expect(items).toHaveLength(2)
  })

  it('should filter by query', async () => {
    const commandStore = useCommandStore()
    commandStore.open([
      { id: 'new', name: '/new', description: '创建新会话', action: () => {} },
      { id: 'help', name: '/help', description: '显示帮助', action: () => {} },
    ])
    commandStore.setQuery('new')

    const wrapper = mount(CommandPalette)

    const items = wrapper.findAll('[data-test="command-item"]')
    expect(items).toHaveLength(1)
    expect(items[0].text()).toContain('/new')
  })

  it('should show empty state when no match', () => {
    const commandStore = useCommandStore()
    commandStore.open([
      { id: 'new', name: '/new', description: '创建新会话', action: () => {} },
    ])
    commandStore.setQuery('zzz')

    const wrapper = mount(CommandPalette)

    expect(wrapper.find('[data-test="command-item"]').exists()).toBe(false)
  })

  it('should highlight selected item', () => {
    const commandStore = useCommandStore()
    commandStore.open([
      { id: 'new', name: '/new', description: '创建', action: () => {} },
      { id: 'help', name: '/help', description: '帮助', action: () => {} },
    ])
    commandStore.selectedIndex = 1

    const wrapper = mount(CommandPalette)

    const items = wrapper.findAll('[data-test="command-item"]')
    expect(items[1].classes()).toContain('selected')
  })
})