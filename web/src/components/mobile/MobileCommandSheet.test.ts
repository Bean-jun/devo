import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import MobileCommandSheet from '@/components/mobile/MobileCommandSheet.vue'

describe('MobileCommandSheet', () => {
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

  function mountSheet() {
    return mount(MobileCommandSheet, {
      attachTo: container,
      global: {
        stubs: {
          Teleport: true,
          AppIcon: true,
        },
      },
    })
  }

  it('should render search input', () => {
    const wrapper = mountSheet()

    expect(wrapper.find('.search-input').exists()).toBe(true)
  })

  it('should render command groups', () => {
    const wrapper = mountSheet()

    const groups = wrapper.findAll('.command-group')
    expect(groups.length).toBeGreaterThan(0)
  })

  it('should render command items', () => {
    const wrapper = mountSheet()

    const items = wrapper.findAll('.command-item')
    expect(items.length).toBeGreaterThan(0)
  })

  it('should filter commands by search query', async () => {
    const wrapper = mountSheet()

    const searchInput = wrapper.find('.search-input')
    await searchInput.setValue('new')

    const items = wrapper.findAll('.command-item')
    expect(items.length).toBe(1)
    expect(items[0].find('.command-name').text()).toBe('/new')
  })

  it('should show no results when no match', async () => {
    const wrapper = mountSheet()

    const searchInput = wrapper.find('.search-input')
    await searchInput.setValue('xyznonexistent')

    expect(wrapper.find('.sheet-empty').exists()).toBe(true)
  })

  it('should emit select when command clicked', async () => {
    const wrapper = mountSheet()

    const firstItem = wrapper.find('.command-item')
    await firstItem.trigger('click')

    expect(wrapper.emitted('select')).toBeTruthy()
  })

  it('should emit select with correct command data', async () => {
    const wrapper = mountSheet()

    const firstItem = wrapper.find('.command-item')
    await firstItem.trigger('click')

    const emitted = wrapper.emitted('select')
    expect(emitted).toBeTruthy()
    expect(emitted![0]).toBeDefined()
    expect(emitted![0][0]).toHaveProperty('id')
    expect(emitted![0][0]).toHaveProperty('name')
  })
})