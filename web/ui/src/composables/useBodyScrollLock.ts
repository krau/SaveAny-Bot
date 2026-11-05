let lockCount = 0
let previousOverflow = ''
let previousPaddingRight = ''

// lockBodyScroll disables page scrolling while one or more modals are open.
// The returned function is idempotent, so callers can safely release it from
// either a watcher or component teardown.
export function lockBodyScroll(): () => void {
  const body = document.body
  if (lockCount === 0) {
    previousOverflow = body.style.overflow
    previousPaddingRight = body.style.paddingRight
    const scrollbarWidth = window.innerWidth - document.documentElement.clientWidth
    body.style.overflow = 'hidden'
    if (scrollbarWidth > 0) {
      body.style.paddingRight = `${scrollbarWidth}px`
    }
  }
  lockCount += 1

  let released = false
  return () => {
    if (released) return
    released = true
    lockCount = Math.max(0, lockCount - 1)
    if (lockCount > 0) return
    body.style.overflow = previousOverflow
    body.style.paddingRight = previousPaddingRight
  }
}
