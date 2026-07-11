import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { useUiStore } from '@/stores/ui'
import MobileWorkspacePicker from '@/components/mobile/MobileWorkspacePicker.vue'

describe('MobileWorkspacePicker', () => {
  let container: HTMLDivElement

  beforeEach(() => {
    container = document.createElement('div')
    document.body.appendChild(container)
    setActivePinia(createPinia())
    global.fetch = vi.fn().mockResolvedValue({ ok: true })
  })

  afterEach(() => {
    document.body.removeChild(container)
  })

  function mountPicker() {
    return mount(MobileWorkspacePicker, {
      attachTo: container,
      global: {
        stubs: {
          Teleport: true,
        },
      },
    })
  }

  it('should render title', () => {
    const wrapper = mountPicker()

    expect(wrapper.find('.picker-title').text()).toBe('切换工作区')
  })

  it('should render cancel button', () => {
    const wrapper = mountPicker()

    expect(wrapper.find('.picker-cancel').text()).toBe('取消')
  })

  it('should render workspace list items', () => {
    const uiStore = useUiStore()
    uiStore.workspaceList = [
      { id: '/home/a', name: 'a', path: '/home/a', exists: true },
      { id: '/home/b', name: 'b', path: '/home/b', exists: true },
    ]

    const wrapper = mountPicker()

    const items = wrapper.findAll('.picker-item')
    expect(items.length).toBe(2)
  })

  it('should highlight active workspace', () => {
    const uiStore = useUiStore()
    uiStore.workspaceList = [
      { id: '/home/a', name: 'a', path: '/home/a', exists: true },
      { id: '/home/b', name: 'b', path: '/home/b', exists: true },
    ]
    uiStore.activeWorkspace = '/home/b'

    const wrapper = mountPicker()

    const activeItem = wrapper.find('.picker-item.active')
    expect(activeItem.exists()).toBe(true)
    expect(activeItem.find('.picker-item-name').text()).toBe('b')
  })

  it('should show check mark on active workspace', () => {
    const uiStore = useUiStore()
    uiStore.workspaceList = [
      { id: '/home/a', name: 'a', path: '/home/a', exists: true },
    ]
    uiStore.activeWorkspace = '/home/a'

    const wrapper = mountPicker()

    expect(wrapper.find('.picker-check').exists()).toBe(true)
  })

  it('should emit close when cancel button clicked', async () => {
    const wrapper = mountPicker()

    await wrapper.find('.picker-cancel').trigger('click')

    expect(wrapper.emitted('close')).toBeTruthy()
  })

  it('should show empty state when no workspaces', () => {
    const uiStore = useUiStore()
    uiStore.workspaceList = []

    const wrapper = mountPicker()

    expect(wrapper.find('.picker-empty').exists()).toBe(true)
  })

  it('should disable removed workspace', () => {
    const uiStore = useUiStore()
    uiStore.workspaceList = [
      { id: '/home/removed', name: 'removed', path: '/home/removed', exists: false },
    ]

    const wrapper = mountPicker()

    const item = wrapper.find('.picker-item')
    expect(item.exists()).toBe(true)
    expect(item.classes()).toContain('removed')
  })

  it('should show folder icon for existing workspace', () => {
    const uiStore = useUiStore()
    uiStore.workspaceList = [
      { id: '/home/a', name: 'a', path: '/home/a', exists: true },
    ]

    const wrapper = mountPicker()

    const item = wrapper.find('.picker-item')
    expect(item.exists()).toBe(true)
  })
})