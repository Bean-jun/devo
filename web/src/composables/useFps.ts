import { ref, onMounted } from 'vue'

export function useFps() {
  const fps = ref(0)

  let frameCount = 0
  let lastTime = performance.now()
  let rafId = 0

  function tick() {
    frameCount++
    const now = performance.now()
    const elapsed = now - lastTime

    if (elapsed >= 1000) {
      fps.value = Math.round((frameCount * 1000) / elapsed)
      frameCount = 0
      lastTime = now
    }

    rafId = requestAnimationFrame(tick)
  }

  onMounted(() => {
    rafId = requestAnimationFrame(tick)
  })

  return { fps }
}