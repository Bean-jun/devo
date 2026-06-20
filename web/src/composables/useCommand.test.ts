import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useCommand } from '@/composables/useCommand'
import { useCommandStore } from '@/stores/command'
import { useUiStore } from '@/stores/ui'

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

  it('should include /sessions command', () => {
    const { openPalette } = useCommand()
    const commandStore = useCommandStore()

    openPalette()

    const cmd = commandStore.commands.find(c => c.id === 'sessions')
    expect(cmd).toBeTruthy()
  })

  it('should include /help command', () => {
    const { openPalette } = useCommand()
    const commandStore = useCommandStore()

    openPalette()

    const cmd = commandStore.commands.find(c => c.id === 'help')
    expect(cmd).toBeTruthy()
  })

  it('should include /clear command', () => {
    const { openPalette } = useCommand()
    const commandStore = useCommandStore()

    openPalette()

    const cmd = commandStore.commands.find(c => c.id === 'clear')
    expect(cmd).toBeTruthy()
  })
})