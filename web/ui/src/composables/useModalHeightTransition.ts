import { onUnmounted } from 'vue'
import type { Ref } from 'vue'

const RESIZE_DURATION = 280

export function useModalHeightTransition(modal: Ref<HTMLElement | null>) {
  let timer: number | undefined
  let from = 0
  let version = 0

  function reset() {
    from = 0
    timer = undefined
  }

  function startResize() {
    const el = modal.value
    if (!el) return
    if (timer) {
      window.clearTimeout(timer)
      timer = undefined
    }
    // Freeze the current height before Vue swaps the field set.
    version += 1
    from = el.getBoundingClientRect().height
    el.style.transition = 'none'
    el.style.height = `${from}px`
  }

  function endResize() {
    const el = modal.value
    if (!el || !from) return

    const current = from
    const currentVersion = version
    el.style.height = 'auto'
    const to = el.getBoundingClientRect().height
    el.style.height = `${current}px`
    void el.offsetHeight

    const finish = () => {
      reset()
      if (modal.value === el) {
        el.style.transition = ''
        el.style.height = ''
      }
    }

    if (
      Math.abs(to - current) < 1 ||
      window.matchMedia('(prefers-reduced-motion: reduce)').matches
    ) {
      finish()
      return
    }

    el.style.transition = 'height var(--duration-fast) var(--ease-smooth-out)'
    requestAnimationFrame(() => {
      if (currentVersion === version && modal.value === el) {
        el.style.height = `${to}px`
      }
    })
    timer = window.setTimeout(finish, RESIZE_DURATION)
  }

  onUnmounted(() => {
    if (timer) window.clearTimeout(timer)
  })

  return { startResize, endResize }
}
