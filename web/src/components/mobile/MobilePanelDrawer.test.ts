import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { useUiStore } from '@/stores/ui'
import MobilePanelDrawer from '@/components/mobile/MobilePanelDrawer.vue'

describe('MobilePanelDrawer', () => {
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

  function mountDrawer() {
    return mount(MobilePanelDrawer, {
      attachTo: container,
      global: {
        stubs: {
          Teleport: true,
          AppIcon: true,
          FilesPanel: true,
          SkillsPanel: true,
          McpPanel: true,
          MemoryPanel: true,
          DashboardPanel: true,
          SettingsPanel: true,
        },
      },
    })
  }

  it('should render with tabs', () => {
    const wrapper = mountDrawer()

    const tabs = wrapper.findAll('.drawer-tab-btn')
    expect(tabs.length).toBeGreaterThan(0)
  })

  it('should have files tab', () => {
    const wrapper = mountDrawer()

    const tabs = wrapper.findAll('.drawer-tab-btn')
    const filesTab = tabs.find(t => t.text().trim() === 'Files')
    expect(filesTab).toBeTruthy()
  })

  it('should have skills tab', () => {
    const wrapper = mountDrawer()

    const tabs = wrapper.findAll('.drawer-tab-btn')
    const skillsTab = tabs.find(t => t.text().trim() === 'Skills')
    expect(skillsTab).toBeTruthy()
  })

  it('should switch tab on click', async () => {
    const uiStore = useUiStore()
    uiStore.setActiveRightTab('files')

    const wrapper = mountDrawer()

    const tabs = wrapper.findAll('.drawer-tab-btn')
    const skillsTab = tabs.find(t => t.text().trim() === 'Skills')
    expect(skillsTab).toBeTruthy()
    await skillsTab!.trigger('click')

    expect(uiStore.activeRightTab).toBe('skills')
  })

  it('should show back button', () => {
    const wrapper = mountDrawer()

    const backBtn = wrapper.find('.drawer-back-btn')
    expect(backBtn.exists()).toBe(true)
  })

  it('should emit close when back button clicked', async () => {
    const wrapper = mountDrawer()

    await wrapper.find('.drawer-back-btn').trigger('click')

    expect(wrapper.emitted('close')).toBeTruthy()
  })
})