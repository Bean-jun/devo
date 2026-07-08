const audio = new Audio('/bg.wav')

export function useAudio() {
  function playCompletedSound(): void {
    audio.currentTime = 0
    audio.play().catch(() => {})
  }

  return {
    playCompletedSound,
  }
}