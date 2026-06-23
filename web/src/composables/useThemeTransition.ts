export function useThemeTransition() {
  function startTransition(
    x: number,
    y: number,
    updateTheme: () => void,
  ): Promise<void> {
    document.documentElement.style.setProperty('--theme-x', `${x}px`)
    document.documentElement.style.setProperty('--theme-y', `${y}px`)

    if (document.startViewTransition) {
      const transition = document.startViewTransition(() => {
        updateTheme()
      })
      return transition.finished.then(() => {
        cleanup()
      })
    }

    updateTheme()
    cleanup()
    return Promise.resolve()
  }

  function cleanup() {
    document.documentElement.style.removeProperty('--theme-x')
    document.documentElement.style.removeProperty('--theme-y')
  }

  return { startTransition }
}