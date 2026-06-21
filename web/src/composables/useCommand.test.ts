import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useCommand } from '@/composables/useCommand'
import { useCommandStore } from '@/stores/command'

describe('useCommand', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('should open palette with builtin commands', () => {
    const { openPalette } = useCommand()
    const commandStore = useCommandStore()

    openPalette()

    expect(commandStore.isOpen).toBe(true)
    expect(commandStore.commands.length).toBeGreaterThan(0)
  })

  it('should include /new command', () => {
    const { openPalette } = useCommand()
    const commandStore = useCommandStore()

    openPalette()

    const newCmd = commandStore.commands.find(c => c.id === 'new')
    expect(newCmd).toBeTruthy()
    expect(newCmd?.name).toBe('/new')
  })

  it('should include /switch command', () => {
    const { openPalette } = useCommand()
    const commandStore = useCommandStore()

    openPalette()

    const cmd = commandStore.commands.find(c => c.id === 'switch')
    expect(cmd).toBeTruthy()
  })

  it('should include /help command', () => {
    const { openPalette } = useCommand()
    const commandStore = useCommandStore()

    openPalette()

    const cmd = commandStore.commands.find(c => c.id === 'help')
    expect(cmd).toBeTruthy()
  })

  it('should include /export command', () => {
    const { openPalette } = useCommand()
    const commandStore = useCommandStore()

    openPalette()

    const cmd = commandStore.commands.find(c => c.id === 'export')
    expect(cmd).toBeTruthy()
  })
})