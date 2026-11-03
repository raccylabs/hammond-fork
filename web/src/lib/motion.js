import { ref, watch, onBeforeUnmount } from 'vue'

const reduce = () => typeof window !== 'undefined' && window.matchMedia?.('(prefers-reduced-motion: reduce)').matches
const ease = (p) => 1 - Math.pow(1 - p, 3)

/** Returns a ref that eases from its previous value (0 at first) to `getter()` whenever it changes. */
export function useTween(getter, duration = 1000) {
  const out = ref(0)
  let raf
  const run = (to) => {
    cancelAnimationFrame(raf)
    if (reduce() || !isFinite(to)) { out.value = to || 0; return }
    const from = out.value, t0 = performance.now()
    const step = (t) => {
      const p = Math.min(1, (t - t0) / duration)
      out.value = from + (to - from) * ease(p)
      if (p < 1) raf = requestAnimationFrame(step)
    }
    raf = requestAnimationFrame(step)
  }
  watch(getter, run, { immediate: true })
  onBeforeUnmount(() => cancelAnimationFrame(raf))
  return out
}
