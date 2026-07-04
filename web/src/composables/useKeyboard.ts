import { onMounted, onUnmounted } from 'vue'

type KeyHandler = (e: KeyboardEvent) => void

interface Shortcut {
  key: string
  ctrl?: boolean
  shift?: boolean
  alt?: boolean
  handler: KeyHandler
}

export function useKeyboard(shortcuts: Shortcut[]) {
  const handler = (e: KeyboardEvent) => {
    const tag = (e.target as HTMLElement)?.tagName?.toLowerCase()
    const isInput = tag === 'input' || tag === 'textarea' || (e.target as HTMLElement)?.isContentEditable

    for (const shortcut of shortcuts) {
      const ctrlMatch = shortcut.ctrl ? (e.ctrlKey || e.metaKey) : !e.ctrlKey && !e.metaKey
      const shiftMatch = shortcut.shift ? e.shiftKey : !e.shiftKey
      const altMatch = shortcut.alt ? e.altKey : !e.altKey
      if (e.key.toLowerCase() === shortcut.key.toLowerCase() && ctrlMatch && shiftMatch && altMatch) {
        if (!isInput) {
          e.preventDefault()
        }
        shortcut.handler(e)
        return
      }
    }
  }

  onMounted(() => {
    window.addEventListener('keydown', handler)
  })

  onUnmounted(() => {
    window.removeEventListener('keydown', handler)
  })
}