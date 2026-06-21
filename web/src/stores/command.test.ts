import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useCommandStore } from '@/stores/command'
import type { Command } from '@/stores/command'

const mockCommands: Command[] = [
  { id: 'new', name: '/new', description: '创建新会话', action: () => {} },
  { id: 'switch', name: '/switch', description: '切换会话', action: () => {} },
  { id: 'export', name: '/export', description: '下载当前会话记录', action: () => {} },
  { id: 'help', name: '/help', description: '显示帮助', action: () => {} },
]

describe('CommandStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  describe('open', () => {
    it('should open command palette with commands', () => {
      const store = useCommandStore()

      store.open(mockCommands)

      expect(store.isOpen).toBe(true)
      expect(store.commands).toHaveLength(4)
      expect(store.query).toBe('')
      expect(store.selectedIndex).toBe(0)
    })

    it('should set onSelect callback', () => {
      const store = useCommandStore()
      const callback = () => {}

      store.open(mockCommands, callback)

      expect(store.isOpen).toBe(true)
    })
  })

  describe('close', () => {
    it('should close command palette', () => {
      const store = useCommandStore()

      store.open(mockCommands)
      store.close()

      expect(store.isOpen).toBe(false)
      expect(store.query).toBe('')
      expect(store.selectedIndex).toBe(0)
    })
  })

  describe('setQuery', () => {
    it('should set query and reset selectedIndex', () => {
      const store = useCommandStore()

      store.open(mockCommands)
      store.setQuery('/new')

      expect(store.query).toBe('/new')
      expect(store.selectedIndex).toBe(0)
    })
  })

  describe('filteredCommands', () => {
    it('should return all commands when query is empty', () => {
      const store = useCommandStore()

      store.open(mockCommands)

      expect(store.filteredCommands).toHaveLength(4)
    })

    it('should filter commands by name', () => {
      const store = useCommandStore()

      store.open(mockCommands)
      store.setQuery('new')

      expect(store.filteredCommands).toHaveLength(1)
      expect(store.filteredCommands[0].id).toBe('new')
    })

    it('should filter commands by description', () => {
      const store = useCommandStore()

      store.open(mockCommands)
      store.setQuery('帮助')

      expect(store.filteredCommands).toHaveLength(1)
      expect(store.filteredCommands[0].id).toBe('help')
    })

    it('should return empty when no match', () => {
      const store = useCommandStore()

      store.open(mockCommands)
      store.setQuery('zzz')

      expect(store.filteredCommands).toHaveLength(0)
    })
  })

  describe('moveUp/moveDown', () => {
    it('should move selection up', () => {
      const store = useCommandStore()

      store.open(mockCommands)
      store.selectedIndex = 2
      store.moveUp()

      expect(store.selectedIndex).toBe(1)
    })

    it('should not move above 0', () => {
      const store = useCommandStore()

      store.open(mockCommands)
      store.moveUp()

      expect(store.selectedIndex).toBe(0)
    })

    it('should move selection down', () => {
      const store = useCommandStore()

      store.open(mockCommands)
      store.moveDown()

      expect(store.selectedIndex).toBe(1)
    })

    it('should not move beyond last item', () => {
      const store = useCommandStore()

      store.open(mockCommands)
      store.selectedIndex = 3
      store.moveDown()

      expect(store.selectedIndex).toBe(3)
    })
  })

  describe('select', () => {
    it('should call action and close', () => {
      const store = useCommandStore()
      let called = false
      const cmd: Command = { id: 'test', name: '/test', description: 'test', action: () => { called = true } }

      store.open([cmd])
      store.select()

      expect(called).toBe(true)
      expect(store.isOpen).toBe(false)
    })

    it('should call onSelect callback', () => {
      const store = useCommandStore()
      let selectedCmd: Command | null = null
      const cmd: Command = { id: 'test', name: '/test', description: 'test', action: () => {} }

      store.open([cmd], (c: Command) => { selectedCmd = c })
      store.select()

      expect(selectedCmd!.id).toBe('test')
    })

    it('should do nothing when no commands match', () => {
      const store = useCommandStore()

      store.open(mockCommands)
      store.setQuery('zzz')
      store.select()

      expect(store.isOpen).toBe(true)
    })
  })

  describe('selectedCommand', () => {
    it('should return selected command', () => {
      const store = useCommandStore()

      store.open(mockCommands)
      store.selectedIndex = 1

      expect(store.selectedCommand?.id).toBe('switch')
    })

    it('should return null when no commands', () => {
      const store = useCommandStore()

      store.open([])

      expect(store.selectedCommand).toBeNull()
    })
  })
})